## Context

wtmag has two source providers: `github` (issue, pr) and `prompt`. Users brainstorm with OpenSpec on their dev branch, writing changes in `openspec/changes/<name>/` (proposal, design, tasks, spec deltas). They then want to hand those changes to wtmag to implement in an isolated worktree. Currently there is no `opsx` source and no mechanism to move uncommitted OpenSpec change files from the working directory into the new worktree.

The worktree is created by `runtime.Create`, which delegates to `wt switch --create` and discovers the worktree path via `git worktree list`. The worktree path can be anywhere — it depends on the user's `wt` `worktree-path` template. The source files live at `cwd/openspec/changes/<name>/`, wherever `cwd` happens to be.

## Goals / Non-Goals

**Goals:**
- Add `--opsx <change-name>` flag to `wtmag create` that reads an OpenSpec change and builds a work item from its `proposal.md`.
- Move the change directory from `cwd` into the new worktree after creation, so the agent has the full change (design, tasks, specs) to read.
- The move is a cut (rename), not a copy — the change files leave the working directory and land in the worktree.
- The move works regardless of worktree layout — source is `cwd`, destination is the worktree path returned by `runtime.Create`.

**Non-Goals:**
- Committing the OpenSpec change on the base branch before creating the worktree. The user handles git.
- Splitting `runtime.Create` into separate "create worktree" and "launch agent" phases. The move happens after `runtime.Create` returns.
- Task checkbox tracking in `tasks.md`. The agent uses the `openspec-apply-change` skill, which handles that.
- Defaulting `--pr` on for OpenSpec changes. `--pr` remains opt-in.

## Decisions

### 1. Flag name: `--opsx`, system: `opsx`, kind: `change`

The CLI flag is `--opsx` (short form) to keep branch names manageable: `opsx-change-<identifier>`. Inside the codebase, the provider package is `openspec` (descriptive), but `SourceRef.System` is `opsx` (matches the flag and branch prefix). `describeWorkItem` returns the human-readable `"OpenSpec change '<name>'"`.

### 2. Provider reads from `cwd`, not from a project root

The provider's `Fetch` reads `openspec/changes/<name>/` relative to `os.Getwd()`. This is correct because OpenSpec writes files to wherever the user is working, and `wtmag create` is run from that same directory. No project root detection needed — `cwd` is the source of truth.

### 3. File move happens after `runtime.Create` returns

`runtime.Create` creates the worktree, launches the agent in tmux, and returns the worktree path. The move (`os.Rename`) runs immediately after, before the agent has finished starting up. The agent takes 1-2 seconds to launch, read `.wtmag/prompt.md`, and look for `openspec/changes/`. The `os.Rename` is instant (milliseconds). The race is negligible.

**Alternative considered:** Split `runtime.Create` into "create worktree" + "launch agent" so the move happens between them. Rejected — it is a bigger refactor justified by one source, and the race is practically zero.

### 4. Move is a cut (rename), not a copy

The change files leave `cwd` and land in the worktree. This is the user's intent: "send it out to implement." If the files stayed in `cwd`, the user would have two copies and would need to manually clean up.

### 5. Skip the move if files are tracked by git

If `git ls-files openspec/changes/<name>/` returns non-empty, the files are committed on the base branch and the new worktree already has them. The move is skipped. This handles the case where the user committed the OpenSpec change before running `wtmag create`.

### 6. Cross-filesystem fallback

If `os.Rename` fails with `EXDEV` (source and destination on different filesystems), fall back to `cp -r` + `rm -rf`. This handles the case where the worktree is on a different filesystem from `cwd` (e.g., a tmpfs, a separate partition). Same-filesystem rename is the fast path.

### 7. Prompt content: proposal + pointer

The work item's `Description` is the proposal content plus a pointer to read the full change in the worktree and use the `openspec-apply-change` skill. The `implement` template renders this as context. The agent reads the actual files (design.md, tasks.md, specs/) from the worktree.

## Risks / Trade-offs

- **Agent race**: The agent launches before the file move completes. Mitigated by the move being instant (same-filesystem rename). If the agent somehow reads the directory before the move, it would see no OpenSpec files — but the agent takes seconds to start and the move takes milliseconds.
- **Move failure**: If `os.Rename` fails (permissions, disk full), `createCmdRun` returns the error. The user runs `wtmag cleanup <id>` to remove the worktree and session. Not auto-rolled back inside `createCmdRun` because the agent is already launched.
- **Untracked files left behind**: If OpenSpec wrote files outside `openspec/changes/<name>/` (e.g., `openspec/specs/` for a synced spec), those are not moved. The move is scoped to the change directory only. This is correct — synced specs are committed and already in the worktree.
