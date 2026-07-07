## Why

Adding a source today requires hand-wiring four hardcoded ladders (flag vars, source-detection bools, mutual-exclusion counts, per-source validation branches) in `main.go` and `create_input.go`, and the `-t` flag is locked to GitHub. The registry (`internal/source/registry.go`) is the only clean, extensible layer — but it can't answer "what kinds does this source offer?" or "what mode does this kind map to?", forcing every upstream layer to hardcode. This change makes the registry the single source of truth for a source's `(kind, mode)` surface area, so adding a source becomes one `Register` call instead of four ladder edits, and `-t` becomes a polymorphic kind selector that works for any multi-kind source.

## What Changes

- Registry gains awareness of **mode** alongside `(system, kind)`: `Register(system, kind, mode, fetcher)`.
- Registry exposes `KindsFor(system)` so callers can ask what kinds a source offers and what mode each maps to.
- Each provider's `init()` declares its `(system, kind, mode)` at registration; the hardcoded `Mode` assignment inside each `Fetch` is removed.
- `ResolveCreateInput` stops hardcoding per-source branches: it finds the set flag, asks the registry for that source's kinds, and validates `-t` against the returned list. Single-kind sources forbid `-t`; multi-kind sources require it.
- `-t` is decoupled from GitHub: it becomes the kind selector for whichever source is selected. Adding GitLab (`-t issue|mr`) requires no resolver or flag changes beyond GitLab's own `Register` call + cobra flag.
- **No change to existing CLI behavior**: `--github`, `--jira`, `--prompt`, `-t` keep the same accepted values and error messages. A table test pinning `ResolveCreateInput` is added before any code moves.

## Capabilities

### New Capabilities
- `source-registry`: The registry's shape, what it stores per entry (`system`, `kind`, `mode`, `fetcher`), and the queries it answers (`KindsFor`, `Resolve`). Covers how providers self-register and declare their `(kind, mode)` surface area.
- `source-input-resolution`: How CLI flags (`--github`/`--jira`/`--prompt`/`-t`) resolve to a `workitem.SourceRef`. Covers source mutual-exclusion, `-t` validation against the selected source's declared kinds, and the single-kind vs multi-kind distinction.

### Modified Capabilities
<!-- No existing specs in openspec/specs/. All capabilities here are new. -->

## Impact

- **Code**: `internal/source/registry.go` (signature + new query), `internal/source/{github,jira,prompt}/*.go` (init() registration, Fetch drops hardcoded Mode), `internal/cli/create_input.go` (resolver rewritten to be registry-driven), `main.go` (callsite to `ResolveCreateInput` — flag vars and cobra declarations unchanged).
- **New tests**: `internal/cli/create_input_test.go` — table test pinning every valid combination and every error path of `ResolveCreateInput` before the refactor. No other tests added (providers' Fetch logic is unchanged; no tests exist today).
- **APIs/dependencies**: None. `go.mod` unchanged.
- **CLI surface**: Unchanged for users. `-t` accepts the same values for GitHub; the change is structural (it now validates against the registry, not a hardcoded list).
- **Future sources**: Adding GitLab, Bitbucket (multi-kind), or Linear (single-kind) becomes one `Register` call + one cobra flag + one blank import — no edits to the resolver or registry.
