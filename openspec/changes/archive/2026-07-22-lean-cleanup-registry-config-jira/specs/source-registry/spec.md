## MODIFIED Requirements

### Requirement: Registry answers KindsFor queries

The registry SHALL expose `KindsFor(system string) []entry` returning every registered kind for the given system, each with its mode and fetcher. Callers SHALL use this to discover a source's `(kind, mode)` surface area without hardcoding.

#### Scenario: Multi-kind source lookup
- **WHEN** `KindsFor("github")` is called after `github:issue` (Implement) and `github:pr` (Review) are registered
- **THEN** it returns two entries: `{Kind:"issue", Mode:Implement}` and `{Kind:"pr", Mode:Review}`

#### Scenario: Single-kind source lookup
- **WHEN** `KindsFor("prompt")` is called after `prompt:prompt` (Implement) is registered
- **THEN** it returns one entry: `{Kind:"prompt", Mode:Implement}`

#### Scenario: Unknown system lookup
- **WHEN** `KindsFor("nonexistent")` is called
- **THEN** it returns an empty slice (no error)
