---
name: shiphq
description: |
  Orchestrate parallel AI agents using Git worktrees, tmux, and opencode. 
  
  Use this skill whenever the user wants to:
  - Spawn multiple AI agents working on different tasks in parallel
  - Create isolated development environments from GitHub issues or Jira tickets
  - Set up a workflow where one agent (orchestrator) dispatches work to other agents
  - Manage parallel development workflows with Git worktrees
  - Create agent sessions that persist in tmux for manual intervention
  
  Trigger on mentions of: parallel agents, worktrees, GitHub issues, orchestration, 
  spawning agents, background tasks, tmux sessions, shiphq, or any workflow involving
  multiple agents working on separate branches/issues simultaneously.
---

# shiphq Agent Orchestration Skill

## What is shiphq?

shiphq is a CLI tool that converts GitHub issues, Jira tickets, or custom prompts into isolated AI agent sessions. It enables an **orchestrator pattern** where you (the agent in the main tmux session) dispatch work to parallel agents running in their own isolated environments.

**Default agent:** opencode. Other agents (claude, codex, custom) available if user specifically requests them via `--agent` flag.

**The workflow:**
1. You are the **orchestrator agent** running in the main tmux session (e.g., `project-dev`)
2. You spawn **worker agents** using `shiphq create` - each gets their own isolated workspace
3. Each worker agent runs in its own tmux session, worktree, and git branch
4. Workers run in the background - you do NOT attach to them
5. The human user can attach to worker sessions to intervene, assist, or monitor
6. You (the orchestrator) continue planning and dispatching more work

Each worker session gets:
- **Git worktree** - Isolated branch/workspace via WorkTrunk
- **tmux session** - Persistent terminal environment (named `{project}-{branch}`)
- **Running agent** - AI agent (default: opencode) with the issue/ticket as its task specification

**Key insight:** The issue/ticket becomes the worker agent's prompt. Title + description = task specification. You spawn workers, they do the work, the human can jump in anytime.

## When to Use shiphq

**Perfect for:**
- Working on multiple GitHub issues simultaneously
- Parallel feature development (each feature = separate agent)
- Long-running background tasks you can jump into anytime
- Issue-driven development workflows
- Complex projects requiring multiple parallel workstreams

**Not for:**
- Simple single-file edits
- Tasks that don't need isolated git branches
- Quick one-off questions

## Prerequisites

Before using shiphq, ensure these are installed and configured:

1. **WorkTrunk** (`wt`) - Git worktree management
   - Install: `brew install worktrunk && wt config shell install`
   - Config: `~/.config/worktrunk/config.toml` with bare repo layout

2. **tmux** - Terminal multiplexer
   - Install: `brew install tmux` or `apt install tmux`

3. **AI Agent** - One of:
   - **opencode** (default) - Install from opencode.ai
   - **claude** (Claude Code) - `brew install claude-code`  
   - **codex** (OpenAI Codex) - see OpenAI docs
   - **Custom agents** - can be configured by user
   
   **YOU (the orchestrator) always default to opencode. Only use other agents if the user specifically requests them.**

4. **GitHub CLI** (`gh`) - For fetching GitHub issues
   - Install: `brew install gh && gh auth login`

5. **tmux-sessionx** - Fuzzy finder for tmux sessions
   - Install: https://github.com/omerxx/tmux-sessionx

## The shiphq Workflow

### 1. Setup Bare Repository

```bash
git clone --bare <repo-url> project/.git
cd project
wt switch ^  # Creates worktree for default branch (dev/main), cds into it
```

**Note:** `^` is WorkTrunk shorthand for default branch. You're now IN the worktree.

### 2. Start the Orchestrator

```bash
tmux new -s project-dev
opencode  # This is your orchestrator agent
```

Chat here, plan work, then dispatch to parallel agents.

### 3. Spawn Parallel Agents

From the orchestrator (or any terminal in the bare repo):

**Default (use opencode):**
```bash
# From GitHub issue (requires --type)
shiphq create --github 456 -t issue

# From GitHub PR (requires --type)
shiphq create --github 456 -t pr

# From Jira ticket
shiphq create --jira PROJ-123

# From custom prompt
shiphq create --prompt "Refactor authentication middleware"
```

**Only if user specifically requests a different agent (you default to opencode):**
```bash
# User asks for Claude Code
shiphq create --github 456 -t issue --agent claude

# User asks for Codex CLI
shiphq create --github 456 -t pr --agent codex
```

**Important:** Do NOT suggest or recommend other agents. Do NOT install agents. Only use `--agent` flag if the user explicitly says "use claude" or "use codex". Default is always opencode.

**Custom prompts for any source:**
You can pass a custom prompt to override the default agent instructions:
```bash
# Override PR review prompt
shiphq create --github 456 -t pr --prompt "Focus on security issues in this PR"

# Override issue implementation
shiphq create --github 456 -t issue --prompt "Write tests for this issue first"

# Jira with custom prompt
shiphq create --jira PROJ-123 --prompt "Focus on database migration"

# Combine custom agent + custom prompt (only if user requests specific agent)
shiphq create --github 456 -t issue --agent claude --prompt "Focus on type safety"
```

