# shiphq

**Local agent orchestration** that enables you and your **orchestrator AI agent** to spawn parallel worker agents from GitHub issues, PRs, Jira tickets, or custom prompts—each running in isolated Git worktrees + tmux sessions.

Chat naturally with your orchestrator about what needs to be done. It uses the built-in [orchestrator skill](./skill/SKILL.md) to understand your intent and automatically dispatches specialized agents in parallel. You stay in control while the orchestrator handles the logistics.

Need to jump in? Use tmux session switchers to fuzzy-find and instantly attach to any running agent. Each session is a persistent workspace you can peek into, override, or collaborate with anytime.

> **Note:** Currently supports **local workflow only** (WorkTrunk + tmux + opencode). Cloud sandbox support (Daytona, etc.) is planned for future releases.

## How It Works

**The core idea:** Chat with your orchestrator agent about what needs to be done, and it spawns parallel worker agents for each task.

```
You (in orchestrator session)
│
├─ "Fix login bug #456"     →  Orchestrator uses shiphq skill
│                               └─ shiphq create --github 456 -t issue
│                                  ├─ Creates: github-issue-456 worktree
│                                  ├─ Starts: tmux session blog-github-issue-456
│                                  └─ Spawns: opencode agent with issue context
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
   opencode  # Load the shiphq skill and chat with your orchestrator
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
| [opencode](https://opencode.ai/) | AI coding agent | [Website](https://opencode.ai/) |
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

# Manage sessions
shiphq list                                    # Show all sessions for current project
shiphq attach blog-github-issue-456             # Attach directly
shiphq cleanup --id blog-github-issue-456       # Remove worktree + tmux
```

## What shiphq Does

1. **Fetches issue/PR/ticket** via GitHub/Jira CLI → extracts title + description
2. **Creates branch** from issue → `github-issue-456` (uses number only)
3. **Creates worktree** via `wt switch --create`
4. **Starts tmux session** → `blog-github-issue-456` (format: {project}-{source}-{type}-{id})
5. **Spawns agent** → `opencode --prompt "Implement GitHub issue #456..."` or custom prompt
6. **Cleans up** worktree + tmux on `cleanup`

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
         │ shiphq create --github 456 -t issue
         ▼
┌──────────────────────────────────┐
│ blog-github-issue-456            │
│ ├─ Worktree: ./github-issue-456  │
│ ├─ Tmux: 2 windows               │
│ └─ Agent: opencode (local)       │
└──────────────────────────────────┘
```

**Future:** Cloud runtime support (Daytona sandboxes, etc.) for remote agent execution.

## Commands

- `create --github <num> -t <type>` - Spawn agent from GitHub issue/PR (type: issue, pr)
- `create --jira <id>` - Spawn agent from Jira ticket
- `create --prompt "text"` - Spawn agent from custom prompt
- `create ... --prompt "custom"` - Override default prompt with custom instructions
- `list` - Show active sessions for current project
- `attach <id>` - Attach to tmux session
- `cleanup --id <id>` - Remove worktree and session

## License

MIT
