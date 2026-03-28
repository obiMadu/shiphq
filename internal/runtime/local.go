package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/obiMadu/shiphq/internal/adapters"
)

type Session struct {
	ID      string
	Runtime string
	Status  string
	URL     string
}

type Runtime interface {
	Create(project string, task adapters.Task) (Session, error)
	List(project string) ([]Session, error)
	Attach(sessionID string) error
	Cleanup(sessionID string) error
}

var runtimes = map[string]Runtime{
	"local": &LocalRuntime{},
}

func Get(name string) (Runtime, error) {
	rt, ok := runtimes[name]
	if !ok {
		return nil, fmt.Errorf("unknown runtime: %s", name)
	}
	return rt, nil
}

type LocalRuntime struct{}

func (l *LocalRuntime) Create(project string, task adapters.Task) (Session, error) {
	branch := generateBranchName(task)
	sessionID := fmt.Sprintf("%s-%s", project, branch)

	cmd := exec.Command("wt", "switch", "--create", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("wt switch failed: %w\n%s", err, out)
	}

	tmuxCmd := exec.Command("tmux", "new-session", "-d", "-s", sessionID)
	if out, err := tmuxCmd.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("tmux create failed: %w\n%s", err, out)
	}

	opencodeCmd := fmt.Sprintf("opencode --prompt %q", task.Prompt)
	windowCmd := exec.Command("tmux", "new-window", "-t", sessionID, "-n", "agent", opencodeCmd)
	if out, err := windowCmd.CombinedOutput(); err != nil {
		return Session{}, fmt.Errorf("tmux window failed: %w\n%s", err, out)
	}

	return Session{
		ID:      sessionID,
		Runtime: "local",
		Status:  "running",
	}, nil
}

func (l *LocalRuntime) List(project string) ([]Session, error) {
	cmd := exec.Command("tmux", "list-sessions", "-F", "#S:#{session_attached}")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, nil
	}

	var sessions []Session
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		if strings.HasPrefix(name, project+"-") {
			sessions = append(sessions, Session{
				ID:      name,
				Runtime: "local",
				Status:  "running",
			})
		}
	}
	return sessions, nil
}

func (l *LocalRuntime) Attach(sessionID string) error {
	cmd := exec.Command("tmux", "attach", "-t", sessionID)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (l *LocalRuntime) Cleanup(sessionID string) error {
	parts := strings.SplitN(sessionID, "-", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid session ID format: %s", sessionID)
	}
	branch := parts[1]

	cmd := exec.Command("wt", "remove", branch)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("wt remove failed: %w\n%s", err, out)
	}

	tmuxCmd := exec.Command("tmux", "kill-session", "-t", sessionID)
	tmuxCmd.CombinedOutput()

	return nil
}

func generateBranchName(task adapters.Task) string {
	// Format: {project}-{source}-{type}-{id}
	// Examples:
	//   blog-github-issue-456
	//   blog-jira-ticket-PROJ-123
	//   blog-prompt-refactor-auth-middleware...

	id := strings.ToLower(task.ID)
	id = strings.ReplaceAll(id, " ", "-")
	id = strings.ReplaceAll(id, "/", "-")
	id = strings.ReplaceAll(id, "_", "-")
	id = strings.ReplaceAll(id, ":", "-")
	if len(id) > 40 {
		id = id[:40]
	}

	return fmt.Sprintf("%s-%s-%s", task.Source, task.Type, id)
}
