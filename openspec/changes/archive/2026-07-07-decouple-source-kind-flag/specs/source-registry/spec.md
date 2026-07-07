## ADDED Requirements

### Requirement: Registry stores mode alongside system, kind, and fetcher

The source registry SHALL store a `workitem.WorkMode` value for every registered `(system, kind)` entry. `Register` SHALL accept `mode` as a parameter. Each provider's `init()` SHALL declare its mode at registration rather than setting it inside `Fetch`.

#### Scenario: Provider registers with its mode
- **WHEN** `source.Register("github", "issue", workitem.ModeImplement, issueProvider{})` is called
- **THEN** the registry entry for `github:issue` stores `ModeImplement` alongside the fetcher

#### Scenario: Fetch no longer sets Mode
- **WHEN** `issueProvider{}.Fetch(sourceRef)` is called
- **THEN** the returned `WorkItem.Mode` is the zero value (caller stamps it from the registry entry)

### Requirement: Registry answers KindsFor queries

The registry SHALL expose `KindsFor(system string) []entry` returning every registered kind for the given system, each with its mode and fetcher. Callers SHALL use this to discover a source's `(kind, mode)` surface area without hardcoding.

#### Scenario: Multi-kind source lookup
- **WHEN** `KindsFor("github")` is called after `github:issue` (Implement) and `github:pr` (Review) are registered
- **THEN** it returns two entries: `{Kind:"issue", Mode:Implement}` and `{Kind:"pr", Mode:Review}`

#### Scenario: Single-kind source lookup
- **WHEN** `KindsFor("jira")` is called after `jira:ticket` (Implement) is registered
- **THEN** it returns one entry: `{Kind:"ticket", Mode:Implement}`

#### Scenario: Unknown system lookup
- **WHEN** `KindsFor("nonexistent")` is called
- **THEN** it returns an empty slice (no error)

### Requirement: Registry resolves a (system, kind) to its fetcher and mode

The registry SHALL expose a way to look up the fetcher and mode for a specific `(system, kind)` pair, so the resolver can fetch the work item and stamp its mode from a single source of truth.

#### Scenario: Resolve a registered kind
- **WHEN** the resolver looks up `github:pr`
- **THEN** it receives the `pullRequestProvider` fetcher and `ModeReview`

#### Scenario: Resolve an unregistered kind
- **WHEN** the resolver looks up `github:mr` (not registered)
- **THEN** it receives an error indicating the kind is unsupported for that system

### Requirement: Provider self-registration remains init()-based

Providers SHALL continue to self-register via package `init()`, wired by a blank import in `internal/source/providers/providers.go`. No central registration list SHALL be maintained outside the providers themselves.

#### Scenario: Adding a provider package
- **WHEN** a new provider package is added and blank-imported in `providers.go`
- **THEN** its `init()` runs and its `(system, kind, mode)` is registered without any edit to the registry or resolver
