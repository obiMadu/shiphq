package workitem

import "strings"

type WorkMode string

const (
	ModeImplement WorkMode = "implement"
	ModeReview    WorkMode = "review"
)

type SourceRef struct {
	System    string
	Kind      string
	Reference string
}

func (sourceRef SourceRef) RegistryKey() string {
	return sourceRef.System + ":" + sourceRef.Kind
}

type WorkItem struct {
	Mode         WorkMode
	Source       SourceRef
	Identifier   string
	TargetBranch string
	Title        string
	Description  string
	URL          string
}

func (workItem WorkItem) Context() string {
	var builder strings.Builder

	if workItem.Title != "" {
		builder.WriteString(workItem.Title)
	}
	if workItem.Description != "" && workItem.Description != workItem.Title {
		if builder.Len() > 0 {
			builder.WriteString("\n\n")
		}
		builder.WriteString(workItem.Description)
	}

	return builder.String()
}

func NormalizeIdentifier(value string) string {
	trimmedValue := strings.TrimSpace(strings.ToLower(value))
	if trimmedValue == "" {
		return ""
	}

	var builder strings.Builder
	previousWasDash := false

	for _, character := range trimmedValue {
		switch {
		case character >= 'a' && character <= 'z':
			builder.WriteRune(character)
			previousWasDash = false
		case character >= '0' && character <= '9':
			builder.WriteRune(character)
			previousWasDash = false
		default:
			if !previousWasDash && builder.Len() > 0 {
				builder.WriteRune('-')
				previousWasDash = true
			}
		}
	}

	normalizedValue := strings.Trim(builder.String(), "-")
	if len(normalizedValue) > 40 {
		normalizedValue = strings.Trim(normalizedValue[:40], "-")
	}

	return normalizedValue
}
