## 1. Characterization: pin ResolveCreateInput behavior (GREEN — pins existing behavior)

- [x] 1.1 Create `internal/cli/create_input_test.go` with a table test covering every valid combination: `--github 456 -t issue`, `--github 456 -t pr`, `--jira PROJ-123`, `--prompt "do something"` — assert `SourceRef` (System, Kind, Reference) and `PromptOverride` for each
- [x] 1.2 Add the `--prompt` dual-purpose cases: `--github 456 -t issue --prompt "override"` → SourceRef is github, PromptOverride is "override"; `--jira PROJ-123 --prompt "override"` → SourceRef is jira, PromptOverride is "override" (prompt is an override, not a source, when github or jira is also set)
- [x] 1.3 Add error-path cases: no source flag (assert "must specify --github, --jira, or --prompt"), `--github` + `--jira` together (assert "must specify only one source"), `--github` without `-t` (assert "--type (-t) is required for GitHub"), `--github -t mr` (assert error names issue and pr), `--jira -t ticket` (assert "--type is not used with --jira"), `--prompt -t prompt` (assert "--type can only be used with --github"), `--github -1` (assert "must be a positive number")
- [x] 1.4 Run `go test ./internal/cli/...` against unchanged code, confirm all cases GREEN (this pins behavior the refactor must preserve)

## 2. Registry: KindsFor query (RED → GREEN)

- [x] 2.1 Create `internal/source/registry_test.go` with a failing test: register test providers via `source.Register("github", "issue", workitem.ModeImplement, fakeProvider{})` and `source.Register("github", "pr", workitem.ModeReview, fakeProvider{})`, call `source.KindsFor("github")`, assert two entries with correct Kind and Mode; call `KindsFor("nonexistent")`, assert empty slice — RED (Register doesn't accept mode, KindsFor doesn't exist)
- [x] 2.2 Run test, confirm RED (compile failure: Register signature mismatch, KindsFor undefined)
- [x] 2.3 In `internal/source/registry.go`: add `mode workitem.WorkMode` field to the internal entry struct, change `Register` signature to `Register(system, kind string, mode workitem.WorkMode, provider Provider)`, implement `KindsFor(system string) []Entry` returning `{Kind, Mode, Fetcher}` per entry (empty slice for unknown system)
- [x] 2.4 Update all four provider `init()` calls to pass mode: `github/issue.go` → `ModeImplement`, `github/pr.go` → `ModeReview`, `jira/ticket.go` → `ModeImplement`, `prompt/prompt.go` → `ModeImplement`
- [x] 2.5 Run `go build ./...`, confirm it compiles
- [x] 2.6 Run `go test ./internal/source/...`, confirm KindsFor test is GREEN
- [x] 2.7 Run `go test ./internal/cli/...`, confirm characterization test still GREEN (resolver not yet changed)

## 3. Registry: Resolve query (RED → GREEN)

- [x] 3.1 Add a failing test to `internal/source/registry_test.go`: call `source.Resolve("github", "pr")`, assert returned fetcher and `ModeReview`; call `Resolve("github", "mr")`, assert error — RED (Resolve doesn't exist)
- [x] 3.2 Run test, confirm RED
- [x] 3.3 Implement `Resolve(system, kind string) (Provider, workitem.WorkMode, error)` in `registry.go` — returns fetcher and mode for a registered `(system, kind)`, error for unregistered
- [x] 3.4 Run `go test ./internal/source/...`, confirm GREEN

## 4. Resolver: registry-driven with Mode (RED → GREEN)

- [x] 4.1 Write a failing test in `internal/cli/create_input_test.go`: for each valid combination, assert `CreateInput.Mode` is correct (`--github -t issue` → `ModeImplement`, `--github -t pr` → `ModeReview`, `--jira` → `ModeImplement`, `--prompt` → `ModeImplement`) — RED (`CreateInput` has no `Mode` field)
- [x] 4.2 Run test, confirm RED (compile failure: no Mode field on CreateInput)
- [x] 4.3 Add test provider registration to the test file (via `TestMain` or `init`): register `github:issue` (Implement), `github:pr` (Review), `jira:ticket` (Implement), `prompt:prompt` (Implement) using a `fakeProvider` — do NOT import `internal/source/providers` (would cause duplicate-registration panic); this setup is required because the rewritten resolver calls `registry.KindsFor`
- [x] 4.4 Add `Mode workitem.WorkMode` field to `CreateInput` in `internal/cli/create_input.go`
- [x] 4.5 Rewrite `ResolveCreateInput`: delete `hasGitHubSource`/`hasJiraSource`/`hasPromptSource` bools and the per-source branch ladder; replace with source detection via a small `{flagValue → systemName}` table, call `registry.KindsFor(system)` for the selected source; if `len(kinds) == 1` forbid `-t` and use the single kind; if `len(kinds) > 1` require `-t` and validate against the returned kinds; set `CreateInput.Mode` from the matched entry; preserve `--prompt` dual-purpose (prompt is override, not source, when github or jira is also set) and mutual exclusion (github + jira = error); error messages must match the pinned strings from task 1.3
- [x] 4.6 Run `go test ./internal/cli/...`, confirm both characterization test (task 1) and Mode test (task 4.1) are GREEN
- [x] 4.7 Run `go vet ./...`, confirm clean

## 5. Providers: delete Mode from Fetch (REFACTOR — stay green)

- [x] 5.1 Delete the `Mode: workitem.ModeImplement` (or `ModeReview`) line from each provider's `Fetch` return in `github/issue.go`, `github/pr.go`, `jira/ticket.go`, `prompt/prompt.go` — `Fetch` now returns `WorkItem` with Mode as zero value; the caller stamps it
- [x] 5.2 Run `go test ./...`, confirm all tests still GREEN (characterization test asserts `CreateInput.Mode`, not `WorkItem.Mode`; registry tests assert registry-stored mode)

## 6. main.go: stamp WorkItem.Mode from CreateInput (REFACTOR)

- [x] 6.1 In `main.go` `createCmdRun`, after `source.Fetch(createInput.SourceRef)` returns the `WorkItem`, add `workItem.Mode = createInput.Mode` so the work item carries the mode from the registry (via the resolver) rather than from `Fetch`
- [x] 6.2 Run `go build ./...`, confirm it compiles
- [x] 6.3 Run `go test ./...`, confirm all tests GREEN

## 7. End-to-end verification

- [x] 7.1 Build the binary (`go build .`), run `wtmag create --github <real-issue-number> -t issue` against a test repo, confirm the worker is created in implement mode
- [x] 7.2 Run `wtmag create --github <real-pr-number> -t pr`, confirm the worker is created in review mode
- [x] 7.3 Run `wtmag create --jira PROJ-123` and `wtmag create --prompt "do something"`, confirm both resolve correctly and reject stray `-t` with the pinned error messages
- [x] 7.4 Run `wtmag create --github <number> -t issue --prompt "custom override"`, confirm the prompt override is applied
- [x] 7.5 Confirm `--help` output for `-t` is still sensible (help text may optionally be generalized beyond GitHub, but the flag's accepted values are unchanged)
