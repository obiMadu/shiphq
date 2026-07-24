## Context

All prompt text in `internal/prompt/builder.go` (144 lines) is hard-coded as Go string literals. Four concerns are tangled together: `describeWorkItem` (source label), `deliveryInstruction` (host-specific PR creation wording), `referenceInstruction` (GitHub issue backlink), and `reviewInstruction` (review-only constraint). None are user-editable. The `BuildDefault`/`BuildOverride` split is redundant (audit #7) — `BuildOverride` just trims the input and calls through to the same helpers.

The builder is called from `main.go:135-137` with a `workitem.WorkItem` and `repository.Target`. The result goes to `runtime.LocalRuntime.Create()` which writes it to `.wtmag/prompt.md` in the worker's worktree.

The repo already uses `//go:embed` for the config template (`config.example.toml` is embedded in `main.go` via `defaultConfigFileContents`). Go's `text/template` is stdlib — no new dependency.

## Goals / Non-Goals

**Goals:**
- Extract all prompt strings into `text/template` files embedded via `//go:embed`.
- Ship 12 embedded templates: 3 generic defaults (`implement.tmpl`, `review.tmpl`, `pr.tmpl`) plus 3 host-specific overrides for each (GitHub, GitLab, Bitbucket) — 9 host-specific files.
- Allow users to override templates from disk: project-level (`./wtmag-prompts/`) and global (`~/.config/wtmag/prompts/`), following the existing config layering pattern.
- Add a `--pr` flag to `create` that opts in to the PR creation instruction on implementation tasks. Without `--pr`, no delivery instruction is injected.
- Collapse `BuildDefault` and `BuildOverride` into a single `Build` function.

**Non-Goals:**
- Per-source-kind template overrides (e.g., `github-issue.tmpl`). The override dimension is host, not source kind. Each kind maps to exactly one mode (`implement` or `review`), so the frame is determined by mode + host.
- Template inheritance or partials. Each template is a standalone file.
- Hot-reloading templates. Templates are read at `Build` time from disk; no file watcher.
- Changing the `workitem.WorkItem` or `repository.Target` types.
- Changing how the prompt is written to the worktree (`runtime.LocalRuntime.Create`).

## Decisions

### Decision 1: Go `text/template` (stdlib), no new dependency

The template engine is `text/template` from the Go standard library. No external template engine (Handlebars, Liquid, etc.) is added. The templates use Go template syntax (`{{.Field}}`, `{{if}}`, `{{end}}`).

**Alternative considered:** a simpler token-replacement scheme (`{{source_label}}` string replacement). Rejected — `text/template` is stdlib, already handles conditionals and whitespace, and the learning curve for the simple subset we use is minimal.

### Decision 2: `//go:embed` for default templates, disk-based overrides

Default templates are embedded in the binary via `//go:embed internal/prompt/templates/*.tmpl`. Override resolution checks disk paths first, falling back to embedded defaults. This mirrors the existing config layering (project `wtmag.toml` → global `~/.config/wtmag/config.toml` → hardcoded defaults).

Template directory naming:
- Project overrides: `./wtmag-prompts/` (alongside `wtmag.toml`)
- Global overrides: `~/.config/wtmag/prompts/` (alongside `config.toml`)

**Alternative considered:** store templates in the config file as TOML strings. Rejected — multi-line strings in TOML are awkward, templates are text files not config values, and users want to edit them in their native format.

### Decision 3: Override dimension is host, not source kind

Templates are resolved by `{type}-{host}.tmpl` then `{type}.tmpl`, where `type` is `implement`, `review`, or `pr`, and `host` is the detected repository host (`github`, `gitlab`, `bitbucket`, `unknown`).

The source kind (`github:issue`, `github:pr`, `prompt:prompt`) is NOT a template dimension. Each kind maps to exactly one mode (`implement` or `review`), so the frame is determined by mode + host. The source label ("GitHub issue #456") is resolved in Go and passed as template data.

**Alternative considered:** per-source-kind templates (`github-issue.tmpl`, `github-pr.tmpl`). Rejected — adds a dimension the user explicitly did not want. Host is the customization dimension.

### Decision 4: Resolution order — project host-specific → global host-specific → project generic → global generic → embedded

```
1. ./wtmag-prompts/{type}-{host}.tmpl       (project, host-specific)
2. ~/.config/wtmag/prompts/{type}-{host}.tmpl  (global, host-specific)
3. ./wtmag-prompts/{type}.tmpl              (project, generic)
4. ~/.config/wtmag/prompts/{type}.tmpl         (global, generic)
5. embedded {type}.tmpl                       (last resort)
```

Project overrides global, host-specific overrides generic. This matches the existing config precedence (project `wtmag.toml` overrides global `config.toml`).

**Alternative considered:** global before project. Rejected — the existing config system already establishes project-overrides-global, and templates should follow the same instinct.

### Decision 5: `--pr` flag — opt-in, errors on review

The PR creation instruction is injected only when `--pr` is passed on an implementation task. Without `--pr`, implementation tasks get the work item context with no delivery instruction. This is a **breaking change** from the current behavior where every implementation task gets "commit, push, open a PR" by default.

Passing `--pr` on a review task (`github:pr`) errors: `--pr is only valid with implementation tasks; review tasks already inspect a PR`.

**Alternative considered:** always inject PR instruction (current behavior) with `--no-pr` to opt out. Rejected — the user explicitly wants `--pr` to be opt-in, not opt-out.

### Decision 6: Reference backlink resolved in Go, passed as template data

The reference backlink ("Refs #456" for GitHub issues on GitHub host) is computed in Go from the work item source + repository host, and passed to the PR template as a data field (`{{.Reference}}`). Templates do not contain source-specific logic.

**Alternative considered:** compute the reference inside the template. Rejected — the logic is source+host specific (only GitHub issues on GitHub host get a backlink), and templates should be plain text without branching on source system/kind.

### Decision 7: Collapse BuildDefault/BuildOverride into single Build

One function: `Build(workItem, repoTarget, opts) string` where `opts` carries the override text and the `--pr` flag. `BuildDefault` becomes `opts` with empty override and `--pr` false. `BuildOverride` becomes `opts` with the override text set. This resolves audit finding #7.

### Decision 8: Ship 12 embedded templates

3 generic + 9 host-specific = 12 files in `internal/prompt/templates/`:

| Generic | GitHub | GitLab | Bitbucket |
|---------|--------|--------|-----------|
| `implement.tmpl` | `implement-github.tmpl` | `implement-gitlab.tmpl` | `implement-bitbucket.tmpl` |
| `review.tmpl` | `review-github.tmpl` | `review-gitlab.tmpl` | `review-bitbucket.tmpl` |
| `pr.tmpl` | `pr-github.tmpl` | `pr-gitlab.tmpl` | `pr-bitbucket.tmpl` |

The host-specific PR templates have concrete differences (CLI tool names: `gh`, `glab`, etc.). The host-specific implement and review templates are structurally the same as their generic counterparts by default but exist for per-host customization — a user who wants different implement framing for GitLab repos can edit `implement-gitlab.tmpl` without affecting GitHub.

The generic `pr.tmpl` instructs the agent to detect the git remote host and use the matching CLI tool, so it works for any host including unknown ones. Host-specific PR templates name the concrete tool directly.

## Risks / Trade-offs

- [Breaking change: implementation tasks no longer get PR instruction by default] → Users who rely on the auto-injected "commit, push, open a PR" instruction must add `--pr`. Documented in the change and the README. The PR instruction is still available, just opt-in.
- [Host-specific implement/review templates are identical to generic by default] → This is intentional — they exist as customization seams. The disk space cost is negligible (12 small embedded files), and the benefit is that users can customize per host without touching the generic defaults.
- [Template syntax errors crash at Build time] → Template parsing happens at `Build` call time. A malformed user override template will return an error that surfaces to the CLI. Embedded templates are tested at build time.
- [Unknown host falls back to generic template] → The generic `pr.tmpl` handles this by instructing the agent to detect the host. The generic `implement.tmpl` and `review.tmpl` are host-agnostic, so no fallback issue.
