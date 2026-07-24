---
name: wtmag
description: |
  Orchestrate local AI workers with Git worktrees, tmux, and wtmag.

  Use this skill whenever the user wants to:
  - Spawn parallel AI agents from task descriptions (GitHub issues, OpenSpec changes, custom prompts) or PRs
  - Create isolated local workspaces for agent execution
  - Run an orchestrator/worker workflow inside tmux
  - Manage background agent sessions they can manually jump into later

  Trigger on mentions of: wtmag, task descriptions, worktrees, tmux sessions,
  parallel workers, orchestrators, background coding agents, or local task-description-to-PR delivery.
---

# wtmag Skill

## What wtmag is for

wtmag is a local workflow for turning task descriptions into PRs and for reviewing PRs. It creates an isolated worktree and a tmux worker (dedicated session or parent-session window), runs a worker agent inside that real local workspace, and lets the human attach whenever they want.

This is why tmux matters: the worker is running in a normal tmux environment the human can inspect, override, and extend when needed.

## Orchestrator behavior

You are the orchestrator running in the main tmux session.

- Spawn workers with `wtmag create`
- Do not attach to the worker after spawning it
- Tell the human how to attach if they want to inspect or intervene
- Let wtmag use its configured default agent when the user does not specify one
- Only use `--agent <name>` if the user explicitly asks for a different agent or wants to override config
- Only use `--model <provider/model[:thinking]>` if the user explicitly asks for a specific model or thinking level
- Before using `--model`, inspect the selected agent's local CLI help and treat that local output as the source of truth
- Never recommend or install a different agent on your own
- Let wtmag use its default placement unless the user explicitly asks for `-s`, `-w`, or `--launch`
- Generated config defaults both implementation work and review work to a worker window in the current tmux session
- With the generated defaults, `wtmag create` should usually be run inside tmux unless the user explicitly overrides placement to `session`

## Default agent choice

wtmag resolves `agents.default.name` from `wtmag.toml` in the current project first, then `~/.config/wtmag/config.toml`.

On first run, wtmag creates the global home config if it does not exist yet. The generated config includes the bundled agent definitions (`pi`, `opencode`, `claude`, `codex`) and sets `pi` as the initial default.

wtmag resolves `[launch] implementation` and `[launch] review` with the same precedence. The generated config sets both defaults to `window`.

That default can point to one of those bundled agents or to a custom agent the user defines under `[agents.<name>]`, but you should only select a different agent when the user explicitly requests it.

Built-in prompt delivery:

- `pi`, `claude`, and `codex` use positional prompts
- `opencode` uses `--prompt`

Model override format:

- use `--model provider/model[:thinking]` when the user explicitly asks for a model override
- wtmag currently translates `--model` for `pi` and `opencode`
- recognized thinking levels are `off`, `none`, `minimal`, `low`, `medium`, `high`, `xhigh`, and `max`
- `pi` receives `--model provider/model[:thinking]`
- `opencode` receives `--model provider/model` and maps `:thinking` to `--variant`
- if the selected agent does not support wtmag model overrides, say so instead of guessing provider-specific flags
- before using `--model`, inspect the locally installed agent CLI help (`pi --help`, `opencode --help`, `opencode run --help`, `opencode models --help`) and use that output as the source of truth; add future agents to this list as wtmag model override support expands
- after verification, still call `wtmag create ... --model provider/model[:thinking]`; do not bypass wtmag by passing provider-specific model flags directly

## What the worker is supposed to do

`Task descriptions` is the umbrella product term only. Do not flatten source-native terminology in commands or prompts: GitHub issues are still GitHub issues, and PR review targets are still PRs.

For **task descriptions** such as GitHub issues, wtmag's default prompt is implementation-oriented:

- implement the task
- report back

The PR creation instruction (commit, push, open a PR) is **not injected by default**. Pass `--pr` to inject it:

- implement the task
- commit the changes
- push the branch
- open and submit a GitHub PR with `gh`
- report the PR URL back

For **GitHub PRs**, the default worker behavior is review-only:

