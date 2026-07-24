package prompt

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func writeTmplFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func executeTmpl(t *testing.T, tmpl *template.Template, data interface{}) string {
	t.Helper()
	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		t.Fatalf("execute template: %v", err)
	}
	return strings.TrimSpace(sb.String())
}

func TestResolveTemplate_ProjectHostSpecificWins(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	writeTmplFile(t, projectDir, "pr-github.tmpl", "project-github")
	writeTmplFile(t, globalDir, "pr-github.tmpl", "global-github")
	writeTmplFile(t, projectDir, "pr.tmpl", "project-generic")
	writeTmplFile(t, globalDir, "pr.tmpl", "global-generic")

	tmpl, err := resolveTemplateFrom("pr", "github", projectDir, globalDir)
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if result != "project-github" {
		t.Errorf("expected 'project-github', got %q", result)
	}
}

func TestResolveTemplate_GlobalHostSpecificBeatsProjectGeneric(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	writeTmplFile(t, globalDir, "pr-github.tmpl", "global-github")
	writeTmplFile(t, projectDir, "pr.tmpl", "project-generic")
	writeTmplFile(t, globalDir, "pr.tmpl", "global-generic")

	tmpl, err := resolveTemplateFrom("pr", "github", projectDir, globalDir)
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if result != "global-github" {
		t.Errorf("expected 'global-github', got %q", result)
	}
}

func TestResolveTemplate_ProjectGenericBeatsGlobalGeneric(t *testing.T) {
	projectDir := t.TempDir()
	globalDir := t.TempDir()

	writeTmplFile(t, projectDir, "pr.tmpl", "project-generic")
	writeTmplFile(t, globalDir, "pr.tmpl", "global-generic")

	tmpl, err := resolveTemplateFrom("pr", "github", projectDir, globalDir)
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if result != "project-generic" {
		t.Errorf("expected 'project-generic', got %q", result)
	}
}

func TestResolveTemplate_GlobalGenericBeatsEmbedded(t *testing.T) {
	globalDir := t.TempDir()

	writeTmplFile(t, globalDir, "pr.tmpl", "global-generic")

	tmpl, err := resolveTemplateFrom("pr", "github", "", globalDir)
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if result != "global-generic" {
		t.Errorf("expected 'global-generic', got %q", result)
	}
}

func TestResolveTemplate_EmbeddedHostSpecific(t *testing.T) {
	tmpl, err := resolveTemplateFrom("pr", "github", "", "")
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if !strings.Contains(result, "gh") {
		t.Errorf("expected 'gh' from embedded pr-github.tmpl, got %q", result)
	}
}

func TestResolveTemplate_EmbeddedGenericForUnknownHost(t *testing.T) {
	tmpl, err := resolveTemplateFrom("pr", "unknown", "", "")
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, PRData{})
	if !strings.Contains(result, "Check the git remote") {
		t.Errorf("expected generic pr.tmpl for unknown host, got %q", result)
	}
}

func TestResolveTemplate_EmbeddedDefaultImplement(t *testing.T) {
	tmpl, err := resolveTemplateFrom("implement", "unknown", "", "")
	if err != nil {
		t.Fatalf("resolveTemplateFrom: %v", err)
	}
	result := executeTmpl(t, tmpl, FrameData{
		SourceLabel: "GitHub issue #1",
		Context:     "Fix the bug",
	})
	if !strings.Contains(result, "Implement GitHub issue #1.") {
		t.Errorf("expected 'Implement GitHub issue #1.', got %q", result)
	}
	if !strings.Contains(result, "Fix the bug") {
		t.Errorf("expected context in output, got %q", result)
	}
}
