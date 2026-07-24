## Why

All prompt text is hard-coded in `internal/prompt/builder.go` as Go string literals. Users cannot customize prompts without rebuilding the binary. Per-source customization is impossible — GitHub issues and custom prompts use the same frame. The PR/delivery instruction ("commit, push, open a PR") is always injected for every implementation task with no way to opt out. The `BuildDefault`/`BuildOverride` split is redundant (audit finding #7) and should collapse into one function.

## What Changes

- Extract all prompt strings into template files using Go `text/template` (stdlib, no new dependency). Three template types: `implement`, `review`, and `pr` (renamed from "delivery").
- Ship 12 embedded template files: a generic default plus host-specific overrides for GitHub, GitLab, and Bitbucket, for each of the three types. The generic `pr.tmpl` tells the agent to detect the git remote host and use the matching CLI tool; host-specific templates bake in the concrete tool name (`gh`, `glab`, etc.).
- Templates are overridable from disk. Resolution per template type: `{type}-{host}.tmpl` (project `./wtmag-prompts/`) → `{type}-{host}.tmpl` (global `~/.config/wtmag/prompts/`) → `{type}.tmpl` (project) → `{type}.tmpl` (global) → embedded default.
- **BREAKING**: Add `--pr` flag to `create`. The PR instruction ("commit, push, open a PR") is only injected when `--pr` is passed on an implementation task. Without `--pr`, implementation tasks get the work item context with no delivery instruction. Passing `--pr` on a review task (`github:pr`) errors.
- Collapse `BuildDefault` and `BuildOverride` into a single `Build(workItem, repoTarget, opts)` function. `opts` carries the override text and the `--pr` flag.
- The reference backlink note (e.g., "Refs #456" for GitHub issues on GitHub host) is resolved in Go and passed to the `pr` template as a data field, not embedded in the template text.

## Capabilities

### New Capabilities
- `prompt-templates`: Template-based prompt building with per-host overrides, a `--pr` flag for opt-in PR instructions, and a layered file-based override system (project + global + embedded defaults).

### Modified Capabilities
<!-- None — the prompt builder is not currently covered by any existing spec. -->

## Impact

- **Code**: `internal/prompt/builder.go` (rewritten), `internal/prompt/templates/` (new embedded template files), `main.go` (new `--pr` flag, updated call to `Build`), `internal/cli/create_input.go` (`--pr` flag passed through). No new dependencies.
- **CLI surface**: New `--pr` flag on `create`. Existing implementation behavior changes: delivery instruction is no longer injected by default.
- **Config**: New `~/.config/wtmag/prompts/` and `./wtmag-prompts/` directories for user overrides. No config file changes.
- **Dependencies**: none added or removed. `text/template` is stdlib.
