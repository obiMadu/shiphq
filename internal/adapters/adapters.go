package adapters

import (
	"fmt"
	"os/exec"
	"strings"
)

type Task struct {
	ID          string
	Source      string
	Type        string
	Title       string
	Description string
	Prompt      string
}

func FetchGitHub(issueNum int, specifiedType string) (Task, error) {
	// Type must be specified for GitHub
	if specifiedType == "" {
		return Task{}, fmt.Errorf("--type (-t) is required for GitHub (use 'issue' or 'pr')")
	}

	if specifiedType == "pr" {
		return fetchGitHubPR(issueNum)
	} else if specifiedType == "issue" {
		return fetchGitHubIssue(issueNum)
	}

	return Task{}, fmt.Errorf("unknown type '%s' for GitHub (use 'issue' or 'pr')", specifiedType)
}

func fetchGitHubPR(num int) (Task, error) {
	cmd := exec.Command("gh", "pr", "view", fmt.Sprintf("%d", num), "--json", "title,body", "-q", ".title,.body")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Task{}, fmt.Errorf("failed to fetch GitHub PR %d: %w\n%s", num, err, out)
	}
	return parseGitHubOutput(num, out, "pr")
}

func fetchGitHubIssue(num int) (Task, error) {
	cmd := exec.Command("gh", "issue", "view", fmt.Sprintf("%d", num), "--json", "title,body", "-q", ".title,.body")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Task{}, fmt.Errorf("failed to fetch GitHub issue %d: %w\n%s", num, err, out)
	}
	return parseGitHubOutput(num, out, "issue")
}

func parseGitHubOutput(num int, out []byte, typ string) (Task, error) {
	lines := strings.SplitN(string(out), "\n", 2)
	title := strings.TrimSpace(lines[0])
	desc := ""
	if len(lines) > 1 {
		desc = strings.TrimSpace(lines[1])
	}

	var prompt string
	if typ == "pr" {
		prompt = fmt.Sprintf("Review GitHub PR #%d: %s\n\n%s", num, title, desc)
	} else {
		prompt = fmt.Sprintf("Implement GitHub issue #%d: %s\n\n%s", num, title, desc)
	}

	return Task{
		ID:          fmt.Sprintf("%d", num),
		Source:      "github",
		Type:        typ,
		Title:       title,
		Description: desc,
		Prompt:      prompt,
	}, nil
}

func FetchJira(ticketID string) (Task, error) {
	return Task{
		ID:          ticketID,
		Source:      "jira",
		Type:        "ticket",
		Title:       ticketID,
		Description: "",
		Prompt:      fmt.Sprintf("Implement Jira ticket %s", ticketID),
	}, fmt.Errorf("Jira adapter not yet implemented (ticket: %s)", ticketID)
}

func TaskFromPrompt(prompt string) Task {
	id := generateID(prompt)
	return Task{
		ID:          id,
		Source:      "prompt",
		Type:        "prompt",
		Title:       prompt,
		Description: prompt,
		Prompt:      prompt,
	}
}

func generateID(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "-")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "_", "-")
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
