package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/obiMadu/shiphq/internal/agent"
	"github.com/obiMadu/shiphq/internal/session"
	"github.com/obiMadu/shiphq/internal/workitem"
)

type Session struct {
	ID     string
	Status string
}

type LocalRuntime struct{}

const workerPromptFileName = "prompt.md"

func (localRuntime LocalRuntime) Create(project string, workItem workitem.WorkItem, workerPrompt, agentName string) (Session, error) {
	sessionLabel, err := generateBranchName(workItem)
	if err != nil {
		return Session{}, err
	}

	sessionID := fmt.Sprintf("%s-%s", project, sessionLabel)

	configuredAgent, err := agent.Get(agentName)
	if err != nil {
		return Session{}, err
	}

	worktreeSwitchTarget, worktreeLookupBranch, shouldCreateWorktree, err := resolveWorktreeSwitch(workItem, sessionLabel)
	if err != nil {
		return Session{}, err
	}

	worktreeSwitchCommand := buildWorktreeSwitchCommand(worktreeSwitchTarget, shouldCreateWorktree)
	if output, err := worktreeSwitchCommand.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("wt switch failed: %w\n%s", err, output)
	}

	worktreePath, err := findWorktreePath(worktreeLookupBranch)
	if err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to resolve worktree path: %w", err), sessionID, worktreeLookupBranch)
	}

	actualBranch, err := currentWorktreeBranch(worktreePath)
	if err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to resolve worktree branch: %w", err), sessionID, worktreePath)
	}

	if err := ignoreWorktreeFile(worktreePath, workerPromptFileName); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to ignore worker prompt file: %w", err), sessionID, worktreePath)
	}

	if err := writeWorkerPromptFile(worktreePath, workerPrompt); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to write worker prompt file: %w", err), sessionID, worktreePath)
	}

	bootstrapPrompt := buildBootstrapPrompt(workerPromptFileName)
	agentCommand := configuredAgent.BuildCommand(bootstrapPrompt)

	sessionCreateCommand := exec.Command("tmux", "new-session", "-d", "-s", sessionID, "-n", "agent", "-c", worktreePath, agentCommand)
	if output, err := sessionCreateCommand.CombinedOutput(); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("tmux create failed: %w\n%s", err, output), sessionID, worktreePath)
	}

	metadata := session.Metadata{
		SessionID:    sessionID,
		Project:      project,
		Branch:       actualBranch,
		WorktreePath: worktreePath,
		WorkItem:     workItem,
	}
	if err := session.Save(metadata); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to save session metadata: %w", err), sessionID, worktreePath)
	}

	return Session{
		ID:     sessionID,
		Status: "running",
	}, nil
}

func (localRuntime LocalRuntime) List(project string) ([]Session, error) {
	listCommand := exec.Command("tmux", "list-sessions", "-F", "#S")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tmux list failed: %w\n%s", err, output)
	}

	var sessions []Session
	for _, line := range strings.Split(string(output), "\n") {
		sessionID := strings.TrimSpace(line)
		if sessionID == "" {
			continue
		}
		if strings.HasPrefix(sessionID, project+"-") {
			sessions = append(sessions, Session{ID: sessionID, Status: "running"})
		}
	}

	return sessions, nil
}

func writeWorkerPromptFile(worktreePath, prompt string) error {
	promptPath := filepath.Join(worktreePath, workerPromptFileName)
	promptContents := prompt
	if !strings.HasSuffix(promptContents, "\n") {
		promptContents += "\n"
	}

	if err := os.WriteFile(promptPath, []byte(promptContents), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", promptPath, err)
	}

	return nil
}

func ignoreWorktreeFile(worktreePath, fileName string) error {
	gitDir, err := findGitDir(worktreePath)
	if err != nil {
		return err
	}

	excludePath := filepath.Join(gitDir, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return fmt.Errorf("failed to prepare exclude file directory: %w", err)
	}

	pattern := "/" + fileName
	existingContents, err := os.ReadFile(excludePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read %s: %w", excludePath, err)
	}

	for _, line := range strings.Split(string(existingContents), "\n") {
		if strings.TrimSpace(line) == pattern {
			return nil
		}
	}

	updatedContents := string(existingContents)
	if updatedContents != "" && !strings.HasSuffix(updatedContents, "\n") {
		updatedContents += "\n"
	}
	updatedContents += pattern + "\n"

	if err := os.WriteFile(excludePath, []byte(updatedContents), 0644); err != nil {
		return fmt.Errorf("failed to update %s: %w", excludePath, err)
	}

	return nil
}

func findGitDir(worktreePath string) (string, error) {
	gitDirCommand := exec.Command("git", "rev-parse", "--git-dir")
	gitDirCommand.Dir = worktreePath
	output, err := gitDirCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-dir failed: %w\n%s", err, output)
	}

	gitDir := strings.TrimSpace(string(output))
	if gitDir == "" {
		return "", fmt.Errorf("git directory cannot be empty")
	}

	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(worktreePath, gitDir)
	}

	return filepath.Clean(gitDir), nil
}

func buildBootstrapPrompt(fileName string) string {
	return fmt.Sprintf("Read ./%s and use it as the full task brief.", fileName)
}

