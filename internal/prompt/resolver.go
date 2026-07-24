package prompt

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/obiMadu/wtmag/internal/config"
)

//go:embed templates/*.tmpl
var embeddedTemplates embed.FS

func resolveTemplate(typeName, host string) (*template.Template, error) {
	projectPromptsDir, err := resolveProjectPromptsDir()
	if err != nil {
		return nil, err
	}
	globalPromptsDir := filepath.Join(config.GetConfigDir(), "prompts")
	return resolveTemplateFrom(typeName, host, projectPromptsDir, globalPromptsDir)
}

func resolveTemplateFrom(typeName, host, projectPromptsDir, globalPromptsDir string) (*template.Template, error) {
	var candidates []string
	addCandidate := func(dir, suffix string) {
		if dir != "" {
			candidates = append(candidates, filepath.Join(dir, suffix))
		}
	}

	addCandidate(projectPromptsDir, typeName+"-"+host+".tmpl")
	addCandidate(globalPromptsDir, typeName+"-"+host+".tmpl")
	addCandidate(projectPromptsDir, typeName+".tmpl")
	addCandidate(globalPromptsDir, typeName+".tmpl")

	for _, candidate := range candidates {
		content, err := os.ReadFile(candidate)
		if err == nil {
			return template.New(filepath.Base(candidate)).Parse(string(content))
		}
	}

	embeddedHostPath := "templates/" + typeName + "-" + host + ".tmpl"
	if content, err := embeddedTemplates.ReadFile(embeddedHostPath); err == nil {
		return template.New(typeName + "-" + host + ".tmpl").Parse(string(content))
	}

	embeddedGenericPath := "templates/" + typeName + ".tmpl"
	content, err := embeddedTemplates.ReadFile(embeddedGenericPath)
	if err != nil {
		return nil, fmt.Errorf("no template found for %s (host %s): %w", typeName, host, err)
	}
	return template.New(typeName + ".tmpl").Parse(string(content))
}

func resolveProjectPromptsDir() (string, error) {
	projectConfigPath, err := config.GetProjectConfigPath()
	if err != nil || projectConfigPath == "" {
		return "", err
	}
	return filepath.Join(filepath.Dir(projectConfigPath), "wtmag-prompts"), nil
}
