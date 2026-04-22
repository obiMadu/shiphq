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
	ID        string
	Status    string
	Placement session.PlacementKind
}

type LocalRuntime struct{}

const (
	workerPromptDirName        = ".shiphq"
	workerPromptFileName       = "prompt.md"
	workerPromptRelativePath   = workerPromptDirName + "/" + workerPromptFileName
	workerPromptExcludePattern = "/.shiphq/"
)

func (localRuntime LocalRuntime) Create(project string, workItem workitem.WorkItem, workerPrompt, agentName string, placementKind session.PlacementKind) (Session, error) {
	sessionLabel, err := generateBranchName(workItem)
	if err != nil {
		return Session{}, err
	}

	workerID := fmt.Sprintf("%s-%s", project, sessionLabel)
	existingWorker, err := session.Exists(workerID)
	if err != nil {
		return Session{}, err
	}
	if existingWorker {
		return Session{}, fmt.Errorf("worker %s already exists; attach to it or clean it up before creating another", workerID)
	}

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
		worktreeBaseBranch, err = resolveDefaultBranch()
		if err != nil {
			return Session{}, fmt.Errorf("failed to resolve default branch: %w", err)
		}

		if err := pullLatestBranchChanges(worktreeBaseBranch); err != nil {
			return Session{}, fmt.Errorf("failed to update default branch %s: %w", worktreeBaseBranch, err)
		}
	}

	worktreeSwitchCommand := buildWorktreeSwitchCommand(worktreeSwitchTarget, shouldCreateWorktree, worktreeBaseBranch)
	if output, err := worktreeSwitchCommand.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("wt switch failed: %w\n%s", err, output)
	}

	metadata := session.Metadata{
		SessionID:     workerID,
		Project:       project,
		WorkItem:      workItem,
		PlacementKind: placementKind,
	}

	worktreePath, err := findWorktreePath(worktreeLookupBranch)
	if err != nil {
		metadata.Branch = worktreeLookupBranch
		return Session{}, withCreateRollback(fmt.Errorf("failed to resolve worktree path: %w", err), metadata)
	}
	metadata.WorktreePath = worktreePath

	actualBranch, err := currentWorktreeBranch(worktreePath)
	if err != nil {
		metadata.Branch = worktreeLookupBranch
		return Session{}, withCreateRollback(fmt.Errorf("failed to resolve worktree branch: %w", err), metadata)
	}
	metadata.Branch = actualBranch

	if err := ignoreWorktreePattern(worktreePath, workerPromptExcludePattern); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to ignore worker prompt directory: %w", err), metadata)
	}

	if err := writeWorkerPromptFile(worktreePath, workerPrompt); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to write worker prompt file: %w", err), metadata)
	}

	bootstrapPrompt := buildBootstrapPrompt(workerPromptRelativePath)
	agentCommand := configuredAgent.BuildCommand(bootstrapPrompt)
	sessionCommand, err := buildSessionCommand(agentCommand)
	if err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to build session shell command: %w", err), metadata)
	}

	launchInfo, err := launchTmuxTarget(workerID, sessionLabel, worktreePath, sessionCommand, placementKind)
	if err != nil {
		return Session{}, withCreateRollback(err, metadata)
	}
	metadata.TmuxSessionID = launchInfo.SessionID
	metadata.TmuxSessionName = launchInfo.SessionName
	metadata.TmuxWindowID = launchInfo.WindowID
	metadata.TmuxWindowName = launchInfo.WindowName
	metadata.PlacementKind = launchInfo.PlacementKind

	if err := session.Save(metadata); err != nil {
		return Session{}, withCreateRollback(fmt.Errorf("failed to save session metadata: %w", err), metadata)
	}

	return Session{
		ID:        workerID,
		Status:    "running",
		Placement: metadata.EffectivePlacementKind(),
	}, nil
}

