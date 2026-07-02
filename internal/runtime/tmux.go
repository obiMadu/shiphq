package runtime

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/obiMadu/wtmag/internal/session"
)

const tmuxLaunchFormat = "#{session_id}\t#{session_name}\t#{window_id}\t#{window_name}\t#{pane_id}"

type tmuxLaunchInfo struct {
	PlacementKind session.PlacementKind
	SessionID     string
	SessionName   string
	WindowID      string
	WindowName    string
	PaneID        string
}

type tmuxState struct {
	sessionIDs   map[string]struct{}
	sessionNames map[string]struct{}
	windowIDs    map[string]struct{}
}

type tmuxSessionContext struct {
	SessionID   string
	SessionName string
}

func launchTmuxTarget(logicalWorkerID, windowName, worktreePath, sessionCommand string, placementKind session.PlacementKind) (tmuxLaunchInfo, error) {
	if placementKind == session.PlacementKindWindow {
		parentSession, err := currentTmuxSession()
		if err != nil {
			return tmuxLaunchInfo{}, err
		}

		return createWorkerWindow(parentSession, windowName, worktreePath, sessionCommand)
	}

	return createWorkerSession(logicalWorkerID, worktreePath, sessionCommand)
}

func createWorkerSession(logicalWorkerID, worktreePath, sessionCommand string) (tmuxLaunchInfo, error) {
	createCommand := exec.Command(
		"tmux",
		"new-session",
		"-d",
		"-P",
		"-F",
		tmuxLaunchFormat,
		"-s",
		logicalWorkerID,
		"-n",
		"agent",
		"-c",
		worktreePath,
		sessionCommand,
	)
	output, err := createCommand.CombinedOutput()
	if err != nil {
		return tmuxLaunchInfo{}, fmt.Errorf("tmux new-session failed: %w\n%s", err, output)
	}

	launchInfo, err := parseTmuxLaunchOutput(output)
	if err != nil {
		_ = killTmuxSession(logicalWorkerID)
		return tmuxLaunchInfo{}, fmt.Errorf("failed to parse tmux session output: %w", err)
	}
	launchInfo.PlacementKind = session.PlacementKindSession

	return launchInfo, nil
}

func createWorkerWindow(parentSession tmuxSessionContext, windowName, worktreePath, sessionCommand string) (tmuxLaunchInfo, error) {
	targetSession := strings.TrimSpace(parentSession.SessionID)
	if targetSession == "" {
		targetSession = strings.TrimSpace(parentSession.SessionName)
	}
	if targetSession == "" {
		return tmuxLaunchInfo{}, fmt.Errorf("parent tmux session cannot be empty")
	}

	createCommand := exec.Command(
		"tmux",
		"new-window",
		"-d",
		"-P",
		"-F",
		tmuxLaunchFormat,
		"-t",
		targetSession+":",
		"-n",
		windowName,
		"-c",
		worktreePath,
		sessionCommand,
	)
	output, err := createCommand.CombinedOutput()
	if err != nil {
		return tmuxLaunchInfo{}, fmt.Errorf("tmux new-window failed: %w\n%s", err, output)
	}

	launchInfo, err := parseTmuxLaunchOutput(output)
	if err != nil {
		return tmuxLaunchInfo{}, fmt.Errorf("failed to parse tmux window output: %w", err)
	}
	launchInfo.PlacementKind = session.PlacementKindWindow

	return launchInfo, nil
}

func currentTmuxSession() (tmuxSessionContext, error) {
	if !isInsideTmux() {
		return tmuxSessionContext{}, fmt.Errorf("window placement requires running inside tmux; use -s or --launch session outside tmux")
	}

	contextCommand := exec.Command("tmux", "display-message", "-p", "#{session_id}\t#{session_name}")
	output, err := contextCommand.CombinedOutput()
	if err != nil {
		return tmuxSessionContext{}, fmt.Errorf("failed to resolve current tmux session: %w\n%s", err, output)
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "\t", 2)
	if len(parts) != 2 {
		return tmuxSessionContext{}, fmt.Errorf("unexpected tmux current-session output: %s", strings.TrimSpace(string(output)))
	}

	return tmuxSessionContext{
		SessionID:   strings.TrimSpace(parts[0]),
		SessionName: strings.TrimSpace(parts[1]),
	}, nil
}

