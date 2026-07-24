package source

import (
	"testing"

	"github.com/obiMadu/wtmag/internal/workitem"
)

type fakeProvider struct{}

func (fakeProvider) Fetch(sourceRef workitem.SourceRef) (workitem.WorkItem, error) {
	return workitem.WorkItem{}, nil
}

func TestMain(m *testing.M) {
	Register("github", "issue", workitem.ModeImplement, fakeProvider{})
	Register("github", "pr", workitem.ModeReview, fakeProvider{})
	Register("prompt", "prompt", workitem.ModeImplement, fakeProvider{})
	m.Run()
}

func TestResolve(t *testing.T) {
	t.Run("registered kind", func(t *testing.T) {
		fetcher, mode, err := Resolve("github", "pr")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if fetcher == nil {
			t.Error("fetcher = nil, want non-nil")
		}
		if mode != workitem.ModeReview {
			t.Errorf("mode = %q, want %q", mode, workitem.ModeReview)
		}
	})

	t.Run("unregistered kind", func(t *testing.T) {
		_, _, err := Resolve("github", "mr")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestKindsFor(t *testing.T) {
	t.Run("multi-kind source", func(t *testing.T) {
		entries := KindsFor("github")
		if len(entries) != 2 {
			t.Fatalf("got %d entries, want 2", len(entries))
		}
		wantKinds := map[string]workitem.WorkMode{
			"issue": workitem.ModeImplement,
			"pr":    workitem.ModeReview,
		}
		for _, entry := range entries {
			wantMode, ok := wantKinds[entry.Kind]
			if !ok {
				t.Errorf("unexpected kind %q", entry.Kind)
				continue
			}
			if entry.Mode != wantMode {
				t.Errorf("kind %q: mode = %q, want %q", entry.Kind, entry.Mode, wantMode)
			}
		}
	})

	t.Run("unknown system", func(t *testing.T) {
		entries := KindsFor("nonexistent")
		if len(entries) != 0 {
			t.Fatalf("got %d entries, want 0", len(entries))
		}
	})
}