**Custom prompts for any source:**
You can pass a custom prompt to override the default agent instructions:
```bash
# Override PR review prompt
shiphq create --github 456 -t pr --prompt "Focus on security issues in this PR"

# Override issue implementation
shiphq create --github 456 -t issue --prompt "Write tests for this issue first"

# Jira with custom prompt
shiphq create --jira PROJ-123 --prompt "Focus on database migration"

# Combine custom agent + custom prompt
shiphq create --github 456 -t issue --agent claude --prompt "Focus on type safety"
```

**What happens:**
1. Fetches issue/PR/ticket details (title, description)
2. Creates branch: `github-issue-456` or `github-pr-456` (uses issue/PR number only)
3. Creates worktree via `wt switch --create`
4. Starts tmux session: `blog-github-issue-456` (format: {project}-{source}-{type}-{id})
5. Spawns opencode (default) with prompt: "Implement GitHub issue #456..." or custom prompt
6. If user requested different agent via `--agent`, spawns that agent instead

**CRITICAL - AGENT MUST NOT ATTACH:** After running `shiphq create`, you (the agent) must NOT attempt to attach to the new session. Do NOT run `shiphq attach` or any tmux attach command. The new session runs in the background in tmux for the human user to interact with. Your job is to spawn it and move on. Simply acknowledge to the user that the session was created.

**Advise the user how to attach:** After creating the session, inform the user they can attach to it using either:
- **tmux-sessionx:** Press tmux prefix + f (e.g., `Ctrl+b f` or `Ctrl+a f`) to fuzzy-find and select the session
- **Direct attach:** `shiphq attach <session-id>` (the session ID was printed when created)

**Example response after creating a session:**
> Created session: `blog-github-issue-456`. The agent is now running in the background. You can attach to it using tmux-sessionx (prefix + f) or run `shiphq attach blog-github-issue-456`.

### 4. Switch Between Sessions (User Action)

The **user** (not the orchestrator agent) switches between sessions using tmux-sessionx or by attaching directly:

```bash
# User presses tmux prefix + f to fuzzy-find sessions
# Or user attaches directly:
shiphq attach blog-github-issue-456
```

**Note:** Session switching is for the user. Agents stay in their own sessions.

### 5. Cleanup When Done

```bash
shiphq cleanup --id blog-github-issue-456
```

This removes:
- Worktree directory
- Git branch
- tmux session

## shiphq Commands

| Command | Usage | Description |
|---------|-------|-------------|
| `create` | `shiphq create --github 456 -t issue` | Spawn opencode agent from GitHub issue (requires -t) |
| `create` | `shiphq create --github 456 -t issue --agent claude` | Spawn with specific agent (only if user requests) |
| `create` | `shiphq create --github 456 -t pr` | Spawn agent from GitHub PR (requires -t) |
| `create` | `shiphq create --jira PROJ-123` | Spawn agent from Jira ticket |
| `create` | `shiphq create --prompt "text"` | Spawn agent from custom prompt |
| `create` | `shiphq create --github 456 -t pr --prompt "custom"` | Override default prompt |
| `list` | `shiphq list` | Show all active sessions for current project |
| `attach` | `shiphq attach <id>` | Attach to tmux session |
| `cleanup` | `shiphq cleanup --id <id>` | Remove worktree + tmux session |

**Note on agents:** Always default to opencode. Only use `--agent` flag if user specifically requests claude, codex, or another agent. Available agents: opencode (default), claude (Claude Code), codex (OpenAI Codex), or custom agents configured in `~/.config/shiphq/config.toml`.

## Naming Convention

Sessions are named: `{project}-{source}-{type}-{id}`