func promoteTmuxWindowToSession(logicalWorkerID, worktreePath, windowTarget, windowName string) (tmuxLaunchInfo, error) {
	bootstrapCommand := exec.Command(
		"tmux",
		"new-session",
		"-d",
		"-P",
		"-F",
		tmuxLaunchFormat,
		"-s",
		logicalWorkerID,
		"-n",
		"bootstrap",
		"-c",
		worktreePath,
	)
	output, err := bootstrapCommand.CombinedOutput()
	if err != nil {
		return tmuxLaunchInfo{}, fmt.Errorf("tmux new-session failed during promotion: %w\n%s", err, output)
	}

	launchInfo, err := parseTmuxLaunchOutput(output)
	if err != nil {
		_ = killTmuxSession(logicalWorkerID)
		return tmuxLaunchInfo{}, fmt.Errorf("failed to parse tmux promotion session output: %w", err)
	}

	moveCommand := exec.Command("tmux", "move-window", "-k", "-s", windowTarget, "-t", launchInfo.SessionID+":0")
	moveOutput, moveErr := moveCommand.CombinedOutput()
	if moveErr != nil {
		_ = killTmuxSession(launchInfo.SessionID)
		return tmuxLaunchInfo{}, fmt.Errorf("tmux move-window failed during promotion: %w\n%s", moveErr, moveOutput)
	}

	launchInfo.PlacementKind = session.PlacementKindSession
	launchInfo.WindowID = strings.TrimSpace(windowTarget)
	launchInfo.WindowName = strings.TrimSpace(windowName)

	return launchInfo, nil
}

func attachTmuxPlacement(metadata session.Metadata) error {
	sessionTarget := metadata.EffectiveTmuxSessionTarget()
	if metadata.EffectivePlacementKind() == session.PlacementKindWindow {
		return attachTmuxWindow(sessionTarget, metadata.EffectiveTmuxWindowTarget())
	}

	return attachTmuxSession(sessionTarget)
}

func attachTmuxSession(sessionTarget string) error {
	if strings.TrimSpace(sessionTarget) == "" {
		return fmt.Errorf("tmux session target cannot be empty")
	}

	if isInsideTmux() {
		return runTmuxInteractiveCommand("switch-client", "-t", sessionTarget)
	}

	return runTmuxInteractiveCommand("attach-session", "-t", sessionTarget)
}

func attachTmuxWindow(sessionTarget, windowTarget string) error {
	if strings.TrimSpace(sessionTarget) == "" {
		return fmt.Errorf("tmux session target cannot be empty")
	}
	if strings.TrimSpace(windowTarget) == "" {
		return fmt.Errorf("tmux window target cannot be empty")
	}

	if isInsideTmux() {
		if err := runTmuxInteractiveCommand("switch-client", "-t", sessionTarget); err != nil {
			return err
		}

		return runTmuxInteractiveCommand("select-window", "-t", windowTarget)
	}

	if err := runTmuxInteractiveCommand("select-window", "-t", windowTarget); err != nil {
		return err
	}

	return runTmuxInteractiveCommand("attach-session", "-t", sessionTarget)
}

func configureTmuxPaneLogPipe(metadata session.Metadata, logOptions LogOptions) error {
	if !logOptions.Enabled() {
		return nil
	}

	if err := prepareLogDestination(logOptions); err != nil {
		return err
	}

	return enableTmuxPanePipe(metadata.EffectiveTmuxPaneTarget(), buildLogPipeCommand(logOptions.Path))
}

func prepareLogDestination(logOptions LogOptions) error {
	logPath := strings.TrimSpace(logOptions.Path)
	if logPath == "" {
		return fmt.Errorf("log path cannot be empty")
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return fmt.Errorf("failed to create log directory for %s: %w", logPath, err)
	}

	if logOptions.MaxBytes > 0 {
		if err := rotateLogFileIfNeeded(logPath, logOptions.MaxBytes); err != nil {
			return err
		}
	}

	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to prepare log file %s: %w", logPath, err)
	}

	return logFile.Close()
}

func rotateLogFileIfNeeded(logPath string, maxBytes int64) error {
	logFileInfo, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to inspect log file %s: %w", logPath, err)
	}
	if logFileInfo.IsDir() {
		return fmt.Errorf("log path %s is a directory", logPath)
	}
	if logFileInfo.Size() < maxBytes {
		return nil
	}

	rotatedPath := logPath + ".1"
	if err := os.Rename(logPath, rotatedPath); err != nil {
		return fmt.Errorf("failed to rotate log file %s to %s: %w", logPath, rotatedPath, err)
	}

	return nil
}

func buildLogPipeCommand(logPath string) string {
	return buildShellCommand("sh", "-c", `cat >> "$1"`, "sh", logPath)
}

func enableTmuxPanePipe(paneTarget, pipeCommand string) error {
	if strings.TrimSpace(paneTarget) == "" {
		return fmt.Errorf("tmux pane target cannot be empty")
	}

	pipeCommandExec := exec.Command("tmux", "pipe-pane", "-t", paneTarget, pipeCommand)
	output, err := pipeCommandExec.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux pipe-pane failed: %w\n%s", err, output)
	}

	return nil
}

