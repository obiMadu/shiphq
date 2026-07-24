## 1. OpenSpec source provider (TDD)

- [x] 1.1 Write `internal/source/openspec/change_test.go` — test `Fetch` with a temp `openspec/changes/<name>/proposal.md`: valid change returns WorkItem with Identifier from NormalizeIdentifier, Title set to change name, Description containing proposal content; missing directory errors; missing proposal.md errors.
- [x] 1.2 Create `internal/source/openspec/change.go` — `changeProvider` struct, `init()` registering `source.Register("opsx", "change", workitem.ModeImplement, changeProvider{})`, `Fetch(sourceRef)` reads `{cwd}/openspec/changes/{ref}/proposal.md`, returns `workitem.WorkItem` with Source, Identifier (`NormalizeIdentifier(ref)`), Title (change name), Description (proposal content + pointer to read full change and use `openspec-apply-change` skill).
- [x] 1.3 Add blank import for `internal/source/openspec` in `internal/source/providers/providers.go`.
- [x] 1.4 Run `go test -buildvcs=false ./internal/source/openspec/...` — provider tests green.

## 2. CLI integration (TDD)

- [x] 2.1 Update `internal/cli/create_input_test.go` — add test cases for `--opsx`: no source flag error message includes `--opsx`; `--opsx` resolves to `SourceRef{System:"opsx", Kind:"change", Reference:"<name>"}`; `--opsx` with `--type` rejected; `--opsx` with `--github` rejected as multiple sources.
- [x] 2.2 Update `internal/cli/create_input.go` — add `opsxChange string` parameter to `ResolveCreateInput`, add `--opsx` to the source selection logic (mutually exclusive with `--github` and `--prompt`), update error messages to include `--opsx`. The `opsx` system has one kind (`change`), so `--type` is rejected for it.
- [x] 2.3 Run `go test -buildvcs=false ./internal/cli/...` — CLI tests green.

## 3. File move logic (TDD)

- [x] 3.1 Write `internal/source/openspec/move_test.go` — test `MoveChange(srcDir, destDir, changeName)`: untracked directory is moved from src to dest; dest already exists → skip (no error); cross-filesystem fallback (copy+remove) when `os.Rename` returns `EXDEV`; source missing → error.
- [x] 3.2 Implement `MoveChange(srcDir, destDir, changeName string) error` in `internal/source/openspec/change.go` — `os.MkdirAll` dest parent, `os.Rename` the change dir, fallback to copy+remove on `EXDEV`, skip if dest already exists.
- [x] 3.3 Run `go test -buildvcs=false ./internal/source/openspec/...` — move tests green.

## 4. Prompt builder integration

- [x] 4.1 Update `internal/prompt/builder.go` `describeWorkItem` — add case for `opsx:change`: return `fmt.Sprintf("OpenSpec change '%s'", workItem.Source.Reference)`.
- [x] 4.2 Run `go test -buildvcs=false ./internal/prompt/...` — builder tests green.

## 5. main.go wiring

- [x] 5.1 Add `opsxFlag string` var and `--opsx` flag registration on `createCmd` in `main.go` `init()`: `createCmd.Flags().StringVar(&opsxFlag, "opsx", "", "OpenSpec change name")`.
- [x] 5.2 Update `createCmdRun` in `main.go`: pass `opsxFlag` to `ResolveCreateInput`; after `runtime.Create` returns, if `workItem.Source.System == "opsx"`, capture `cwd` before `runtime.Create`, then `os.MkdirAll` the dest parent, `os.Rename` the change dir from `{cwd}/openspec/changes/{changeName}/` to `{worktreePath}/openspec/changes/{changeName}/`, skip if dest exists or files are tracked, fallback to copy+remove on `EXDEV`.
- [x] 5.3 Run `go build -buildvcs=false` — compiles clean.
- [x] 5.4 Run `go test -buildvcs=false ./...` — all tests green.

## 6. Documentation

- [x] 6.1 Update `README.md` — document `--opsx` flag in Commands and Usage sections.
- [x] 6.2 Update `skill/SKILL.md` — document `--opsx` flag and the OpenSpec source workflow.
