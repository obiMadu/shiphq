# shiphq

**Local agent orchestration** that enables you and your **orchestrator AI agent** to spawn parallel worker agents from GitHub issues, PRs, Jira tickets, or custom prompts—each running in isolated Git worktrees + tmux sessions.

Chat naturally with your orchestrator about what needs to be done. It uses the built-in [orchestrator skill](./skill/SKILL.md) to understand your intent and automatically dispatches specialized agents in parallel. You stay in control while the orchestrator handles the logistics.

Need to jump in? Use tmux session switchers to fuzzy-find and instantly attach to any running agent. Each session is a persistent workspace you can peek into, override, or collaborate with anytime.

> **Note:** Currently supports **local workflow only** (WorkTrunk + tmux + local agent CLIs such as pi, opencode, claude, or codex). Cloud sandbox support (Daytona, etc.) is planned for future releases.

## How It Works

**The core idea:** Chat with your orchestrator agent about what needs to be done, and it spawns parallel worker agents for each task.

```
You (in orchestrator session)
│
├─ "Fix login bug #456"     →  Orchestrator uses shiphq skill
│                               └─ shiphq create --github 456 -t issue
│                                  ├─ Creates: github-issue-456 worktree
│                                  ├─ Starts: tmux session blog-github-issue-456
│                                  └─ Spawns: default agent with issue context
│
└─ "Review PR #234"         →  Orchestrator spawns another agent
                                  └─ shiphq create --github 234 -t pr
                                     └─ Parallel agent session

Result: Multiple isolated worktrees + tmux sessions + running agents
```

## Workflow