func disableTmuxPanePipe(metadata session.Metadata) error {
	paneTarget := metadata.EffectiveTmuxPaneTarget()
	if strings.TrimSpace(paneTarget) == "" {
		return nil
	}

	pipeCommandExec := exec.Command("tmux", "pipe-pane", "-t", paneTarget)
	output, err := pipeCommandExec.CombinedOutput()
	if err == nil || isMissingTmuxPane(output) || isNoTmuxServer(output) {
		return nil
	}

	return fmt.Errorf("tmux pipe-pane disable failed: %v\n%s", err, output)
}

func inspectTmuxState() (tmuxState, error) {
	listCommand := exec.Command("tmux", "list-windows", "-a", "-F", "#{window_id}\t#{session_id}\t#{session_name}")
	output, err := listCommand.CombinedOutput()
	if err != nil {
		if isNoTmuxServer(output) {
			return tmuxState{
				sessionIDs:   map[string]struct{}{},
				sessionNames: map[string]struct{}{},
				windowIDs:    map[string]struct{}{},
			}, nil
		}

		return tmuxState{}, fmt.Errorf("tmux list-windows failed: %w\n%s", err, output)
	}

	state := tmuxState{
		sessionIDs:   make(map[string]struct{}),
		sessionNames: make(map[string]struct{}),
		windowIDs:    make(map[string]struct{}),
	}

	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		parts := strings.SplitN(trimmedLine, "\t", 3)
		if len(parts) != 3 {
			return tmuxState{}, fmt.Errorf("unexpected tmux list-windows output: %s", trimmedLine)
		}

		windowID := strings.TrimSpace(parts[0])
		sessionID := strings.TrimSpace(parts[1])
		sessionName := strings.TrimSpace(parts[2])

		if windowID != "" {
			state.windowIDs[windowID] = struct{}{}
		}
		if sessionID != "" {
			state.sessionIDs[sessionID] = struct{}{}
		}
		if sessionName != "" {
			state.sessionNames[sessionName] = struct{}{}
		}
	}

	return state, nil
}

func killTmuxPlacement(metadata session.Metadata) error {
	var cleanupErrors []string

	if err := disableTmuxPanePipe(metadata); err != nil {
		cleanupErrors = append(cleanupErrors, err.Error())
	}

	if metadata.EffectivePlacementKind() == session.PlacementKindWindow {
		if err := killTmuxWindow(metadata.EffectiveTmuxWindowTarget()); err != nil {
			cleanupErrors = append(cleanupErrors, err.Error())
		}
	} else {
		if err := killTmuxSession(metadata.EffectiveTmuxSessionTarget()); err != nil {
			cleanupErrors = append(cleanupErrors, err.Error())
		}
	}

	if len(cleanupErrors) == 0 {
		return nil
	}

	return errors.New(strings.Join(cleanupErrors, "\n"))
}

func killTmuxSession(sessionTarget string) error {
	if strings.TrimSpace(sessionTarget) == "" {
		return nil
	}

	sessionKillCommand := exec.Command("tmux", "kill-session", "-t", sessionTarget)
	output, err := sessionKillCommand.CombinedOutput()
	if err == nil || isMissingTmuxSession(output) || isNoTmuxServer(output) {
		return nil
	}

	return fmt.Errorf("tmux kill-session failed: %v\n%s", err, output)
}

func killTmuxWindow(windowTarget string) error {
	if strings.TrimSpace(windowTarget) == "" {
		return nil
	}

	windowKillCommand := exec.Command("tmux", "kill-window", "-t", windowTarget)
	output, err := windowKillCommand.CombinedOutput()
	if err == nil || isMissingTmuxWindow(output) || isNoTmuxServer(output) {
		return nil
	}

	return fmt.Errorf("tmux kill-window failed: %v\n%s", err, output)
}

func runTmuxInteractiveCommand(args ...string) error {
	command := exec.Command("tmux", args...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	return command.Run()
}

func isInsideTmux() bool {
	return strings.TrimSpace(os.Getenv("TMUX")) != ""
}

func parseTmuxLaunchOutput(output []byte) (tmuxLaunchInfo, error) {
	trimmedOutput := strings.TrimRight(string(output), "\r\n")
	parts := strings.SplitN(trimmedOutput, "\t", 5)
	if len(parts) != 5 {
		return tmuxLaunchInfo{}, fmt.Errorf("unexpected tmux output: %s", trimmedOutput)
	}

	return tmuxLaunchInfo{
		SessionID:   strings.TrimSpace(parts[0]),
		SessionName: strings.TrimSpace(parts[1]),
		WindowID:    strings.TrimSpace(parts[2]),
		WindowName:  strings.TrimSpace(parts[3]),
		PaneID:      strings.TrimSpace(parts[4]),
	}, nil
}

func isMissingTmuxPane(output []byte) bool {
	return strings.Contains(strings.ToLower(string(output)), "can't find pane")
}

func isMissingTmuxWindow(output []byte) bool {
	lowerOutput := strings.ToLower(string(output))
	return strings.Contains(lowerOutput, "can't find window") || strings.Contains(lowerOutput, "can't find pane")
}
