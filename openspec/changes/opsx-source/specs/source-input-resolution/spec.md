## MODIFIED Requirements

### Requirement: Exactly one source flag must be set

`ResolveCreateInput` SHALL accept at most one of the source flags (`--github`, `--prompt`, `--opsx`). If zero are set, it SHALL error "must specify --github, --opsx, or --prompt". If more than one is set, it SHALL error "must specify only one source: --github, --opsx, or --prompt". The "more than one" guard SHALL remain as defensive validation for future source flags.

#### Scenario: No source flag
- **WHEN** `ResolveCreateInput(0, "", "", "")` is called
- **THEN** it returns an error whose message contains "must specify --github, --opsx, or --prompt"

#### Scenario: Multiple source flags
- **WHEN** two source flags are set simultaneously (e.g., `--github` and `--opsx`)
- **THEN** it returns an error whose message contains "must specify only one source"

#### Scenario: OpenSpec flag resolves
- **WHEN** `ResolveCreateInput` is called with `opsxChange` set to `"add-auth"` and no other source flags
- **THEN** it returns a `SourceRef` with `System:"opsx"`, `Kind:"change"`, `Reference:"add-auth"`

#### Scenario: OpenSpec with -t rejected
- **WHEN** `ResolveCreateInput` is called with `opsxChange` set and `typeFlag` set to `"change"`
- **THEN** it returns an error indicating `--type` is not used with `--opsx`
