## Why

A whole-repo over-engineering audit surfaced three cuts to ship together: a duplicated registry lookup, a dead config-path alias, and a `--jira` surface (flag, resolver branch, test fixtures, spec scenarios) for an adapter that was never implemented and whose stub was already removed. Carrying dead code and dead spec scenarios taxes every future read of these files.

## What Changes

- `source.Fetch` delegates to `source.Resolve` instead of redoing the `providers` map lookup and error string inline. One source of truth for "is this `(system, kind)` registered".
- Remove `config.GetConfigPath`, a one-line alias for `GetGlobalConfigPath` with a single in-package caller (`EnsureConfigFile`); inline that caller to use `GetGlobalConfigPath` directly.
- Remove the `--jira` CLI flag from `main.go` and the `jiraTicketID` parameter + jira branch from `cli.ResolveCreateInput`. The flag never produced a worker: the jira provider's `Fetch` always returned "not yet implemented", and the provider package was deleted in the prior audit step. Error messages that listed `--jira` now list `--github` and `--prompt` only.
- Remove jira registration from test fixtures in `internal/cli/create_input_test.go` and `internal/source/registry_test.go`, and drop the jira-specific test cases.
- Update specs: drop `jira:ticket` scenarios from `source-registry` and `source-input-resolution`, and remove `--jira` from the accepted-flags contract and error messages.

## Capabilities

### New Capabilities
<!-- None -->

### Modified Capabilities
- `source-registry`: Remove the `jira:ticket` single-kind lookup scenario; restate the single-kind scenario against the `prompt:prompt` entry that still exists. No requirement is added or removed — the registry contract is unchanged, only the illustrative scenario's example system is swapped off a kind that no longer registers.
- `source-input-resolution`: Remove `--jira` from the accepted source flags, error messages, and scenarios. The "exactly one source flag" and "single-kind sources forbid -t" requirements now cover `--github` and `--prompt` only.

## Impact

- **Code**: `internal/source/registry.go`, `internal/config/paths.go`, `main.go`, `internal/cli/create_input.go`, `internal/cli/create_input_test.go`, `internal/source/registry_test.go`. No new files, no new packages, no dependency changes.
- **CLI surface**: `--jira` flag is removed. Since the flag never produced a worker (provider always errored), no working workflow breaks. Users who previously saw the "not yet implemented" error will now see cobra's "unknown flag: --jira" at parse time.
- **Specs**: `openspec/specs/source-registry/spec.md` and `openspec/specs/source-input-resolution/spec.md` updated via delta specs in this change.
- **Dependencies**: none added or removed.
