## MODIFIED Requirements

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

### Requirement: Existing CLI behavior is preserved

The set of accepted `(flag, -t)` combinations and the error messages for invalid input SHALL remain identical to current behavior for `--github` and `--prompt`. A table test SHALL pin every valid combination and every error path before any code moves.

#### Scenario: Pinned test passes before and after the refactor
- **WHEN** `internal/cli/create_input_test.go` is run against the current code and against the refactored code
- **THEN** all cases pass in both runs (behavior is unchanged)
