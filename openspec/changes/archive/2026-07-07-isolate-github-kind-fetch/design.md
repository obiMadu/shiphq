## Context

The prior change `decouple-source-kind-flag` made the registry the single source of truth for `(system, kind, mode)` and made adding a *kind to an existing system* a one-file operation. But that only democratized the upstream layers (registry, resolver, `-t` validation). Inside the `internal/source/github` package, `common.go` is still an issue/pr-shaped helper pretending to be generic: it owns the `gh` invocation, the `--json` field list (with a hardcoded `if commandName == "pr"` branch for `headRefName`), the shared `viewPayload` struct, and the full decode→normalize→build-WorkItem transform. Each kind file controls only the subcommand name — 1 of ~5 things that define a `gh` invocation. Adding a github kind that doesn't fit the `gh <kind> view <ref> --json <fields>` mould (discussion via graphql, release via `gh release view`) forces edits to shared code, which is the opposite of the isolation the registry work unlocked.

This change is scoped entirely to `internal/source/github/`. The registry, resolver, main.go flags, and the jira/prompt providers are untouched.

## Goals / Non-Goals

**Goals:**
- Each github kind file owns its full `gh` invocation: args, payload struct, WorkItem field mapping.
- Delete the `if commandName == "pr"` branch and any kind-aware logic from `common.go`.
- Keep the genuinely-shared transform (exec + decode + normalize + base-build) available to view-shaped kinds via opt-in helpers, so adding a view-shaped kind reuses them without extending them.
- Adding a non-view-shaped github kind (graphql, release) touches only the new kind file.

**Non-Goals:**
- Generalizing wtmag's own cobra flags (`--github`, `--jira`, `--prompt`) to be registry-driven — out of scope; the user explicitly accepted the current per-system flag wiring in main.go.
- Touching the jira or prompt providers — they don't shell out to `gh` and have no shared-helper problem.
- Adding new github kinds (discussion, release) — this change only restructures ownership; new kinds land as follow-ups that now require zero shared-code edits.
- Generalizing `runView` into a `runBinary(name, args)` abstraction — YAGNI until a second binary appears.

## Decisions

### Decision 1: `runView` + `baseWorkItem` as opt-in shared helpers (not a forced interface)
`common.go` exposes two unexported helpers:
- `runView(sourceRef, args) (viewPayload, error)` — execs `gh` with the kind's args, decodes JSON into the shared `viewPayload` struct, wraps exec/decode errors with the kind-aware message format.
- `baseWorkItem(sourceRef, payload) (WorkItem, error)` — normalizes the reference identifier and builds the common fields (`Source`, `Identifier`, `Title`, `Description`, `URL`).

A kind calls them in sequence; `pr.go` then sets `TargetBranch` from `payload.HeadRefName` itself. `viewPayload` stays shared because it is the shape `gh <x> view --json` returns — a kind that wants extra fields extends the struct's json tags (harmless for kinds that don't decode them) or, for a totally different shape, bypasses the helpers.

**Why not force every kind through a `Fetch` interface or a command-builder abstraction:** the only thing all github kinds share is the *binary name* (`gh`); the command shape varies. A `CommandBuilder` interface for one binary with two shapes is speculative abstraction — exactly the ladder rung we skip. The helpers are a convenience, not a contract.

**Alternative considered:** each kind calls `exec.Command` directly with zero shared helpers (pure isolation). Rejected because the exec + decode + normalize + base-build transform is ~25 genuinely identical lines across issue/pr; duplicating it is dogmatic, not lazy, and the user called this out.

### Decision 2: `viewPayload` stays in `common.go`, not per-kind
`viewPayload{Title, Body, URL, HeadRefName}` lives in `common.go` because it is the decode target for the shared `runView` helper. `issue.go` does not declare its own payload struct — it reuses `viewPayload` and ignores `HeadRefName` (empty, harmless). `pr.go` does not declare its own either — it reuses `viewPayload` and reads `HeadRefName`.

**Why not per-kind payload structs:** the only divergence today is `HeadRefName`, which is already in the shared struct. Per-kind structs would add two ~4-line structs for zero behavioral gain. If a future kind needs a field `viewPayload` doesn't have, it either adds a json tag to the shared struct (harmless) or, for a non-view shape, bypasses `runView` and declares its own. The threshold for splitting is "a kind whose payload shape differs," not "a kind with one extra field."

**Alternative considered:** each kind declares its own payload struct and calls a generic `runGh(args) ([]byte, error)` runner, doing its own `Unmarshal`. Rejected as over-rotation toward isolation — the decode is part of the shared transform, not kind-specific.

### Decision 3: `runView` hardcodes `"gh"` as the binary
The helper execs `exec.Command("gh", args...)`. A kind that needs a different binary calls `exec.Command` directly in its own file. No `runBinary(name, args)` generalization until a second binary actually shows up.

**Why:** zero current callers need anything but `gh`. Generalizing now is speculative.

### Decision 4: Refactor only, no behavior change
The `gh issue view …` and `gh pr view …` calls produce identical `WorkItem` fields before and after. No new tests for Fetch behavior (there are none today and the external `gh` dependency makes unit-testing Fetch impractical without mocking exec). The safety net is `go build` + `go vet` + the existing `create_input_test.go` (which exercises the resolver, not Fetch) + the e2e smoke from the prior change's task 7.

## Risks / Trade-offs

- **[Risk] A future view-shaped kind needs a field `viewPayload` lacks** → Mitigation: add a json tag to the shared struct; kinds that don't decode it are unaffected. The struct grows by one field, not a branch. Only when a kind's *payload shape* differs (graphql nested objects, release's `tagName`) does it bypass `runView` and declare its own struct.
- **[Risk] The empty-reference guard and `NormalizeIdentifier` check now live behind `baseWorkItem`, so a non-view kind that bypasses the helpers must re-implement them** → Mitigation: accepted. A non-view kind that shells out to `gh` still needs the same guard + normalization; duplicating ~6 lines in the rare non-view kind is cheaper than forcing it through a helper whose decode shape doesn't fit. If two non-view kinds appear, extract then.
- **[Trade-off] `viewPayload` is shared but only `pr` uses `HeadRefName`** → accepted. The struct models the `gh <x> view --json` response shape, which uniformly *can* include `headRefName`; issue simply doesn't request it. This is structural truth about the `gh` CLI, not kind-specific leakage.
- **[Risk] No unit tests for Fetch** → Mitigation: the refactor is mechanical (move code, not change it). `go build`/`go vet` catch type errors; the e2e smoke (`wtmag create --github <n> -t issue|pr`) confirms the real `gh` invocation still works end-to-end.
