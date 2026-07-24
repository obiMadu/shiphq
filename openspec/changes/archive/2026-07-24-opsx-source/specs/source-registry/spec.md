## ADDED Requirements

### Requirement: OpenSpec provider is registered

The `opsx:change` provider SHALL be registered via `init()` in the `openspec` provider package, wired by a blank import in `internal/source/providers/providers.go`. The registration SHALL pass `workitem.ModeImplement` as the mode.

#### Scenario: KindsFor returns the OpenSpec entry
- **WHEN** `KindsFor("opsx")` is called after the `openspec` provider package is blank-imported
- **THEN** it returns one entry: `{Kind:"change", Mode:ModeImplement}`
