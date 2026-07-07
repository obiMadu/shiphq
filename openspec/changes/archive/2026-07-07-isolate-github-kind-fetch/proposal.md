## Why

Today `internal/source/github/common.go` is an issue/pr-shaped helper pretending to be generic: it owns the `gh` invocation, the `--json` field list (with a hardcoded `if commandName == "pr"` branch for `headRefName`), the shared `viewPayload` struct, and the decode→normalize→build-WorkItem transform. Each kind file (`issue.go`, `pr.go`) controls only **one** of the ~5 things that define a `gh` invocation — the subcommand name. Adding a github kind that doesn't fit the `gh <kind> view <ref> --json <fields>` mould (e.g. `discussion` via `gh api graphql`, `release` via `gh release view`) forces edits to the shared helper, which is the opposite of the isolation the registry work unlocked upstream. This change makes each kind own its full `gh` invocation, so adding a kind — even one with a totally different command shape — touches only the new kind file.

## What Changes

- Each github kind file owns its complete `gh` invocation: the `gh` args, its own payload struct, and the `WorkItem` field mapping (including kind-specific fields like `TargetBranch`).
- `common.go` keeps only what is genuinely shared by *view-shaped* kinds: a `runView` helper that execs `gh`, decodes a `viewPayload`, and returns it; and a `baseWorkItem` helper that normalizes the reference and builds the common WorkItem fields. Both are **opt-in** — a kind that uses a non-view `gh` shape bypasses them and calls `exec.Command` directly.
- The `if commandName == "pr"` branch in `common.go` is deleted. `pr.go` declares `headRefName` in its own field list and sets `TargetBranch` itself.
- The empty-reference guard, `json.Unmarshal`, and `NormalizeIdentifier` check stay shared (via `runView` + `baseWorkItem`) because they genuinely are identical across view-shaped kinds — pure isolation that duplicates real work is dogmatic, not lazy.
- **No change to external behavior**: `gh issue view …` and `gh pr view …` produce the same `WorkItem` fields as today. The refactor is structural — who owns what inside the github package.

## Capabilities

### New Capabilities
- `github-kind-fetch`: The contract for how each github kind declares its `gh` CLI invocation (args, payload struct, WorkItem mapping) and what shared helpers (`runView`, `baseWorkItem`) are available to view-shaped kinds. Covers the ownership boundary between a kind file and `common.go`, and how a non-view-shaped kind bypasses the shared helpers.

### Modified Capabilities
<!-- No existing specs in openspec/specs/. `source-registry` and `source-input-resolution` from the prior change live only as change artifacts (not yet synced to main specs), and this change does not alter their requirements. -->

## Impact

- **Code**: `internal/source/github/common.go` (delete `fetchWithGitHubCLI` + the `pr` branch + `viewPayload`; add `runView` + `baseWorkItem`), `internal/source/github/issue.go` (owns its args + Fetch body), `internal/source/github/pr.go` (owns its args + payload + `TargetBranch` mapping).
- **No change**: `internal/source/jira/`, `internal/source/prompt/` (they don't shell out to `gh`), `internal/source/registry.go`, `internal/cli/create_input.go`, `main.go`. The registry and resolver are unaffected — this change is internal to the github provider package.
- **Future kinds**: Adding `github:discussion`, `github:release`, or any kind with a non-`view` `gh` shape becomes one new file with its own args + payload + Fetch, touching zero existing files. Adding another view-shaped kind reuses `runView`/`baseWorkItem` without extending them.
- **CLI surface**: Unchanged. `--github <n> -t issue|pr` behaves identically.
- **APIs/dependencies**: None. `go.mod` unchanged.
