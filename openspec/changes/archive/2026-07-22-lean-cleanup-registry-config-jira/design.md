## Context

A whole-repo over-engineering audit (ponytail-audit) ranked cuts by impact. Three were approved for this change: a duplicated registry lookup, a dead config-path alias, and the leftover `--jira` surface (flag, resolver branch, test fixtures, spec scenarios) for an adapter whose stub provider was already deleted in the prior audit step. The registry and resolver were last touched in the `decouple-source-kind-flag` change; `GetConfigPath` was demoted to an alias in `add-layered-project-config`.

## Goals / Non-Goals

**Goals:**
- Collapse the registry's two lookup paths into one (`Fetch` delegates to `Resolve`).
- Delete `config.GetConfigPath` and inline its sole caller.
- Remove every `--jira` reference from code, tests, and specs so the surface matches reality (no jira adapter exists or is planned).

**Non-Goals:**
- Re-introducing jira support. If jira is wanted later, it comes back as a new provider package + a new change, not by reverting this one.
- Touching the `shellQuote` duplication across `agent` and `runtime` (audit #3, deferred — shared package overhead not yet justified).
- Touching the `supportsThinkingLevel` duplication across translators (audit #4, kept deliberately as an extension seam for future agents).
- Removing the "more than one source flag" guard in `ResolveCreateInput`. It becomes currently-unreachable once `--jira` is gone, but it is a defensive guard at an input boundary for a function explicitly designed to grow new source flags. It stays.
- Changing the registry's `Provider` interface, `Entry` shape, or `init()`-based self-registration model.

## Decisions

### Decision 1: `Fetch` delegates to `Resolve`, not the reverse

`Resolve(system, kind) (Provider, WorkMode, error)` is the richer API (returns mode too) and already has tests. `Fetch(sourceRef) (WorkItem, error)` is the convenience wrapper for the common case. Making `Resolve` call `Fetch` would lose the mode return; making `Fetch` call `Resolve` preserves both and removes the duplicated map lookup + error string from `Fetch`.

**Alternative considered:** inline `Resolve` into `Fetch` and delete `Resolve`. Rejected — `Resolve` is the lower-level API `create_input` could use for single-kind resolution, and it is tested. Killing it would force `main.go` to redo the lookup `Fetch` already does.

### Decision 2: Delete `GetConfigPath` outright, no deprecation

`GetConfigPath` is an internal-package function with one in-package caller (`EnsureConfigFile`). No external API, no cross-package callers. A deprecation period has no audience. Inline the caller to `GetGlobalConfigPath` and delete the alias.

**Alternative considered:** keep `GetConfigPath` as a stable name. Rejected — it adds a second name for the same path with no caller benefit, which is exactly the dead surface this change exists to remove.

### Decision 3: Remove `--jira` flag entirely, let cobra handle the unknown-flag error

The jira provider's `Fetch` always returned "not yet implemented"; the provider package is already gone. The flag never produced a worker. Removing it means a user who passes `--jira` gets cobra's standard "unknown flag: --jira" at parse time, before `ResolveCreateInput` is ever called. No custom error message is needed for a flag that never worked.

**Alternative considered:** keep `--jira` flag and emit a friendly "jira is not yet supported" error. Rejected — that keeps a dead flag and a dead resolver branch alive, which is the surface this change removes. Cobra's unknown-flag error is clear enough.

### Decision 4: Update error messages to drop `--jira`, keep the "only one source" guard

Error strings that listed `--github, --jira, or --prompt` become `--github or --prompt`. The `len(selected) > 1` guard stays as defensive validation for future source flags; its message becomes `must specify only one source: --github or --prompt`. The guard is currently unreachable with only two flags (prompt is only selected when nothing else is), but it is a cheap boundary guard on a function built to grow new flags.

## Risks / Trade-offs

- [A user script passes `--jira`] → They get cobra's "unknown flag" instead of the prior "not yet implemented". Acceptable: the flag never produced a worker, so no working script depends on it. If this surfaces real users, jira support is a new change away.
- [The "multiple source flags" spec scenario becomes unreachable with current flags] → The scenario is reframed as a hypothetical guard description, consistent with the existing hypothetical gitlab scenario in the same spec. The guard stays in code as boundary validation.
- [Spec drift: `source-registry` loses its only single-kind example using a "real" second system] → The single-kind scenario is restated against `prompt:prompt`, which is already registered and exercises the same code path.
