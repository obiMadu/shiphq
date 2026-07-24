package openspec

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveChange(t *testing.T) {
	t.Run("untracked directory moved", func(t *testing.T) {
		srcDir := t.TempDir()
		destDir := t.TempDir()
		changeName := "test-change"

		srcChange := filepath.Join(srcDir, "openspec", "changes", changeName)
		if err := os.MkdirAll(srcChange, 0755); err != nil {
			t.Fatalf("failed to create source change dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcChange, "proposal.md"), []byte("test content"), 0644); err != nil {
			t.Fatalf("failed to write proposal: %v", err)
		}

		if err := MoveChange(srcDir, destDir, changeName); err != nil {
			t.Fatalf("MoveChange returned error: %v", err)
		}

		movedContent, err := os.ReadFile(filepath.Join(destDir, "openspec", "changes", changeName, "proposal.md"))
		if err != nil {
			t.Fatalf("file not found at destination: %v", err)
		}
		if string(movedContent) != "test content" {
			t.Errorf("content = %q, want %q", string(movedContent), "test content")
		}

		if _, err := os.Stat(srcChange); !os.IsNotExist(err) {
			t.Errorf("source directory should no longer exist after move")
		}
	})

	t.Run("dest already exists skip", func(t *testing.T) {
		srcDir := t.TempDir()
		destDir := t.TempDir()
		changeName := "existing-change"

		srcChange := filepath.Join(srcDir, "openspec", "changes", changeName)
		if err := os.MkdirAll(srcChange, 0755); err != nil {
			t.Fatalf("failed to create source change dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcChange, "proposal.md"), []byte("source"), 0644); err != nil {
			t.Fatalf("failed to write source proposal: %v", err)
		}

		destChange := filepath.Join(destDir, "openspec", "changes", changeName)
		if err := os.MkdirAll(destChange, 0755); err != nil {
			t.Fatalf("failed to create dest change dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(destChange, "proposal.md"), []byte("dest"), 0644); err != nil {
			t.Fatalf("failed to write dest proposal: %v", err)
		}

		if err := MoveChange(srcDir, destDir, changeName); err != nil {
			t.Fatalf("MoveChange returned error when dest exists: %v", err)
		}

		destContent, err := os.ReadFile(filepath.Join(destChange, "proposal.md"))
		if err != nil {
			t.Fatalf("failed to read dest: %v", err)
		}
		if string(destContent) != "dest" {
			t.Errorf("dest content should be unchanged; got %q, want %q", string(destContent), "dest")
		}
	})

	t.Run("source missing error", func(t *testing.T) {
		srcDir := t.TempDir()
		destDir := t.TempDir()

		err := MoveChange(srcDir, destDir, "nonexistent")
		if err == nil {
			t.Fatal("expected error for missing source, got nil")
		}
		if !strings.Contains(err.Error(), "nonexistent") {
			t.Errorf("error should mention the change name; got: %v", err)
		}
	})

	t.Run("cross-filesystem fallback", func(t *testing.T) {
		srcDir := t.TempDir()
		destDir := t.TempDir()
		changeName := "cross-fs-change"

		srcChange := filepath.Join(srcDir, "openspec", "changes", changeName)
		if err := os.MkdirAll(srcChange, 0755); err != nil {
			t.Fatalf("failed to create source change dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(srcChange, "proposal.md"), []byte("cross fs"), 0644); err != nil {
			t.Fatalf("failed to write proposal: %v", err)
		}

		if err := moveChangeWithFallback(srcDir, destDir, changeName); err != nil {
			t.Fatalf("moveChangeWithFallback returned error: %v", err)
		}

		movedContent, err := os.ReadFile(filepath.Join(destDir, "openspec", "changes", changeName, "proposal.md"))
		if err != nil {
			t.Fatalf("file not found at destination: %v", err)
		}
		if string(movedContent) != "cross fs" {
			t.Errorf("content = %q, want %q", string(movedContent), "cross fs")
		}
	})
}
