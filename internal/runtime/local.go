package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

const (
	workerPromptDirName        = ".shiphq"
	workerPromptFileName       = "prompt.md"
	workerPromptRelativePath   = workerPromptDirName + "/" + workerPromptFileName
	workerPromptExcludePattern = "/.shiphq/"
)

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

	worktreeBaseBranch := ""
	if shouldCreateWorktree {
		worktreeBaseBranch, err = refreshDefaultBranch()
		if err != nil {
			return Session{}, fmt.Errorf("failed to resolve default branch: %w", err)
		}
	}

	worktreeSwitchCommand := buildWorktreeSwitchCommand(worktreeSwitchTarget, shouldCreateWorktree, worktreeBaseBranch)
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

	if err := ignoreWorktreePattern(worktreePath, workerPromptExcludePattern); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to ignore worker prompt directory: %w", err), sessionID, worktreePath)
	}

	if err := writeWorkerPromptFile(worktreePath, workerPrompt); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to write worker prompt file: %w", err), sessionID, worktreePath)
	}

	bootstrapPrompt := buildBootstrapPrompt(workerPromptRelativePath)
	agentCommand := configuredAgent.BuildCommand(bootstrapPrompt)
	sessionCommand, err := buildSessionCommand(agentCommand)
	if err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to build session shell command: %w", err), sessionID, worktreePath)
	}

	sessionCreateCommand := exec.Command("tmux", "new-session", "-d", "-s", sessionID, "-n", "agent", "-c", worktreePath, sessionCommand)
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
	runningSessionIDs, err := listTmuxSessionIDs()
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(runningSessionIDs))
	for sessionID := range runningSessionIDs {
		if strings.HasPrefix(sessionID, project+"-") {
			sessions = append(sessions, Session{ID: sessionID, Status: "running"})
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	return sessions, nil
}

func (localRuntime LocalRuntime) ListAll() ([]Session, error) {
	metadataItems, err := session.List()
	if err != nil {
		return nil, err
	}

	runningSessionIDs, err := listTmuxSessionIDs()
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(metadataItems))
	for _, metadata := range metadataItems {
		status := "stopped"
		if _, ok := runningSessionIDs[metadata.SessionID]; ok {
			status = "running"
		}

		sessions = append(sessions, Session{ID: metadata.SessionID, Status: status})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	return sessions, nil
}

func listTmuxSessionIDs() (map[string]struct{}, error) {
	listCommand := exec.Command("tmux", "list-sessions", "-F", "#S")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		if isNoTmuxServer(output) {
			return map[string]struct{}{}, nil
		}

		return nil, fmt.Errorf("tmux list failed: %w\n%s", err, output)
	}

	sessionIDs := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		sessionID := strings.TrimSpace(line)
		if sessionID == "" {
			continue
		}

		sessionIDs[sessionID] = struct{}{}
	}

	return sessionIDs, nil
}

func isNoTmuxServer(output []byte) bool {
	lowerOutput := strings.ToLower(string(output))
	return strings.Contains(lowerOutput, "no server running") || strings.Contains(lowerOutput, "failed to connect to server")
}

func writeWorkerPromptFile(worktreePath, prompt string) error {
	promptPath := filepath.Join(worktreePath, workerPromptRelativePath)
	promptContents := prompt
	if !strings.HasSuffix(promptContents, "\n") {
		promptContents += "\n"
	}

	if err := os.MkdirAll(filepath.Dir(promptPath), 0755); err != nil {
		return fmt.Errorf("failed to create prompt directory for %s: %w", promptPath, err)
	}

	if err := os.WriteFile(promptPath, []byte(promptContents), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", promptPath, err)
	}

	return nil
}

