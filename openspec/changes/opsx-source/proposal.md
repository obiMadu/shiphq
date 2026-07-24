## Why

Users brainstorm with OpenSpec on their dev branch — writing proposals, designs, and task breakdowns — then want to hand those changes to wtmag to implement in an isolated worktree. Currently the only way to do this is to copy the OpenSpec change files into the worktree manually. wtmag has no source provider that reads OpenSpec changes, and no mechanism to move uncommitted change files from the working directory into the newly created worktree.

## What Changes

- Add a new source provider `opsx` that reads an OpenSpec change from `openspec/changes/<name>/` in the current working directory. The provider reads `proposal.md` as the work item context and instructs the agent to read the full change (design, tasks, specs) in the worktree and use the `openspec-apply-change` skill to implement it.
- Add `--opsx <change-name>` flag to `wtmag create`. Short form for the CLI surface; the provider system is `opsx`, kind is `change`, producing branch names like `opsx-change-<identifier>`.
- After worktree creation, if the source is `opsx`, move the change directory from the current working directory to the new worktree path. This is a cut (rename), not a copy — the change files leave dev and land in the worktree. If the files are already tracked by git (committed on the base branch), the move is skipped since the worktree already has them.
- The move is filesystem-agnostic: same-filesystem rename is instant; cross-filesystem falls back to copy-then-remove.
- The move does not assume any particular worktree layout — source is always `cwd`, destination is the worktree path returned by `runtime.Create`, wherever `wt` placed it.

## Capabilities

### New Capabilities
- `opsx-source`: OpenSpec change as a source provider — reads the proposal, builds the work item, and cuts the change files into the worktree after creation.

### Modified Capabilities
- `source-input-resolution`: Add `--opsx` as a third source flag, mutually exclusive with `--github` and `--prompt`.
- `source-registry`: Register the `opsx:change` provider.

## Impact

- **Code**: `internal/source/openspec/change.go` (new provider), `internal/source/providers/providers.go` (blank import), `internal/cli/create_input.go` (new `--opsx` parameter), `main.go` (new flag, file move after `runtime.Create`), `internal/prompt/builder.go` (`describeWorkItem` case for `opsx:change`).
- **CLI surface**: New `--opsx <change-name>` flag on `create`.
- **Dependencies**: none added or removed.