Examples:
- `blog-github-issue-456` (from GitHub issue #456)
- `blog-github-pr-456` (from GitHub PR #456)
- `blog-jira-ticket-PROJ-123` (from Jira ticket PROJ-123)
- `blog-prompt-refactor-auth-middleware` (from custom prompt, truncated)

**Project** = current directory name (auto-detected)  
**Source** = `github`, `jira`, or `prompt`  
**Type** = `issue`, `pr`, `ticket`, or `prompt`  
**ID** = issue number, ticket ID, or truncated prompt text

This naming enables filtering by project, source, or type in tmux-sessionx.

## Key Benefits

### Why tmux?
Each agent runs in tmux (not headless), so you can:
- Jump in anytime to override or assist the agent
- Create new windows for editing (`vim`), servers (`npm run dev`), debugging
- Have multiple panes: agent output, logs, tests side-by-side
- Persist sessions across terminal disconnects

### Why WorkTrunk?
- Automatic worktree path management
- `wt step copy-ignored` shares `node_modules/`, `.env` between worktrees (no cold starts)
- One command to create branch + worktree + switch to it

## Common Patterns

### Pattern 1: Issue-Driven Development

```bash
# In orchestrator session
gh issue list  # Show open issues
shiphq create --github 456 -t issue  # Work on issue #456
shiphq create --github 789 -t issue  # Work on issue #789 simultaneously

# Both agents now running in parallel
# User switches between them with tmux-sessionx (prefix + f)
```

### Pattern 2: Custom Task Queue

```bash
# Plan features with orchestrator, then spawn
shiphq create --prompt "Add user dashboard with React"
shiphq create --prompt "Implement JWT authentication"
shiphq create --prompt "Write tests for API endpoints"

# Three parallel agents working on different features
```

### Pattern 3: Background Research

```bash
# Spawn long-running research task
shiphq create --prompt "Analyze codebase and propose architecture improvements"

# Continue working on other things
# Jump back in later to see results
shiphq attach blog-prompt-analyze-codebase-and-propose
```

### Pattern 4: Agent Handoffs

```bash
# Initial exploration agent
shiphq create --github 456 -t issue

# Agent discovers it needs API changes, spawns second agent
# From within first agent's tmux session:
shiphq create --prompt "Create API endpoint for user authentication"

# Now two related agents working in parallel
```

### Pattern 5: PR Reviews

```bash
# Spawn PR review agent (default: opencode)
shiphq create --github 456 -t pr

# Agent will review the PR and provide feedback
# You can attach later to see the review results
```

### Pattern 6: Background Research

```bash
# Spawn long-running research task (default: opencode)
shiphq create --prompt "Analyze codebase and propose architecture improvements"

# Continue working on other things
# Jump back in later to see results
shiphq attach blog-prompt-analyze-codebase-and-propose
```

### Pattern 7: Custom Prompts for Specific Tasks

```bash
# Issue with custom focus
shiphq create --github 123 -t issue --prompt "Focus only on error handling"

# PR with security focus  
shiphq create --github 456 -t pr --prompt "Security review: check for vulnerabilities"

# Jira with specific implementation notes
shiphq create --jira PROJ-789 --prompt "Implement with Redis caching"

# If user requests specific agent + custom prompt
shiphq create --github 456 -t issue --agent claude --prompt "Focus on type safety"
```

## Best Practices

### 1. Always Use Project Prefix
Name your orchestrator session with project prefix:
```bash
tmux new -s blog-dev  # Not just "dev"
```

This enables filtering all blog-related sessions in tmux-sessionx.

### 2. Cleanup Completed Work
```bash
shiphq list  # See what's running (shows only current project sessions)
shiphq cleanup --id blog-github-issue-456  # Remove finished work
```

Don't leave orphaned worktrees and tmux sessions.

### 3. Use tmux Windows Effectively
```bash
# Inside an agent session (Ctrl+b c to create window)
Ctrl+b c  # New window for manual editing
vim src/auth.js

Ctrl+b c  # New window for dev server
npm run dev

Ctrl+b c  # New window for tests
npm test -- --watch
```

### 4. Leverage WorkTrunk Hooks
Add to `~/.config/worktrunk/config.toml`:

```toml
[post-start]
# Copy gitignored files to skip cold starts
copy = "wt step copy-ignored"
```

This shares `node_modules/`, `target/`, `.env` between worktrees.

### 5. Monitor Agent Progress
```bash
shiphq list  # Quick status check

# Or attach and check
shiphq attach blog-github-456
# Look at agent output in window 2
```

## Troubleshooting

### "Failed to fetch GitHub issue"
- Run `gh auth login` to authenticate
- Ensure you're in a directory with a git remote

### "wt switch failed"
- Ensure WorkTrunk is configured: `wt config shell install`
- Check WorkTrunk config for bare repo layout

### "tmux create failed"
- Check if tmux is installed: `tmux -V`
- Session name might be too long (max 50 chars)

### Can't find sessions in tmux-sessionx
- Make sure sessions are named `{project}-{branch}` format
- Check `tmux list-sessions` to see all sessions

## Integration Example

Complete workflow from scratch:

```bash
# 1. Setup
cd ~/projects
git clone --bare https://github.com/user/myapp.git myapp/.git
cd myapp
wt switch ^

# 2. Start orchestrator
tmux new -s myapp-dev
opencode
# (Now chatting with orchestrator)

# 3. Dispatch work
shiphq create --github 42
shiphq create --github 56

# 4. Switch to agent work
# Press Ctrl+a f, select myapp-github-42

# 5. Cleanup when done
shiphq cleanup --id myapp-github-42
```

## Summary

shiphq transforms GitHub issues and tasks into isolated, persistent agent workspaces. The orchestrator pattern lets one agent plan while many agents execute in parallel, all manageable through tmux.

**Key Features:**
- Multiple AI agents supported (opencode, claude, codex, custom)
- Issue → Agent prompt (auto-generated or custom)
- `shiphq create` → Worktree + tmux + agent
- `shiphq create --agent <name>` → Choose specific agent
- `tmux-sessionx` → Switch between parallel work
- `shiphq cleanup` → Clean removal when done

**Remember:**
- Issue → Agent prompt
- `shiphq create` → Worktree + tmux + agent
- `--agent` flag to choose AI agent
- `--prompt` flag to customize task instructions
- `tmux-sessionx` → Switch between parallel work
- `shiphq cleanup` → Clean removal when done
