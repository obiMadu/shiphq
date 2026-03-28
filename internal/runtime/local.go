package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
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

func (localRuntime LocalRuntime) Create(project string, workItem workitem.WorkItem, workerPrompt, agentName string) (Session, error) {
	branch, err := generateBranchName(workItem)
	if err != nil {
		return Session{}, err
	}

	sessionID := fmt.Sprintf("%s-%s", project, branch)

	configuredAgent, err := agent.Get(agentName)
	if err != nil {
		return Session{}, err
	}

	worktreeCreateCommand := exec.Command("wt", "switch", "--create", branch)
	if output, err := worktreeCreateCommand.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("wt switch failed: %w\n%s", err, output)
	}

	worktreePath, err := findWorktreePath(branch)
	if err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to resolve worktree path: %w", err), sessionID, branch)
	}

	sessionCreateCommand := exec.Command("tmux", "new-session", "-d", "-s", sessionID, "-c", worktreePath)
	if output, err := sessionCreateCommand.CombinedOutput(); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("tmux create failed: %w\n%s", err, output), sessionID, branch)
	}

	agentCommand := configuredAgent.BuildCommand(workerPrompt)
	workerWindowCommand := exec.Command("tmux", "new-window", "-t", sessionID, "-n", "agent", "-c", worktreePath, agentCommand)
	if output, err := workerWindowCommand.CombinedOutput(); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("tmux window failed: %w\n%s", err, output), sessionID, branch)
	}

	metadata := session.Metadata{
		SessionID:    sessionID,
		Project:      project,
		Branch:       branch,
		WorktreePath: worktreePath,
		WorkItem:     workItem,
	}
	if err := session.Save(metadata); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to save session metadata: %w", err), sessionID, branch)
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

	sessionKillCommand := exec.Command("tmux", "kill-session", "-t", sessionID)
	if output, err := sessionKillCommand.CombinedOutput(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("tmux kill failed: %v\n%s", err, output))
	}

	worktreeRemoveCommand := exec.Command("wt", "remove", metadata.Branch)
	if output, err := worktreeRemoveCommand.CombinedOutput(); err != nil {
		cleanupErrors = append(cleanupErrors, fmt.Sprintf("wt remove failed: %v\n%s", err, output))
	}

	if err := session.Delete(sessionID); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}

	if len(cleanupErrors) > 0 {
		return errors.New(strings.Join(cleanupErrors, "\n"))
	}

	return nil
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
		sessionKillCommand := exec.Command("tmux", "kill-session", "-t", sessionID)
		if output, err := sessionKillCommand.CombinedOutput(); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("tmux rollback failed: %v\n%s", err, output))
		}

		if err := session.Delete(sessionID); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if branch != "" {
		worktreeRemoveCommand := exec.Command("wt", "remove", branch)
		if output, err := worktreeRemoveCommand.CombinedOutput(); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Sprintf("wt rollback failed: %v\n%s", err, output))
		}
	}

	if len(rollbackErrors) == 0 {
		return createErr
	}

	return fmt.Errorf("%v\nrollback errors:\n%s", createErr, strings.Join(rollbackErrors, "\n"))
}
