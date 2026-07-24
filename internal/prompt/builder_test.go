package prompt

import (
	"strings"
	"testing"

	"github.com/obiMadu/wtmag/internal/repository"
	"github.com/obiMadu/wtmag/internal/workitem"
)

func TestBuild_DefaultImplementNoPR(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:       "Fix login bug",
		Description: "The login page crashes on Safari",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Implement GitHub issue #456.") {
		t.Errorf("expected 'Implement GitHub issue #456.', got %q", result)
	}
	if !strings.Contains(result, "Fix login bug") {
		t.Errorf("expected title in output, got %q", result)
	}
	if !strings.Contains(result, "The login page crashes on Safari") {
		t.Errorf("expected description in output, got %q", result)
	}
	if strings.Contains(result, "commit") || strings.Contains(result, "pull request") {
		t.Errorf("expected no PR instruction without --pr, got %q", result)
	}
}

func TestBuild_ImplementWithPR(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:       "Fix login bug",
		Description: "The login page crashes on Safari",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Implement GitHub issue #456.") {
		t.Errorf("expected 'Implement GitHub issue #456.', got %q", result)
	}
	if !strings.Contains(result, "commit your changes") {
		t.Errorf("expected PR instruction with --pr, got %q", result)
	}
	if !strings.Contains(result, "gh") {
		t.Errorf("expected 'gh' in GitHub PR template, got %q", result)
	}
}

func TestBuild_DefaultReview(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeReview,
		Source:      workitem.SourceRef{System: "github", Kind: "pr", Reference: "234"},
		Title:       "Improve logging",
		Description: "Add structured logging",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Review GitHub PR #234.") {
		t.Errorf("expected 'Review GitHub PR #234.', got %q", result)
	}
	if !strings.Contains(result, "Improve logging") {
		t.Errorf("expected title in output, got %q", result)
	}
	if !strings.Contains(result, "review-only task") {
		t.Errorf("expected review instruction in output, got %q", result)
	}
}

func TestBuild_PromptSourceNoLabel(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      workitem.SourceRef{System: "prompt", Kind: "prompt"},
		Title:       "Do something custom",
		Description: "Do something custom",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostUnknown}, BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(result, "Implement") {
		t.Errorf("expected no 'Implement' label for prompt source, got %q", result)
	}
	if !strings.Contains(result, "Do something custom") {
		t.Errorf("expected prompt text in output, got %q", result)
	}
}

func TestBuild_OverrideOnImplement(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:       "Fix login bug",
		Description: "The login page crashes on Safari",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{
		OverrideText: "Focus on tests",
		PRFlag:       true,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.HasPrefix(result, "Focus on tests") {
		t.Errorf("expected override text at start, got %q", result)
	}
	if !strings.Contains(result, "Context:") {
		t.Errorf("expected 'Context:' label with override, got %q", result)
	}
	if !strings.Contains(result, "Fix login bug") {
		t.Errorf("expected context in output, got %q", result)
	}
	if !strings.Contains(result, "commit your changes") {
		t.Errorf("expected PR instruction with --pr, got %q", result)
	}
}

func TestBuild_OverrideOnPromptSource(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:        workitem.ModeImplement,
		Source:      workitem.SourceRef{System: "prompt", Kind: "prompt"},
		Title:       "Do something custom",
		Description: "Do something custom",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostUnknown}, BuildOptions{
		OverrideText: "Custom instructions",
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Custom instructions") {
		t.Errorf("expected override text in output, got %q", result)
	}
	if strings.Contains(result, "Implement") {
		t.Errorf("expected no 'Implement' label for prompt source, got %q", result)
	}
}

func TestBuild_PropReviewReturnsError(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeReview,
		Source: workitem.SourceRef{System: "github", Kind: "pr", Reference: "234"},
	}
	_, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{PRFlag: true})
	if err == nil {
		t.Fatal("expected error for --pr on review task")
	}
	if !strings.Contains(err.Error(), "--pr is only valid with implementation tasks") {
		t.Errorf("expected '--pr is only valid' error, got %q", err.Error())
	}
}

func TestBuild_ReferenceBacklinkForGitHubIssueOnGitHubHost(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:  "Fix login bug",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Refs #456") {
		t.Errorf("expected reference backlink for GitHub issue on GitHub host, got %q", result)
	}
}

func TestBuild_NoReferenceForGitHubPR(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "github", Kind: "pr", Reference: "234"},
		Title:  "Improve logging",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(result, "Refs #") {
		t.Errorf("expected no reference backlink for GitHub PR, got %q", result)
	}
}

func TestBuild_NoReferenceForPromptSource(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "prompt", Kind: "prompt"},
		Title:  "Do something",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(result, "Refs #") {
		t.Errorf("expected no reference backlink for prompt source, got %q", result)
	}
}

func TestBuild_GitLabPRTemplateUsesGlab(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:  "Fix login bug",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitLab}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "glab") {
		t.Errorf("expected 'glab' in GitLab PR template, got %q", result)
	}
}

func TestBuild_UnknownHostFallsBackToGenericPR(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
		Title:  "Fix login bug",
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostUnknown}, BuildOptions{PRFlag: true})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Check the git remote") {
		t.Errorf("expected generic PR template to tell agent to detect host, got %q", result)
	}
}

func TestBuild_EmptyContextImplement(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeImplement,
		Source: workitem.SourceRef{System: "github", Kind: "issue", Reference: "456"},
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Implement GitHub issue #456.") {
		t.Errorf("expected 'Implement GitHub issue #456.', got %q", result)
	}
}

func TestBuild_EmptyContextReview(t *testing.T) {
	workItem := workitem.WorkItem{
		Mode:   workitem.ModeReview,
		Source: workitem.SourceRef{System: "github", Kind: "pr", Reference: "234"},
	}
	result, err := Build(workItem, repository.Target{Host: repository.HostGitHub}, BuildOptions{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if !strings.Contains(result, "Review GitHub PR #234.") {
		t.Errorf("expected 'Review GitHub PR #234.', got %q", result)
	}
	if !strings.Contains(result, "review-only task") {
		t.Errorf("expected review instruction in output, got %q", result)
	}
}
