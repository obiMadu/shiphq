# source-input-resolution

## Purpose

TBD

## Requirements

### Requirement: Exactly one source flag must be set

`ResolveCreateInput` SHALL accept at most one of the source flags (`--github`, `--prompt`). If zero are set, it SHALL error "must specify --github or --prompt". If more than one is set, it SHALL error "must specify only one source: --github or --prompt". The "more than one" guard SHALL remain as defensive validation for future source flags, even though it is currently unreachable with only two flags.

#### Scenario: No source flag
- **WHEN** `ResolveCreateInput(0, "", "")` is called
- **THEN** it returns an error whose message contains "must specify --github or --prompt"

#### Scenario: Multiple source flags
- **WHEN** two source flags are set simultaneously (a hypothetical third source flag exists alongside `--github`)
- **THEN** it returns an error whose message contains "must specify only one source"

### Requirement: Single-kind sources forbid -t and infer their kind

For a source whose registered kinds number exactly one, `ResolveCreateInput` SHALL forbid `-t` (error if set) and SHALL use the single registered kind as the `SourceRef.Kind`.

#### Scenario: Prompt with -t rejected
- **WHEN** `ResolveCreateInput(0, "", "do something", "prompt")` is called
- **THEN** it returns an error indicating `--type` is not used with `--prompt`

#### Scenario: Prompt without -t resolves
- **WHEN** `ResolveCreateInput(0, "", "do something", "")` is called
- **THEN** it returns a `SourceRef` with `System:"prompt"`, `Kind:"prompt"`, `Reference:"do something"`

### Requirement: Multi-kind sources require -t and validate it against the registry

For a source whose registered kinds number more than one, `ResolveCreateInput` SHALL require `-t` (error if empty) and SHALL validate `-t` against the kinds returned by `registry.KindsFor(system)`. An unknown value SHALL error naming the valid kinds for that source.

#### Scenario: GitHub without -t rejected
- **WHEN** `ResolveCreateInput(456, "", "", "")` is called
- **THEN** it returns an error indicating `--type (-t)` is required for GitHub

#### Scenario: GitHub with valid -t issue
- **WHEN** `ResolveCreateInput(456, "", "", "issue")` is called
- **THEN** it returns a `SourceRef` with `System:"github"`, `Kind:"issue"`, `Reference:"456"`

#### Scenario: GitHub with valid -t pr
- **WHEN** `ResolveCreateInput(456, "", "", "pr")` is called
- **THEN** it returns a `SourceRef` with `System:"github"`, `Kind:"pr"`, `Reference:"456"`

#### Scenario: GitHub with unknown -t rejected
- **WHEN** `ResolveCreateInput(456, "", "", "mr")` is called
- **THEN** it returns an error naming `issue` and `pr` as the valid kinds for GitHub

### Requirement: -t is decoupled from GitHub and works for any multi-kind source

`-t` validation SHALL be driven by `registry.KindsFor(selectedSystem)`, not by a hardcoded GitHub branch. Adding a new multi-kind source (e.g., GitLab with `issue` and `mr`) SHALL require no edit to `ResolveCreateInput` for `-t` to accept the new source's kinds.

#### Scenario: A hypothetical second multi-kind source validates -t against its own kinds
- **WHEN** a source `gitlab` is registered with kinds `issue` (Implement) and `mr` (Review), and `ResolveCreateInput` is called with the gitlab flag set and `-t mr`
- **THEN** it returns a `SourceRef` with `System:"gitlab"`, `Kind:"mr"` (no resolver edit was needed beyond the gitlab flag existing)

### Requirement: Mode is stamped from the registry entry, not hardcoded in Fetch

After `source.Fetch` returns a `WorkItem`, the caller SHALL set `WorkItem.Mode` from the matched registry entry's mode. `Fetch` SHALL NOT set `Mode` itself.

#### Scenario: GitHub issue gets ModeImplement from registry
- **WHEN** `ResolveCreateInput(456, "", "", "issue")` resolves and the work item is fetched
- **THEN** `WorkItem.Mode` is `ModeImplement` because the `github:issue` registry entry declares it, not because `issueProvider.Fetch` hardcoded it

#### Scenario: GitHub pr gets ModeReview from registry
- **WHEN** `ResolveCreateInput(456, "", "", "pr")` resolves and the work item is fetched
- **THEN** `WorkItem.Mode` is `ModeReview` because the `github:pr` registry entry declares it

### Requirement: Existing CLI behavior is preserved

The set of accepted `(flag, -t)` combinations and the error messages for invalid input SHALL remain identical to current behavior for `--github` and `--prompt`. A table test SHALL pin every valid combination and every error path before any code moves.

#### Scenario: Pinned test passes before and after the refactor
- **WHEN** `internal/cli/create_input_test.go` is run against the current code and against the refactored code
- **THEN** all cases pass in both runs (behavior is unchanged)
