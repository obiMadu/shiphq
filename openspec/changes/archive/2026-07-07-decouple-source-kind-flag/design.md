## Context

wtmag resolves task sources through two layers:

1. **Registry** (`internal/source/registry.go`): a `map[string]Provider` keyed by `system:kind`. Providers self-register via `init()`. Clean, but stores no metadata beyond the fetcher.
2. **CLI resolver** (`internal/cli/create_input.go`): hand-coded per-source branches (`hasGitHubSource`, `hasJiraSource`, `hasPromptSource`), a mutual-exclusion counter, and a hardcoded validation ladder where `-t` is validated against GitHub's `{issue, pr}` set and explicitly forbidden for jira/prompt.

`workitem.WorkMode` (Implement/Review) is currently baked into each provider's `Fetch` (`github/issue.go` returns `ModeImplement`, `github/pr.go` returns `ModeReview`, etc.). The registry key `system:kind` cannot answer "what mode does this kind map to?" — callers can't query it, so they hardcode.

There are **no tests** in the repo today.

The two shapes a source can take:

- **Single-kind**: jira (`ticket`), prompt (`prompt`), linear (`issue`). Kind is implicit; `-t` forbidden.
- **Multi-kind**: github (`issue`, `pr`), gitlab (`issue`, `mr`), bitbucket (`issue`, `pr`). Kind is a second flag; `-t` required.

Today `-t` is GitHub-locked. Adding another multi-kind source forces a decision the codebase hasn't made.

## Goals / Non-Goals

**Goals:**
- Registry becomes the single source of truth for a source's `(kind, mode)` surface area.
- Adding a source (single or multi-kind) requires one `Register` call + one cobra flag + one blank import — no edits to the resolver or registry.
- `-t` is decoupled from GitHub: it validates against whichever source is selected, via the registry.
- Existing CLI behavior (`--github`, `--jira`, `--prompt`, `-t` accepted values, error messages) is preserved.
- A table test pins `ResolveCreateInput` behavior before any code moves.

**Non-Goals:**
- Dynamic CLI flag generation in `main.go` (the per-source `var` + cobra flag ladder stays). Adding a source still requires declaring its flag. This dies when a second multi-kind source lands and the 3-line tax actually repeats.
- Generalizing the registry key to `system:kind:mode` (allowing the same kind to map to both modes). No consumer needs it; `KindSpec` would not preclude it but it is not built.
- Adding new sources (GitLab, Bitbucket, Linear). This change only opens the door; adding them is separate work.
- Refactoring provider `Fetch` bodies. Their internal logic is unchanged; only their `init()` registration and the `Mode` line in `Fetch` move.
- A full test suite. Only the logic that moves (`ResolveCreateInput`) gets pinned.

## Decisions

### Decision 1: `Register` gains a `mode` parameter; no `SourceSpec`/`KindSpec` types

**Choice:** `Register(system, kind string, mode workitem.WorkMode, provider Provider)`. Add a query `KindsFor(system string) []entry` where `entry` is a small struct `{ Kind string; Mode workitem.WorkMode; Fetcher Provider }` (package-internal, not exported as a named type if not needed).

**Rejected — `SourceSpec`/`KindSpec` types with an `Arity` enum:** Flat registry entries answer every question the resolver asks. `Arity` is derivable (`len(KindsFor(system)) == 1`). SourceSpec adds a type hierarchy for metadata no consumer declares yet.

**Rationale:** The resolver needs two things: "what kinds does this source have?" and "what mode does this kind map to?". Both are answerable from a flat `(system, kind, mode, fetcher)` entry. Adding types for one consumer is the abstraction ponytail avoids.

### Decision 2: `Mode` moves from `Fetch` into the `Register` call

**Choice:** Each provider's `init()` passes its mode at registration. `Fetch` no longer sets `WorkItem.Mode` — the caller (resolver path) sets it from the registry entry after `Fetch` returns, OR `Fetch` receives the mode and sets it. The minimal version: the resolver sets `Mode` on the `WorkItem` after `source.Fetch`, since the resolver now knows the mode from the registry lookup.

**Rejected — `Fetch` keeps setting Mode:** Defeats the point. The registry would store mode but `Fetch` would override it, creating two sources of truth.

**Rejected — `Fetch` signature gains a mode param:** Touches every provider's `Fetch` signature for a value the registry already has. The resolver already has the mode from `KindsFor`/`Resolve`; it can stamp it on the returned `WorkItem`.