func (localRuntime LocalRuntime) List(project string) ([]Session, error) {
	metadataItems, err := session.List()
	if err != nil {
		return nil, err
	}

	tmuxState, err := inspectTmuxState()
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(metadataItems))
	for _, metadata := range metadataItems {
		if metadata.Project == project {
			sessions = append(sessions, Session{
				ID:        metadata.SessionID,
				Status:    statusForMetadata(metadata, tmuxState),
				Placement: metadata.EffectivePlacementKind(),
			})
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

	tmuxState, err := inspectTmuxState()
	if err != nil {
		return nil, err
	}

	sessions := make([]Session, 0, len(metadataItems))
	for _, metadata := range metadataItems {
		sessions = append(sessions, Session{
			ID:        metadata.SessionID,
			Status:    statusForMetadata(metadata, tmuxState),
			Placement: metadata.EffectivePlacementKind(),
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ID < sessions[j].ID
	})

	return sessions, nil
}

func isNoTmuxServer(output []byte) bool {
	lowerOutput := strings.ToLower(string(output))
	return strings.Contains(lowerOutput, "no server running") || strings.Contains(lowerOutput, "failed to connect to server")
}

func statusForMetadata(metadata session.Metadata, tmuxState tmuxState) string {
	if tmuxPlacementExists(metadata, tmuxState) {
		return "running"
	}

	return "stopped"
}

func tmuxPlacementExists(metadata session.Metadata, tmuxState tmuxState) bool {
	if metadata.EffectivePlacementKind() == session.PlacementKindWindow {
		_, exists := tmuxState.windowIDs[metadata.EffectiveTmuxWindowTarget()]
		return exists
	}

	if _, exists := tmuxState.sessionIDs[metadata.EffectiveTmuxSessionTarget()]; exists {
		return true
	}

	_, exists := tmuxState.sessionNames[metadata.EffectiveTmuxSessionTarget()]
	return exists
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
	excludePath, err := findGitPath(worktreePath, "info/exclude")
	if err != nil {
		return err
	}

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

func findGitPath(worktreePath, gitPath string) (string, error) {
	gitPathCommand := exec.Command("git", "rev-parse", "--path-format=absolute", "--git-path", gitPath)
	gitPathCommand.Dir = worktreePath
	output, err := gitPathCommand.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git rev-parse --git-path %s failed: %w\n%s", gitPath, err, output)
	}

	resolvedPath := strings.TrimSpace(string(output))
	if resolvedPath == "" {
		return "", fmt.Errorf("git path %s cannot be empty", gitPath)
	}

	return filepath.Clean(resolvedPath), nil
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

func resolveDefaultBranch() (string, error) {
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

func pullLatestBranchChanges(branch string) error {
	branchWorktreePath, err := findWorktreePath(branch)
	if err != nil {
		return fmt.Errorf("failed to find worktree for branch %s: %w", branch, err)
	}

	pullCommand := exec.Command("git", "pull", "--ff-only")
	pullCommand.Dir = branchWorktreePath
	output, err := pullCommand.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull --ff-only failed: %w\n%s", err, output)
	}

	return nil
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
	metadata, err := session.Load(sessionID)
	if err != nil {
		return err
	}

	return attachTmuxPlacement(metadata)
}

func (localRuntime LocalRuntime) Cleanup(sessionID string, force bool) error {
	metadata, err := session.Load(sessionID)
	if err != nil {
		return err
	}

	if err := ensureCleanupPreflight(sessionID, metadata, force); err != nil {
		return err
	}

	worktreeTarget, err := resolveCleanupWorktreeTarget(metadata)
	if err != nil {
		return err
	}

	var cleanupErrors []string

	if err := killTmuxPlacement(metadata); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}
	if err := cleanupWorktree(worktreeTarget, force); err != nil {
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

func ensureCleanupPreflight(sessionID string, metadata session.Metadata, force bool) error {
	if force {
		return nil
	}

	worktreePath := strings.TrimSpace(metadata.WorktreePath)
	if worktreePath == "" {
		return nil
	}

	worktreeInfo, err := os.Stat(worktreePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("failed to inspect worktree path %s: %w", worktreePath, err)
	}
	if !worktreeInfo.IsDir() {
		return fmt.Errorf("worktree path %s is not a directory", worktreePath)
	}

	dirtyWorktree, err := worktreeHasUncommittedChanges(worktreePath)
	if err != nil {
		return err
	}
	if dirtyWorktree {
		return fmt.Errorf("cleanup aborted: worker %s has uncommitted changes\nworker remains running and tracked; commit or stash changes first, or rerun with `shiphq cleanup --id %s --force`", sessionID, sessionID)
	}

	return nil
}

func resolveCleanupWorktreeTarget(metadata session.Metadata) (string, error) {
	worktreeTarget := strings.TrimSpace(metadata.WorktreePath)
	if worktreeTarget != "" {
		return worktreeTarget, nil
	}

	worktreeTarget = strings.TrimSpace(metadata.Branch)
	if worktreeTarget != "" {
		return worktreeTarget, nil
	}

	return "", fmt.Errorf("session metadata is missing both worktree path and branch")
}

func worktreeHasUncommittedChanges(worktreePath string) (bool, error) {
	statusCommand := exec.Command("git", "status", "--short")
	statusCommand.Dir = worktreePath
	output, err := statusCommand.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("git status --short failed in %s: %w\n%s", worktreePath, err, output)
	}

	return strings.TrimSpace(string(output)) != "", nil
}

func (localRuntime LocalRuntime) Promote(sessionID string) (Session, error) {
	metadata, err := session.Load(sessionID)
	if err != nil {
		return Session{}, err
	}

	if metadata.EffectivePlacementKind() == session.PlacementKindSession {
		return Session{}, fmt.Errorf("worker %s is already in a dedicated tmux session", sessionID)
	}

	windowTarget := metadata.EffectiveTmuxWindowTarget()
	if windowTarget == "" {
		return Session{}, fmt.Errorf("worker %s is missing its tmux window target", sessionID)
	}
	if strings.TrimSpace(metadata.WorktreePath) == "" {
		return Session{}, fmt.Errorf("worker %s is missing its worktree path", sessionID)
	}

	tmuxState, err := inspectTmuxState()
	if err != nil {
		return Session{}, err
	}
	if !tmuxPlacementExists(metadata, tmuxState) {
		return Session{}, fmt.Errorf("worker %s is not currently running in tmux", sessionID)
	}

	launchInfo, err := promoteTmuxWindowToSession(metadata.SessionID, metadata.WorktreePath, windowTarget, metadata.TmuxWindowName)
	if err != nil {
		return Session{}, err
	}

	metadata.PlacementKind = launchInfo.PlacementKind
	metadata.TmuxSessionID = launchInfo.SessionID
	metadata.TmuxSessionName = launchInfo.SessionName
	metadata.TmuxWindowID = launchInfo.WindowID
	metadata.TmuxWindowName = launchInfo.WindowName

	if err := session.Save(metadata); err != nil {
		return Session{}, fmt.Errorf("failed to save promoted worker metadata: %w", err)
	}

	return Session{
		ID:        metadata.SessionID,
		Status:    "running",
		Placement: metadata.EffectivePlacementKind(),
	}, nil
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

func withCreateRollback(createErr error, metadata session.Metadata) error {
	var rollbackErrors []string

	if metadata.SessionID != "" {
		if err := killTmuxPlacement(metadata); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}

		if err := session.Delete(metadata.SessionID); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	worktreeTarget := strings.TrimSpace(metadata.WorktreePath)
	if worktreeTarget == "" {
		worktreeTarget = strings.TrimSpace(metadata.Branch)
	}

	if worktreeTarget != "" {
		if err := cleanupWorktree(worktreeTarget, false); err != nil {
			rollbackErrors = append(rollbackErrors, err.Error())
		}
	}

	if len(rollbackErrors) == 0 {
		return createErr
	}

	return fmt.Errorf("%v\nrollback errors:\n%s", createErr, strings.Join(rollbackErrors, "\n"))
}
