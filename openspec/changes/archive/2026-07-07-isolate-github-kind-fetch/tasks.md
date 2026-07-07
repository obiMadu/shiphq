## 1. Characterization: establish GREEN baseline (pins current Fetch behavior)

- [x] 1.1 Run `go build -buildvcs=false ./...` and `go vet -buildvcs=false ./...`, confirm clean (baseline the current compiles)
- [x] 1.2 Run `go test -buildvcs=false ./...`, confirm all existing tests GREEN (the `create_input_test.go` + `registry_test.go` from the prior change stay green throughout this refactor — they are the regression net for upstream layers this change does NOT touch)
- [x] 1.3 E2e smoke baseline: run `wtmag create --github <issue-number> -t issue` and `wtmag create --github <pr-number> -t pr` against the project repo, confirm both create a worker; record the `WorkItem` shape each produces (Title, Description, URL, and for pr TargetBranch) — this is the characterization the refactor must preserve. Clean up the workers afterward. (No unit test for Fetch: it shells out to the external `gh` binary and the arg slices are trivial literals; mocking exec or extracting arg-builder functions would be speculative infrastructure for a mechanical move — ponytail: trivial one-liners need no test)

## 2. common.go: replace fetchWithGitHubCLI with runView + baseWorkItem (REFACTOR)

- [x] 2.1 In `internal/source/github/common.go`: delete `fetchWithGitHubCLI` and the `if commandName == "pr"` field-list branch. Keep `viewPayload{Title, Body, URL, HeadRefName}`. Add `runView(sourceRef workitem.SourceRef, args []string) (viewPayload, error)` — execs `exec.Command("gh", args...)`, captures `CombinedOutput`, wraps exec error with `failed to fetch GitHub <kind> <ref>: <err>\n<output>`, `json.Unmarshal`s output into `viewPayload`, wraps decode error with `failed to decode GitHub <kind> <ref>: <err>`, returns payload. Add `baseWorkItem(sourceRef workitem.SourceRef, payload viewPayload) (workitem.WorkItem, error)` — calls `workitem.NormalizeIdentifier(sourceRef.Reference)`, errors with `invalid GitHub <kind> reference: <ref>` if empty, returns `WorkItem{Source, Identifier, Title (trimmed), Description (trimmed), URL (trimmed)}`. No `if`/`switch` on kind or commandName may remain (spec: No kind-aware branches in shared code)
- [x] 2.2 Run `go build -buildvcs=false ./...` — expect compile failure (issue.go/pr.go still call the deleted `fetchWithGitHubCLI`); this confirms the old helper is gone before rewiring the kinds

## 3. issue.go: own its gh invocation (REFACTOR — stay green)

- [x] 3.1 In `internal/source/github/issue.go`: rewrite `Fetch` to call `runView(sourceRef, []string{"issue", "view", sourceRef.Reference, "--json", "title,body,url"})`, keep the empty-reference guard (`GitHub <kind> reference cannot be empty`), then `baseWorkItem(sourceRef, payload)` and return its result. The arg slice and the choice to not set `TargetBranch` now live in `issue.go`. No payload struct declared in `issue.go` (reuses shared `viewPayload`)
- [x] 3.2 Run `go build -buildvcs=false ./...`, confirm issue.go compiles against the new helpers

## 4. pr.go: own its gh invocation + TargetBranch mapping (REFACTOR — stay green)

- [x] 4.1 In `internal/source/github/pr.go`: rewrite `Fetch` to call `runView(sourceRef, []string{"pr", "view", sourceRef.Reference, "--json", "title,body,url,headRefName"})`, keep the empty-reference guard, then `baseWorkItem(sourceRef, payload)`, then set `item.TargetBranch = strings.TrimSpace(payload.HeadRefName)` and return `item`. The `headRefName` field list and the `TargetBranch` mapping now live in `pr.go` (the deleted `if commandName == "pr"` branch's behavior, relocated to its owner)
- [x] 4.2 Run `go build -buildvcs=false ./...` and `go vet -buildvcs=false ./...`, confirm clean

## 5. Verify: behavior preserved end-to-end

- [x] 5.1 Run `go test -buildvcs=false ./...`, confirm all existing tests still GREEN (no upstream regression)
- [x] 5.2 Re-run the e2e smoke from task 1.3 with the same issue and pr numbers: confirm identical worker creation and the same `WorkItem` shape (Title, Description, URL, and TargetBranch for pr). Clean up workers. This confirms the refactor preserved Fetch behavior
- [x] 5.3 Inspect `internal/source/github/common.go`: confirm it contains no `if`/`switch` keyed on `kind` or `commandName` (the only `sourceRef.Kind` references are in error-message formatting, per spec scenario "common.go is kind-agnostic")
