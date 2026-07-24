package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type fakeProvider struct{}

func (fakeProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	return workitem.WorkItem{}, nil
}

func TestMain(m *testing.M) {
	source.Register("github", "issue", workitem.ModeImplement, fakeProvider{})
	source.Register("github", "pr", workitem.ModeReview, fakeProvider{})
	source.Register("prompt", "prompt", workitem.ModeImplement, fakeProvider{})
	os.Exit(m.Run())
}

func TestResolveCreateInput(t *testing.T) {
	t.Run("valid combinations", func(t *testing.T) {
		tests := []struct {
			name               string
			githubNumber       int
			promptText         string
			typeFlag           string
			wantSystem         string
			wantKind           string
			wantReference      string
			wantPromptOverride string
			wantMode           workitem.WorkMode
		}{
			{"github issue", 456, "", "issue", "github", "issue", "456", "", workitem.ModeImplement},
			{"github pr", 456, "", "pr", "github", "pr", "456", "", workitem.ModeReview},
			{"prompt", 0, "do something", "", "prompt", "prompt", "do something", "", workitem.ModeImplement},
			{"github issue with prompt override", 456, "override", "issue", "github", "issue", "456", "override", workitem.ModeImplement},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				input, err := ResolveCreateInput(tt.githubNumber, tt.promptText, tt.typeFlag)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if input.SourceRef.System != tt.wantSystem {
					t.Errorf("System = %q, want %q", input.SourceRef.System, tt.wantSystem)
				}
				if input.SourceRef.Kind != tt.wantKind {
					t.Errorf("Kind = %q, want %q", input.SourceRef.Kind, tt.wantKind)
				}
				if input.SourceRef.Reference != tt.wantReference {
					t.Errorf("Reference = %q, want %q", input.SourceRef.Reference, tt.wantReference)
				}
				if input.PromptOverride != tt.wantPromptOverride {
					t.Errorf("PromptOverride = %q, want %q", input.PromptOverride, tt.wantPromptOverride)
				}
				if input.Mode != tt.wantMode {
					t.Errorf("Mode = %q, want %q", input.Mode, tt.wantMode)
				}
			})
		}
	})

	t.Run("error paths", func(t *testing.T) {
		tests := []struct {
			name         string
			githubNumber int
			promptText   string
			typeFlag     string
			wantErrSubs  []string
		}{
			{"no source flag", 0, "", "", []string{"must specify --github or --prompt"}},
			{"github without -t", 456, "", "", []string{"--type (-t) is required for"}},
			{"github with unknown -t", 456, "", "mr", []string{"unknown type", "issue", "pr"}},
			{"prompt with -t", 0, "do something", "prompt", []string{"--type is not used with --prompt"}},
			{"github negative number", -1, "", "issue", []string{"must be a positive number"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := ResolveCreateInput(tt.githubNumber, tt.promptText, tt.typeFlag)
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				for _, want := range tt.wantErrSubs {
					if !strings.Contains(err.Error(), want) {
						t.Errorf("error %q does not contain %q", err.Error(), want)
					}
				}
			})
		}
	})
}
