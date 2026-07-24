package openspec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/obiMadu/wtmag/internal/workitem"
)

func TestFetch(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	changeName := "add-user-auth"
	changeDir := filepath.Join(tempDir, "openspec", "changes", changeName)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}

	proposalContent := "## Why\n\nWe need user authentication.\n\n## What Changes\n\nAdd login and registration."
	if err := os.WriteFile(filepath.Join(changeDir, "proposal.md"), []byte(proposalContent), 0644); err != nil {
		t.Fatalf("failed to write proposal.md: %v", err)
	}

	provider := changeProvider{}
	workItem, err := provider.Fetch(workitem.SourceRef{
		System:    "opsx",
		Kind:      "change",
		Reference: changeName,
	})
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}

	if workItem.Source.System != "opsx" {
		t.Errorf("Source.System = %q, want %q", workItem.Source.System, "opsx")
	}
	if workItem.Source.Kind != "change" {
		t.Errorf("Source.Kind = %q, want %q", workItem.Source.Kind, "change")
	}
	if workItem.Source.Reference != changeName {
		t.Errorf("Source.Reference = %q, want %q", workItem.Source.Reference, changeName)
	}

	wantIdentifier := workitem.NormalizeIdentifier(changeName)
	if workItem.Identifier != wantIdentifier {
		t.Errorf("Identifier = %q, want %q", workItem.Identifier, wantIdentifier)
	}

	if workItem.Title != changeName {
		t.Errorf("Title = %q, want %q", workItem.Title, changeName)
	}

	if !strings.Contains(workItem.Description, proposalContent) {
		t.Errorf("Description does not contain proposal content; got:\n%s", workItem.Description)
	}
	if !strings.Contains(workItem.Description, "openspec-apply-change") {
		t.Errorf("Description does not mention openspec-apply-change skill; got:\n%s", workItem.Description)
	}
	if !strings.Contains(workItem.Description, changeName) {
		t.Errorf("Description does not reference the change name; got:\n%s", workItem.Description)
	}
}

func TestFetchMissingDirectory(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	provider := changeProvider{}
	_, err := provider.Fetch(workitem.SourceRef{
		System:    "opsx",
		Kind:      "change",
		Reference: "nonexistent-change",
	})
	if err == nil {
		t.Fatal("expected error for missing change directory, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent-change") {
		t.Errorf("error should mention the change name; got: %v", err)
	}
}

func TestFetchMissingProposal(t *testing.T) {
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to chdir to temp dir: %v", err)
	}

	changeName := "empty-change"
	changeDir := filepath.Join(tempDir, "openspec", "changes", changeName)
	if err := os.MkdirAll(changeDir, 0755); err != nil {
		t.Fatalf("failed to create change dir: %v", err)
	}

	provider := changeProvider{}
	_, err := provider.Fetch(workitem.SourceRef{
		System:    "opsx",
		Kind:      "change",
		Reference: changeName,
	})
	if err == nil {
		t.Fatal("expected error for missing proposal.md, got nil")
	}
	if !strings.Contains(err.Error(), "proposal.md") {
		t.Errorf("error should mention proposal.md; got: %v", err)
	}
}
