package openspec

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type changeProvider struct{}

func init() {
	source.Register("opsx", "change", workitem.ModeImplement, changeProvider{})
}

func (changeProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	changeName := strings.TrimSpace(sourceRef.Reference)
	if changeName == "" {
		return workitem.WorkItem{}, fmt.Errorf("OpenSpec change name cannot be empty")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return workitem.WorkItem{}, fmt.Errorf("failed to determine current directory: %w", err)
	}

	changeDir := filepath.Join(cwd, "openspec", "changes", changeName)
	if _, err := os.Stat(changeDir); os.IsNotExist(err) {
		return workitem.WorkItem{}, fmt.Errorf("OpenSpec change directory not found: openspec/changes/%s", changeName)
	}

	proposalPath := filepath.Join(changeDir, "proposal.md")
	proposalContent, err := os.ReadFile(proposalPath)
	if err != nil {
		return workitem.WorkItem{}, fmt.Errorf("failed to read proposal.md for change %s: %w", changeName, err)
	}

	description := strings.TrimSpace(string(proposalContent)) +
		"\n\nRead the full change in `openspec/changes/" + changeName + "/` (design.md, tasks.md, specs/) " +
		"and use the `openspec-apply-change` skill to implement it."

	return workitem.WorkItem{
		Source:      sourceRef,
		Identifier:  workitem.NormalizeIdentifier(changeName),
		Title:       changeName,
		Description: description,
	}, nil
}

func MoveChange(srcDir, destDir, changeName string) error {
	srcChangeDir := filepath.Join(srcDir, "openspec", "changes", changeName)
	if _, err := os.Stat(srcChangeDir); os.IsNotExist(err) {
		return fmt.Errorf("OpenSpec change directory not found: %s", srcChangeDir)
	}

	destChangeDir := filepath.Join(destDir, "openspec", "changes", changeName)
	if _, err := os.Stat(destChangeDir); err == nil {
		return nil
	}

	return moveChangeWithFallback(srcDir, destDir, changeName)
}

func moveChangeWithFallback(srcDir, destDir, changeName string) error {
	srcChangeDir := filepath.Join(srcDir, "openspec", "changes", changeName)
	destChangeDir := filepath.Join(destDir, "openspec", "changes", changeName)

	if err := os.MkdirAll(filepath.Dir(destChangeDir), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	if err := os.Rename(srcChangeDir, destChangeDir); err == nil {
		return nil
	}

	if err := copyDir(srcChangeDir, destChangeDir); err != nil {
		return fmt.Errorf("failed to copy change directory: %w", err)
	}
	if err := os.RemoveAll(srcChangeDir); err != nil {
		return fmt.Errorf("failed to remove source after copy: %w", err)
	}
	return nil
}

func copyDir(src, dest string) error {
	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyFile(src, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