1. **Setup bare repo** with [WorkTrunk](https://worktrunk.dev/)
   ```bash
   git clone --bare <repo> project/.git && cd project
   wt switch ^   # Creates worktree for default branch (usually dev, could be main)
                 # ^ = default branch shorthand. wt cds into it automatically.
   ```

2. **Start orchestrator** (already in the default branch worktree)
   ```bash
   tmux new -s project-dev
   pi --skill ./skill  # Load the shiphq skill and chat with your orchestrator
   ```

3. **Discuss and dispatch** with your orchestrator
   
   Simply chat naturally about what needs to be done:
   ```
   You: "Fix the login bug (#456) and review PR #234"
   
   Orchestrator: Uses shiphq skill to spawn agents automatically
   → shiphq create --github 456 -t issue
   → shiphq create --github 234 -t pr
   ```
   
   The orchestrator understands your intent and runs the right commands.

4. **Switch sessions** - Jump into any agent's workspace 
   
   **Recommended:** Use [tmux-sessionx](https://github.com/omerxx/tmux-sessionx) to fuzzy-find and switch:
   ```bash
   # Press prefix + f, fuzzy find "blog-github-issue-456"
   ```
   
   *Alternative:* You can use any tmux session manager, or attach directly:
   ```bash
   shiphq attach blog-github-issue-456
   ```

5. **Cleanup** when done
   ```bash
   shiphq cleanup --id blog-github-issue-456
   ```

## Dependencies

| Tool | Purpose | Links |
|------|---------|-------|
| [WorkTrunk](https://worktrunk.dev/) | Git worktree management | [Install](https://worktrunk.dev/worktrunk/) · [GitHub](https://github.com/max-sixty/worktrunk) |
| [tmux](https://github.com/tmux/tmux) | Terminal multiplexer | [GitHub](https://github.com/tmux/tmux) |
| AI Agent (pick one): | | |
| ├─ [pi](https://pi.dev/) | Fallback default AI agent | [Website](https://pi.dev/) |
| ├─ [opencode](https://opencode.ai/) | OpenCode AI agent | [Website](https://opencode.ai/) |
| ├─ [claude-code](https://docs.anthropic.com/en/docs/claude-code) | Claude Code by Anthropic | [Docs](https://docs.anthropic.com/en/docs/claude-code) |
| └─ [codex](https://help.openai.com/en/articles/11096431-openai-codex-cli-getting-started) | Codex CLI by OpenAI | [Install](https://help.openai.com/en/articles/11096431-openai-codex-cli-getting-started) |
| [GitHub CLI](https://cli.github.com/) | Fetch GitHub issues | [Install](https://github.com/cli/cli#installation) · [Manual](https://cli.github.com/manual/) |
| [tmux-sessionx](https://github.com/omerxx/tmux-sessionx) | Fuzzy find tmux sessions | [GitHub](https://github.com/omerxx/tmux-sessionx) |
| Jira CLI (optional) | Fetch Jira tickets | [Install](https://github.com/ankitpokhrel/jira-cli) |

## Installation

```bash
go install github.com/obiMadu/shiphq@latest
```

## Usage

```bash
# Create agent from GitHub issue (requires --type flag)
shiphq create --github 456 -t issue
shiphq create --github 456 -t pr     # For PR reviews

# Create agent from Jira ticket
shiphq create --jira PROJ-123  

# Create agent from custom prompt
shiphq create --prompt "Custom task"

# Override default prompt with custom instructions
shiphq create --github 456 -t issue --prompt "Focus on test coverage"

# Use different AI agents (built-in: pi, opencode, claude, codex)
shiphq create --github 456 -t issue --agent opencode
shiphq create --github 456 -t issue --agent claude
shiphq create --github 456 -t issue --agent codex

# Manage sessions
shiphq list                                    # Show all sessions for current project
shiphq attach blog-github-issue-456             # Attach directly
shiphq cleanup --id blog-github-issue-456       # Remove worktree + tmux
shiphq cleanup --id blog-github-issue-456 --force
```

## Supported AI Agents

shiphq creates `~/.config/shiphq/config.toml` on first run if it does not exist. That generated file includes the default agent selection plus the bundled agent definitions, so you can edit how `pi`, `opencode`, `claude`, and `codex` launch without touching code.

When `--agent` is omitted, shiphq uses `agents.default.name` from `~/.config/shiphq/config.toml`. Freshly generated configs default that to `pi`.

| Agent | Command | Prompt delivery | Notes |
|-------|---------|-----------------|-------|
| **pi** (default in generated config) | `pi` | positional | Pi coding agent from [pi.dev](https://pi.dev/) |
| **opencode** | `opencode` | `--prompt` | OpenCode AI agent |
| **claude** | `claude` | positional | Claude Code by Anthropic |
| **codex** | `codex` | positional | Codex CLI by OpenAI |

All agents spawn in interactive mode (TUI) so you can jump in and collaborate. shiphq writes the full task brief to `prompt.md` in the worktree, then sends a small bootstrap instruction using the agent's configured prompt delivery style.

### Adding Custom Agents

You can edit the generated config or create it ahead of time yourself. The shipped template is `config.example.toml`, and shiphq copies it to `~/.config/shiphq/config.toml` on first run when that file is missing:

```toml
# ShipHQ writes this template to ~/.config/shiphq/config.toml on first run if the file does not exist.

# Default agent selection.
[agents.default]
# Use one of the built-in agent names below, or whatever comes after `agents.` in a custom agent definition.
name = "pi"

# Built-in agent definitions.
[agents.pi]
command = "pi"
prompt_flag = ""

[agents.opencode]
command = "opencode"
prompt_flag = "--prompt"

[agents.claude]
command = "claude"
prompt_flag = ""

[agents.codex]
command = "codex"
prompt_flag = ""

# Custom agent examples.
# `prompt_flag` is optional. If you omit it, shiphq passes the prompt positionally.
# [agents.aider]
# command = "aider"
# prompt_flag = "--message"

# [agents.custom-agent-with-args]
# command = "my-agent"
# args = ["run", "--profile", "coding"]
```

Set `agents.default.name` to any agent table name in the config. That can be one of the generated built-ins (`pi`, `opencode`, `claude`, `codex`) or a custom `[agents.<name>]` block you add yourself. `--agent` still overrides the config for a single run.

Then use it: `shiphq create --github 456 -t issue --agent aider`

**Why `prompt_flag` matters:** shiphq writes the full brief to `prompt.md`, then passes a bootstrap prompt that tells the agent to read that file. The `prompt_flag` tells shiphq how to send that bootstrap prompt:
- `--prompt` → `opencode --prompt "Read ./prompt.md and use it as the full task brief."`
- `--message` → `aider --message "Read ./prompt.md and use it as the full task brief."`
- `""` (empty) → `pi "Read ./prompt.md and use it as the full task brief."` (positional)

For custom agents, `prompt_flag` is optional. If you leave it out, shiphq uses positional prompt delivery.

This works for both built-in prompts (from issues/PRs) and custom prompts via `--prompt "custom instructions"`.

**For maintainers:** shipped agent defaults now live in `config.example.toml`, which shiphq copies to `~/.config/shiphq/config.toml` on first run.

## What shiphq Does

1. **Fetches issue/PR/ticket** via GitHub/Jira CLI → extracts title + description
2. **Creates branch** from issue → `github-issue-456` (uses number only)
3. **Creates worktree** via `wt switch --create`
4. **Starts tmux session** → `blog-github-issue-456` (format: {project}-{source}-{type}-{id})
5. **Writes `prompt.md`** in the worktree with the full task brief
6. **Spawns agent** → Your choice of AI agent (pi, opencode, claude, codex, or custom) with a bootstrap prompt
7. **Cleans up** worktree + tmux on `cleanup`

## Why tmux?

Each agent session runs in **tmux** (not headless) so you can:

- **Jump in anytime** to override the agent or do manual work
- **Create new windows** for editing (`vim`), running servers (`npm run dev`), debugging (`docker-compose up`)
- **Multiple panes** - agent in one, logs in another, tests in a third

It's a persistent workspace, not just a background process.

## WorkTrunk Optimizations

Configure [WorkTrunk hooks](https://worktrunk.dev/hook/) in `~/.config/worktrunk/config.toml`:

```toml
[post-start]
# Copy gitignored files (node_modules/, .env, build caches) to skip cold starts
copy = "wt step copy-ignored"
```

This shares dependencies between worktrees so agents don't reinstall from scratch.

## Architecture

**Current implementation: Local only**

```
┌──────────────────┐
│ Orchestrator     │ You chat here, dispatch work
│ (tmux: dev)      │
└────────┬─────────┘
         │ shiphq create --github 456 -t issue --agent claude
         ▼
┌──────────────────────────────────┐
│ blog-github-issue-456            │
│ ├─ Worktree: ./github-issue-456  │
│ ├─ Tmux: agent window            │
│ └─ Agent: claude (or pi,         │
│            opencode, codex,      │
│            or custom)            │
└──────────────────────────────────┘
```

**Future:** Cloud runtime support (Daytona sandboxes, etc.) for remote agent execution.

## Commands

- `create --github <num> -t <type>` - Spawn agent from GitHub issue/PR (type: issue, pr)
- `create --jira <id>` - Spawn agent from Jira ticket
- `create --prompt "text"` - Spawn agent from custom prompt
- `create ... --agent <name>` - Use specific AI agent (otherwise shiphq uses config `agents.default.name` or `pi`)
- `create ... --prompt "custom"` - Override default prompt with custom instructions
- `list` - Show active sessions for current project
- `attach <id>` - Attach to tmux session
- `cleanup --id <id> [--force]` - Remove worktree and session (`--force` uses `wt remove --force`)

## License

MIT