func ignoreWorktreePattern(worktreePath, pattern string) error {
	gitDir, err := findGitDir(worktreePath)
	if err != nil {
		return err
	}

	excludePath := filepath.Join(gitDir, "info", "exclude")
	if err := os.MkdirAll(filepath.Dir(excludePath), 0755); err != nil {
		return fmt.Errorf("failed to prepare exclude file directory: %w", err)
	}

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

func buildSessionCommand(agentCommand string) (string, error) {
	shellPath, err := resolveSessionShell()
	if err != nil {
		return "", err
	}

	commandArgs, resumeArgs := shellModes(shellPath)
	resumeShellCommand := buildShellCommand(shellPath, resumeArgs...)
	script := agentCommand + "; exec " + resumeShellCommand

	return buildShellCommand(shellPath, append(commandArgs, script)...), nil
}

func resolveSessionShell() (string, error) {
	candidates := []string{strings.TrimSpace(os.Getenv("SHELL")), "bash", "sh"}
	seen := make(map[string]struct{}, len(candidates))

	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}

		resolvedPath, err := resolveShellPath(candidate)
		if err == nil {
			return resolvedPath, nil
		}
	}

	return "", fmt.Errorf("could not resolve shell from $SHELL, bash, or sh")
}

func resolveShellPath(candidate string) (string, error) {
	if strings.Contains(candidate, "/") {
		info, err := os.Stat(candidate)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("shell path %s is a directory", candidate)
		}
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("shell path %s is not executable", candidate)
		}
		return candidate, nil
	}

	return exec.LookPath(candidate)
}

func shellModes(shellPath string) ([]string, []string) {
	switch filepath.Base(shellPath) {
	case "sh", "dash":
		return []string{"-i", "-c"}, []string{"-i"}
	default:
		return []string{"-i", "-l", "-c"}, []string{"-i", "-l"}
	}
}

func buildShellCommand(command string, args ...string) string {
	parts := []string{shellQuote(command)}
	for _, arg := range args {
		parts = append(parts, shellQuote(arg))
	}
	return strings.Join(parts, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
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

func refreshDefaultBranch() (string, error) {
	clearCommand := exec.Command("wt", "config", "state", "default-branch", "clear")
	if output, err := clearCommand.CombinedOutput(); err != nil {
		return "", fmt.Errorf("wt config state default-branch clear failed: %w\n%s", err, output)
	}

	defaultBranchCommand := exec.Command("wt", "config", "state", "default-branch")
	output, err := defaultBranchCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("wt config state default-branch failed: %w\n%s", err, output)
	}

	defaultBranch := strings.TrimSpace(string(output))
	if defaultBranch == "" {
		return "", fmt.Errorf("wt default branch cannot be empty")
	}

	return defaultBranch, nil
}

func buildWorktreeSwitchCommand(target string, shouldCreate bool, baseBranch string) *exec.Cmd {
	args := []string{"switch"}
	if shouldCreate {
		args = append(args, "--create", "--base", baseBranch)
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

func (localRuntime LocalRuntime) Cleanup(sessionID string, force bool) error {
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
	} else if err := cleanupWorktree(worktreeTarget, force); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}

	if len(cleanupErrors) > 0 {
		if !force {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("Cleanup failed. Consider retrying with `shiphq cleanup --id %s --force`.", sessionID))
		}
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

func cleanupWorktree(worktreeTarget string, force bool) error {
	commandArgs := []string{"remove"}
	commandLabel := "wt remove"
	if force {
		commandArgs = append(commandArgs, "--force")
		commandLabel = "wt remove --force"
	}
	commandArgs = append(commandArgs, worktreeTarget)

	worktreeRemoveCommand := exec.Command("wt", commandArgs...)
	output, err := worktreeRemoveCommand.CombinedOutput()
	if err == nil || isMissingWorktree(output) {
		return nil
	}

	return fmt.Errorf("%s failed: %v\n%s", commandLabel, err, output)
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
		if err := cleanupWorktree(branch, false); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if len(rollbackErrors) == 0 {
		return createErr
	}

	return fmt.Errorf("%v\nrollback errors:\n%s", createErr, strings.Join(rollbackErrors, "\n"))
}