- inspect the diff and relevant context
- report findings back
- do not edit files
- do not implement fixes
- do not commit, push, approve, merge, or otherwise modify the PR

If the user passes `--prompt` along with `--github`, wtmag keeps the source context and adds the user's instructions. **Do not use `--prompt` with `--github` unless the user explicitly asks for custom instructions.** The built-in prompts for issues already contain the work item context. Adding a custom prompt usually strips out the source context because agents rarely include it in their override text. Only add `--prompt` when the user specifically requests extra instructions like "Start by writing tests" or "Use this specific approach."

## Commands

### GitHub issue

```bash
wtmag create --github 456 -t issue
```

### GitHub PR

```bash
wtmag create --github 456 -t pr
```

### Custom prompt

```bash
wtmag create --prompt "Refactor authentication middleware"
```

### OpenSpec change

```bash
wtmag create --opsx add-user-auth
```

The change directory (`openspec/changes/<name>/`) is moved from the current working directory into the new worktree. The worker prompt includes the proposal content and instructs the agent to read the full change and use the `openspec-apply-change` skill to implement it.

### Add custom instructions to a GitHub issue

```bash
wtmag create --github 456 -t issue --prompt "Start by writing tests"
```

### Inject PR creation instructions

```bash
wtmag create --github 456 -t issue --pr
```

### Use a different agent only when requested

```bash
wtmag create --github 456 -t issue --agent claude
wtmag create --github 456 -t issue --agent opencode --model openai/gpt-5.2:high
wtmag create --github 456 -t issue --agent pi --model anthropic/claude-sonnet-4.5:high
```

### Worker management

```bash
wtmag list
wtmag list --all
wtmag attach project-github-issue-456
wtmag promote --id project-github-pr-456
wtmag cleanup --id project-github-issue-456
wtmag cleanup --id project-github-issue-456 --force
```

## Important command rules

- `--type` / `-t` is required for GitHub and must be `issue` or `pr`
- `--opsx <change-name>` spawns a worker from an OpenSpec change; the change files are moved into the worktree
- `--prompt` by itself means a custom prompt task
- `--prompt` with `--github` means "keep the source context and add these instructions"
- `--pr` injects PR creation instructions (commit, push, open PR) into the implementation prompt; it is only valid with implementation tasks, not review tasks
- `--model` uses `provider/model[:thinking]` format and is intended for explicit per-run model overrides
- wtmag currently supports `--model` overrides for `pi` and `opencode`
- `-s` / `--launch session` forces a dedicated tmux session
- `-w` / `--launch window` forces a worker window in the current tmux session (requires running inside tmux)
- `list` shows known workers for the current detected project; use `list --all` to see WTmag workers across projects
- `promote --id ...` upgrades a window worker into a dedicated tmux session
- `cleanup --force` uses `wt remove --force` for the worktree

## Prompt templates

wtmag builds worker prompts from template files using Go's `text/template` syntax. Three template types exist:

- `implement` — the implementation frame ("Implement X.\n\n{context}")
- `review` — the review frame ("Review X.\n\n{context}\n\n{review instructions}")
- `pr` — the PR creation instruction ("commit, push, open a PR..."), only injected when `--pr` is passed

Each type has a generic default and host-specific overrides (GitHub, GitLab, Bitbucket). Templates are overridable from disk:

1. `./wtmag-prompts/{type}-{host}.tmpl` — project, host-specific
2. `~/.config/wtmag/prompts/{type}-{host}.tmpl` — global, host-specific
3. `./wtmag-prompts/{type}.tmpl` — project, generic
4. `~/.config/wtmag/prompts/{type}.tmpl` — global, generic
5. Embedded default — shipped in the binary

Project overrides global, host-specific overrides generic. See the README "Prompt Templates" section for template data fields.

## Human vs orchestrator responsibilities

After `wtmag create`:

- the worker keeps running in the background
- the human can attach to that tmux worker (dedicated session or parent-session window)
- you must not attach on the human's behalf

Good follow-up response:

> Created worker `project-github-issue-456`. The worker is running in the background. You can jump in with tmux-sessionx or run `wtmag attach project-github-issue-456`.
