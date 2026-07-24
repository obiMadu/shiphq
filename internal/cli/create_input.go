package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/obiMadu/wtmag/internal/source"
	"github.com/obiMadu/wtmag/internal/workitem"
)

type CreateInput struct {
	SourceRef      workitem.SourceRef
	PromptOverride string
	Mode           workitem.WorkMode
}

func ResolveCreateInput(githubNumber int, promptText, opsxChange, typeFlag string) (CreateInput, error) {
	trimmedPromptText := strings.TrimSpace(promptText)
	trimmedOpsxChange := strings.TrimSpace(opsxChange)
	trimmedType := strings.TrimSpace(typeFlag)

	type sourceSelection struct {
		system    string
		reference string
	}

	var selected []sourceSelection
	if githubNumber != 0 {
		if githubNumber < 0 {
			return CreateInput{}, fmt.Errorf("--github must be a positive number")
		}
		selected = append(selected, sourceSelection{"github", strconv.Itoa(githubNumber)})
	}
	if trimmedOpsxChange != "" {
		selected = append(selected, sourceSelection{"opsx", trimmedOpsxChange})
	}
	if len(selected) == 0 && trimmedPromptText != "" {
		selected = append(selected, sourceSelection{"prompt", trimmedPromptText})
	}

	if len(selected) == 0 {
		return CreateInput{}, fmt.Errorf("must specify --github, --opsx, or --prompt")
	}
	if len(selected) > 1 {
		return CreateInput{}, fmt.Errorf("must specify only one source: --github, --opsx, or --prompt")
	}

	choice := selected[0]
	promptOverride := ""
	if choice.system != "prompt" {
		promptOverride = trimmedPromptText
	}

	kinds := source.KindsFor(choice.system)
	if len(kinds) == 0 {
		return CreateInput{}, fmt.Errorf("unsupported source: %s", choice.system)
	}

	if len(kinds) == 1 {
		if trimmedType != "" {
			return CreateInput{}, fmt.Errorf("--type is not used with --%s", choice.system)
		}
		return CreateInput{
			SourceRef:      workitem.SourceRef{System: choice.system, Kind: kinds[0].Kind, Reference: choice.reference},
			PromptOverride: promptOverride,
			Mode:           kinds[0].Mode,
		}, nil
	}

	if trimmedType == "" {
		return CreateInput{}, fmt.Errorf("--type (-t) is required for %s (use %s)", choice.system, formatValidKinds(kinds))
	}

	var matched source.Entry
	found := false
	for _, k := range kinds {
		if k.Kind == trimmedType {
			matched = k
			found = true
			break
		}
	}
	if !found {
		return CreateInput{}, fmt.Errorf("unknown type '%s' for %s (use %s)", trimmedType, choice.system, formatValidKinds(kinds))
	}

	return CreateInput{
		SourceRef:      workitem.SourceRef{System: choice.system, Kind: matched.Kind, Reference: choice.reference},
		PromptOverride: promptOverride,
		Mode:           matched.Mode,
	}, nil
}

func formatValidKinds(kinds []source.Entry) string {
	quoted := make([]string, len(kinds))
	for i, k := range kinds {
		quoted[i] = "'" + k.Kind + "'"
	}
	return strings.Join(quoted, " or ")
}
