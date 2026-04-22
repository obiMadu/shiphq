package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/obiMadu/shiphq/internal/config"
	"github.com/obiMadu/shiphq/internal/workitem"
)

type PlacementKind string

const (
	PlacementKindSession PlacementKind = "session"
	PlacementKindWindow  PlacementKind = "window"
)

type Metadata struct {
	SessionID    string            `json:"session_id"`
	Project      string            `json:"project"`
	Branch       string            `json:"branch"`
	WorktreePath string            `json:"worktree_path"`
	WorkItem     workitem.WorkItem `json:"work_item"`

	PlacementKind   PlacementKind `json:"placement_kind,omitempty"`
	TmuxSessionID   string        `json:"tmux_session_id,omitempty"`
	TmuxSessionName string        `json:"tmux_session_name,omitempty"`
	TmuxWindowID    string        `json:"tmux_window_id,omitempty"`
	TmuxWindowName  string        `json:"tmux_window_name,omitempty"`
}

func (metadata Metadata) EffectivePlacementKind() PlacementKind {
	if metadata.PlacementKind == PlacementKindWindow {
		return PlacementKindWindow
	}

	return PlacementKindSession
}

func (metadata Metadata) EffectiveTmuxSessionTarget() string {
	if trimmedSessionID := strings.TrimSpace(metadata.TmuxSessionID); trimmedSessionID != "" {
		return trimmedSessionID
	}

	if trimmedSessionName := strings.TrimSpace(metadata.TmuxSessionName); trimmedSessionName != "" {
		return trimmedSessionName
	}

	return strings.TrimSpace(metadata.SessionID)
}

func (metadata Metadata) EffectiveTmuxWindowTarget() string {
	return strings.TrimSpace(metadata.TmuxWindowID)
}

func Save(metadata Metadata) error {
	if metadata.SessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}
	if metadata.Branch == "" {
		return fmt.Errorf("branch cannot be empty")
	}
	if metadata.WorktreePath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	if err := config.EnsureSessionStateDir(); err != nil {
		return fmt.Errorf("failed to prepare session state directory: %w", err)
	}

	encodedMetadata, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode session metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath(metadata.SessionID), encodedMetadata, 0644); err != nil {
		return fmt.Errorf("failed to write session metadata: %w", err)
	}

	return nil
}

func Load(sessionID string) (Metadata, error) {
	if sessionID == "" {
		return Metadata{}, fmt.Errorf("session ID cannot be empty")
	}

	encodedMetadata, err := os.ReadFile(metadataPath(sessionID))
	if err != nil {
		return Metadata{}, fmt.Errorf("failed to read session metadata for %s: %w", sessionID, err)
	}

	var metadata Metadata
	if err := json.Unmarshal(encodedMetadata, &metadata); err != nil {
		return Metadata{}, fmt.Errorf("failed to decode session metadata for %s: %w", sessionID, err)
	}

	return metadata, nil
}

func List() ([]Metadata, error) {
	if err := config.EnsureSessionStateDir(); err != nil {
		return nil, fmt.Errorf("failed to prepare session state directory: %w", err)
	}

	entries, err := os.ReadDir(config.GetSessionStateDir())
	if err != nil {
		return nil, fmt.Errorf("failed to read session metadata directory: %w", err)
	}

	metadataItems := make([]Metadata, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if sessionID == "" {
			continue
		}

		metadata, err := Load(sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to load session metadata for %s: %w", sessionID, err)
		}

		metadataItems = append(metadataItems, metadata)
	}

	sort.Slice(metadataItems, func(i, j int) bool {
		return metadataItems[i].SessionID < metadataItems[j].SessionID
	})

	return metadataItems, nil
}

func Delete(sessionID string) error {
	if sessionID == "" {
		return fmt.Errorf("session ID cannot be empty")
	}

	if err := os.Remove(metadataPath(sessionID)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove session metadata for %s: %w", sessionID, err)
	}

	return nil
}

func metadataPath(sessionID string) string {
	return filepath.Join(config.GetSessionStateDir(), sessionID+".json")
}
