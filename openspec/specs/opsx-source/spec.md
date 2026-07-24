# opsx-source

## Purpose

OpenSpec change as a source provider — reads the proposal, builds the work item, and cuts the change files into the worktree after creation.

## Requirements

### Requirement: OpenSpec change provider reads proposal from cwd

The `opsx:change` provider SHALL read `openspec/changes/<name>/proposal.md` from the current working directory. The provider SHALL error if the directory or `proposal.md` does not exist. The `WorkItem.Identifier` SHALL be `workitem.NormalizeIdentifier(changeName)`. The `WorkItem.Title` SHALL be the change name. The `WorkItem.Description` SHALL be the proposal content followed by a pointer to read the full change in `openspec/changes/<name>/` and use the `openspec-apply-change` skill to implement it.

#### Scenario: Valid change directory
- **WHEN** `Fetch` is called with `SourceRef{System:"opsx", Kind:"change", Reference:"add-auth"}` and `openspec/changes/add-auth/proposal.md` exists in `cwd`
- **THEN** it returns a `WorkItem` with `Identifier` from `NormalizeIdentifier("add-auth")`, `Title` set to `"add-auth"`, and `Description` containing the proposal content

#### Scenario: Missing change directory
- **WHEN** `Fetch` is called with `SourceRef{System:"opsx", Kind:"change", Reference:"nonexistent"}` and `openspec/changes/nonexistent/` does not exist
- **THEN** it returns an error indicating the change directory was not found

#### Scenario: Missing proposal file
- **WHEN** `Fetch` is called with `SourceRef{System:"opsx", Kind:"change", Reference:"add-auth"}` and the directory exists but `proposal.md` is missing
- **THEN** it returns an error indicating `proposal.md` was not found

### Requirement: OpenSpec change provider registers as ModeImplement

The `opsx:change` provider SHALL register with `workitem.ModeImplement`. The provider SHALL self-register via `init()` and be wired by a blank import in `internal/source/providers/providers.go`.

#### Scenario: Provider registration
- **WHEN** the `openspec` provider package is blank-imported
- **THEN** `source.KindsFor("opsx")` returns one entry: `{Kind:"change", Mode:ModeImplement}`

### Requirement: OpenSpec change files are moved to the worktree after creation

After `runtime.Create` returns, if the work item's source system is `opsx`, the change directory at `{cwd}/openspec/changes/{changeName}/` SHALL be moved to `{worktreePath}/openspec/changes/{changeName}/`. The move SHALL be a rename (cut), not a copy. If the destination already exists, the move SHALL be skipped. If the change files are tracked by git (`git ls-files` returns non-empty), the move SHALL be skipped.

#### Scenario: Uncommitted change moved to worktree
- **WHEN** the change directory exists in `cwd` and is untracked, and the worktree has been created
- **THEN** the directory is moved from `{cwd}/openspec/changes/{changeName}/` to `{worktreePath}/openspec/changes/{changeName}/`

#### Scenario: Committed change skipped
- **WHEN** the change files are tracked by git on the base branch
- **THEN** the move is skipped because the worktree already contains the files

#### Scenario: Cross-filesystem fallback
- **WHEN** `os.Rename` fails with a cross-device error
- **THEN** the move falls back to copy-then-remove so the files land in the worktree regardless of filesystem layout

### Requirement: Source label for OpenSpec changes

`describeWorkItem` SHALL return `"OpenSpec change '<name>'"` for `opsx:change` work items, where `<name>` is the change name (the `SourceRef.Reference`). The label SHALL NOT use the short form `opsx`.

#### Scenario: Label for OpenSpec change
- **WHEN** `describeWorkItem` is called with a work item whose source is `opsx:change` with reference `add-auth`
- **THEN** it returns ` "OpenSpec change 'add-auth'"`