**Rationale:** One source of truth (registry). Smallest diff to providers (delete one line, change `Register` call). `Fetch` returns a `WorkItem` with `Mode` unset; the caller fills it.

### Decision 3: Resolver becomes registry-driven; per-source branches deleted

**Choice:** `ResolveCreateInput` is rewritten. It iterates the known source flags, finds which is set, calls `registry.KindsFor(system)`, and applies one rule:

- `len(kinds) == 1` → `-t` forbidden, kind is the only entry's kind.
- `len(kinds) > 1` → `-t` required, validated against `kinds`. Matched entry's mode is used.

Mutual exclusion (only one source flag set) is preserved via the same counter approach.

**Rejected — keep per-source branches but read mode from registry:** Leaves the ladder intact; adding a source still requires a new branch. Doesn't achieve the goal.

**Rationale:** The branch ladder is the friction. Replacing it with one `len(kinds)` check is the entire unlock. The same code path handles GitHub (multi) and jira/prompt (single) without naming them.

### Decision 4: Test only `ResolveCreateInput`, before anything moves

**Choice:** One table test in `internal/cli/create_input_test.go` covering every valid `(flag, -t)` combination and every error path. Written first, against current code, pinning current behavior. The refactor must keep it green.

**Rejected — full characterization suite (registry, providers, prompt builder):** Providers' `Fetch` logic isn't moving. The prompt builder isn't touched. The registry gains a field and a query but its existing `Register`/`Fetch` semantics are unchanged. Testing them is scaffolding for code that isn't changing.

**Rationale:** `ResolveCreateInput` is the only non-trivial logic being rewritten. Pinning it is the smallest check that fails if the refactor breaks behavior.

### Decision 5: `main.go` flag ladder stays

**Choice:** `var githubFlag int` / `var jiraFlag string` / `var promptFlag string` and their cobra declarations remain hand-coded. The `ResolveCreateInput` callsite signature is unchanged (it still receives the flag values).

**Rejected — build cobra flags dynamically from `registry.All()`:** Zero future sources exist. The per-source flag tax is 3 lines/source and not painful yet. Dynamic generation adds complexity (flag value types differ: `int` for github, `string` for jira/prompt) for no current consumer.

**Rationale:** Ponytail doesn't build generators for zero consumers. The ladder dies the day GitLab lands and the tax repeats — *then* it's felt, *then* it's built.

## Risks / Trade-offs

- **[Risk] No test net beyond `ResolveCreateInput`** → The pinning test covers the logic that moves. Provider `Fetch` bodies and the prompt builder are untouched, so their (untested) behavior is no riskier than today. If a provider's `Fetch` is later restructured, that's when its tests get written.
- **[Risk] Resolver rewrite could change error message wording** → The pinning test asserts on error strings. If the rewrite produces different wording, the test fails and the wording is reconciled (either update the test if the new wording is acceptable, or restore the old wording). This forces an explicit decision rather than silent drift.
- **[Risk] `KindsFor` returns entries in registration order; resolver picks "first" for single-kind** → For single-kind sources there is exactly one entry, so order is irrelevant. For multi-kind sources the resolver matches by `-t` value, not by index. No order dependence.
- **[Trade-off] Adding a source still needs a `main.go` flag edit** → Accepted. The four-ladder tax becomes a three-step tax (flag, Register, import). The remaining flag step is irreducible until dynamic generation is justified by a real second consumer.
- **[Trade-off] `Fetch` no longer sets `Mode`; caller must** → One extra line at the resolver callsite (`workItem.Mode = matchedEntry.Mode`). Centralizes mode truth in the registry, which is the goal.

## Migration Plan

Single atomic diff, ordered to keep the build green at each step (each step compiles and passes the pinning test):

1. **Add test** — `internal/cli/create_input_test.go` pins current `ResolveCreateInput` behavior. Run it green against unchanged code.
2. **Extend registry** — add `mode` field to the registry entry, add `mode` param to `Register`, add `KindsFor` query. Update all four `Register` callsites to pass mode. (Old `Register` signature is gone; there are no external callers.)
3. **Migrate providers** — each `init()` passes its mode; each `Fetch` drops the `Mode:` line. The resolver callsite stamps `Mode` on the returned `WorkItem` from the registry entry.
4. **Rewrite resolver** — delete `has*` bools and branch ladder; replace with flag-iteration + `KindsFor` + `len(kinds)` check. Run test green.

No rollback strategy needed — it's a local code change, `git revert` suffices. No data, no deploy, no config migration.