func resolveWorktreeSwitch(workItem workitem.WorkItem, defaultBranch string) (string, string, bool, error) {
	if workItem.Source.System == "github" && workItem.Source.Kind == "pr" {
		prReference := strings.TrimSpace(workItem.Source.Reference)
		if prReference == "" {
			return "", "", false, fmt.Errorf("GitHub PR reference cannot be empty")
		}

		targetBranch := strings.TrimSpace(workItem.TargetBranch)
		if targetBranch == "" {
			return "", "", false, fmt.Errorf("GitHub PR %s is missing a head branch", prReference)
		}

		return "pr:" + prReference, targetBranch, false, nil
	}

	return defaultBranch, defaultBranch, true, nil
}

func buildWorktreeSwitchCommand(target string, shouldCreate bool) *exec.Cmd {
	args := []string{"switch"}
	if shouldCreate {
		args = append(args, "--create")
	}
	args = append(args, target)
	return exec.Command("wt", args...)
}

func currentWorktreeBranch(worktreePath string) (string, error) {
	branchCommand := exec.Command("git", "branch", "--show-current")
	branchCommand.Dir = worktreePath
	output, err := branchCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git branch --show-current failed: %w\n%s", err, output)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("current branch cannot be empty")
	}

	return branch, nil
}

func (localRuntime LocalRuntime) Attach(sessionID string) error {
	attachCommand := exec.Command("tmux", "attach", "-t", sessionID)
	attachCommand.Stdin = os.Stdin
	attachCommand.Stdout = os.Stdout
	attachCommand.Stderr = os.Stderr
	return attachCommand.Run()
}

func (localRuntime LocalRuntime) Cleanup(sessionID string) error {
	metadata, err := session.Load(sessionID)
	if err != nil {
		return err
	}

	var cleanupErrors []string

	if err := cleanupTmuxSession(sessionID); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}

	worktreeTarget := metadata.WorktreePath
	if worktreeTarget == "" {
		worktreeTarget = metadata.Branch
	}

	if strings.TrimSpace(worktreeTarget) == "" {
		cleanupErrors = append(cleanupErrors, "session metadata is missing both worktree path and branch")
	} else if err := cleanupWorktree(worktreeTarget); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}

	if len(cleanupErrors) > 0 {
		return errors.New(strings.Join(cleanupErrors, "\n"))
	}

	if err := session.Delete(sessionID); err != nil {
		return err
	}

	return nil
}

func cleanupTmuxSession(sessionID string) error {
	sessionKillCommand := exec.Command("tmux", "kill-session", "-t", sessionID)
	output, err := sessionKillCommand.CombinedOutput()
	if err == nil || isMissingTmuxSession(output) {
		return nil
	}

	return fmt.Errorf("tmux kill failed: %v\n%s", err, output)
}

func cleanupWorktree(worktreeTarget string) error {
	worktreeRemoveCommand := exec.Command("wt", "remove", worktreeTarget)
	output, err := worktreeRemoveCommand.CombinedOutput()
	if err == nil || isMissingWorktree(output) {
		return nil
	}

	return fmt.Errorf("wt remove failed: %v\n%s", err, output)
}

func isMissingTmuxSession(output []byte) bool {
	return strings.Contains(strings.ToLower(string(output)), "can't find session")
}

func isMissingWorktree(output []byte) bool {
	lowerOutput := strings.ToLower(string(output))
	return strings.Contains(lowerOutput, "no branch named") || strings.Contains(lowerOutput, "no worktree")
}

func generateBranchName(workItem workitem.WorkItem) (string, error) {
	if workItem.Identifier == "" {
		return "", fmt.Errorf("work item identifier cannot be empty")
	}

	if workItem.Source.System == workItem.Source.Kind {
		return fmt.Sprintf("%s-%s", workItem.Source.System, workItem.Identifier), nil
	}

	return fmt.Sprintf("%s-%s-%s", workItem.Source.System, workItem.Source.Kind, workItem.Identifier), nil
}

func findWorktreePath(branch string) (string, error) {
	listCommand := exec.Command("git", "worktree", "list", "--porcelain")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree list failed: %w\n%s", err, output)
	}

	wantedBranchRef := "refs/heads/" + branch
	for _, block := range strings.Split(strings.TrimSpace(string(output)), "\n\n") {
		if block == "" {
			continue
		}

		var worktreePath string
		var branchRef string
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				worktreePath = strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			case strings.HasPrefix(line, "branch "):
				branchRef = strings.TrimSpace(strings.TrimPrefix(line, "branch "))
			}
		}

		if branchRef == wantedBranchRef && worktreePath != "" {
			return worktreePath, nil
		}
	}

	return "", fmt.Errorf("worktree not found for branch %s", branch)
}

func withCreateRollback(createErr error, sessionID, branch string) error {
	var rollbackErrors []string

	if sessionID != "" {
		if err := cleanupTmuxSession(sessionID); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}

		if err := session.Delete(sessionID); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if branch != "" {
		if err := cleanupWorktree(branch); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if len(rollbackErrors) == 0 {
		return createErr
	}

	return fmt.Errorf("%v\nrollback errors:\n%s", createErr, strings.Join(rollbackErrors, "\n"))
}
