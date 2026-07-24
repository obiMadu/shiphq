## 1. Registry: Fetch delegates to Resolve (refactor, no behavior change)

- [x] 1.1 Rewrite `Fetch` in `internal/source/registry.go` to delegate to `Resolve`: look up via `Resolve(sourceRef.System, sourceRef.Kind)`, return `provider.Fetch(sourceRef)`. Remove the inline map lookup and duplicated error string from `Fetch`.
- [x] 1.2 Run `go test -buildvcs=false ./internal/source/...` — existing registry tests stay green (behavior unchanged).

## 2. Config: remove GetConfigPath alias (refactor, no behavior change)

- [x] 2.1 Delete the `GetConfigPath` function from `internal/config/paths.go`.
- [x] 2.2 Update `EnsureConfigFile` in the same file to call `GetGlobalConfigPath()` directly instead of `GetConfigPath()`.
- [x] 2.3 Run `go build -buildvcs=false` — confirm no remaining references to `GetConfigPath`.

## 3. CLI: remove --jira surface (behavior change — tests pin target first)

- [x] 3.1 Update `internal/cli/create_input_test.go`: remove the `jira` fake registration from `TestMain`; remove the "jira ticket" and "jira with prompt override" valid cases and the "jira with -t" error case; update every `ResolveCreateInput` call site to the new 3-arg signature (drop `jiraTicketID`); update expected error substrings from "--github, --jira, or --prompt" to "--github or --prompt".
- [x] 3.2 Update `internal/source/registry_test.go`: remove the `jira` fake registration from `TestMain`.
- [x] 3.3 Remove `jiraFlag` and its `--jira` flag registration from `main.go`; update the `createCmdRun` call to `ResolveCreateInput` to the new 3-arg signature.
- [x] 3.4 Remove the `jiraTicketID` parameter and the jira `sourceSelection` branch from `ResolveCreateInput` in `internal/cli/create_input.go`; update the two error messages to "--github or --prompt". Keep the `len(selected) > 1` guard as defensive validation; update its message to "--github or --prompt".
- [x] 3.5 Run `go test -buildvcs=false ./internal/cli/... ./internal/source/...` — all tests green.

## 4. Verify

- [x] 4.1 Run `go test -buildvcs=false ./...` — full suite green.
- [x] 4.2 Run `go build -buildvcs=false` — binary builds with no jira references.
- [x] 4.3 Run `openspec status --change lean-cleanup-registry-config-jira` — all artifacts done, ready to verify and archive.
