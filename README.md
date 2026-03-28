# shiphq

shiphq turns repo work into a local delivery workflow: take task descriptions (GitHub issues, Jira tickets, custom prompts), create an isolated worktree, start a tmux session, hand the work to an agent, and drive it all the way to a PR. It can also run PR review workflows in the same local environment.

This is the point of shiphq: it is not just a headless agent launcher. Each worker runs in a normal tmux session on your machine, so you can attach at any time, open more windows, run `docker compose`, tail logs, edit files manually, and benefit from WorkTrunk hooks that make local worktrees fast.

## What shiphq is optimizing for

- Task descriptions in, PRs out
- Local execution, not opaque remote sandboxes
- Persistent tmux sessions you can jump into whenever you want
- WorkTrunk-managed worktrees that are easy to create and easy to clean up
- An orchestrator agent that dispatches work without taking control away from the human
- PR review in the same workflow when you need it

## Why this is different

Other agent workflows stop at "spawn an agent and wait." shiphq keeps the whole workflow inside the tooling you already use:

- **tmux sessions** stay alive in the background and are easy to fuzzy-find
- **worktrees** give each agent an isolated branch and filesystem
- **manual intervention** is normal; you can drop in, run servers, inspect logs, or pair with the agent
- **WorkTrunk hooks** can copy ignored files and caches so workers avoid cold starts

The worker session is a real development workspace, not just a background job.

## Supported workflows

- GitHub issues create worker sessions for implementation and PR delivery
- GitHub PRs create worker sessions for review
- Custom prompts create worker sessions from free-form task descriptions
- Jira tickets use the same CLI shape, but the Jira provider is not implemented
- shiphq runs locally with WorkTrunk and tmux

## Default worker behavior

For task descriptions from GitHub issues, shiphq gives the worker an end-to-end delivery prompt by default:

1. implement the task
2. commit the changes
3. push the branch
4. open and submit a GitHub PR with `gh`
5. report the PR URL back

That same end-to-end behavior is the intended default for task descriptions from Jira tickets once the Jira adapter is implemented.

GitHub PR inputs are different: those workers default to a review prompt rather than an implementation prompt.

If you pass `--prompt` alongside `--github` or `--jira`, shiphq uses your instructions and includes the fetched source context.

## Core workflow

1. **Start from the default worktree**

   ```bash
   git clone --bare <repo-url> project/.git
   cd project
   wt switch ^
   ```

2. **Run your orchestrator in tmux**

   ```bash
   tmux new -s project-dev
   opencode
   ```

3. **Spawn workers from task descriptions or PR review targets**

   ```bash
   shiphq create --github 456 -t issue
   shiphq create --github 234 -t pr
   shiphq create --prompt "Refactor authentication middleware"
   ```

4. **Jump into a worker whenever you want**

   ```bash
   shiphq attach project-github-issue-456
   ```

   Or use a tmux session switcher such as `tmux-sessionx`.

5. **Clean up when the work is done**

   ```bash
   shiphq cleanup --id project-github-issue-456
   ```

## What `create` does

For a task description like `shiphq create --github 456 -t issue`, shiphq:

1. fetches the issue title and body with `gh`
2. creates an isolated branch/worktree
3. starts a tmux session named `{project}-{source}-{type}-{id}`
4. starts the selected agent inside that worktree
5. gives the worker a prompt that aims at implementation plus PR creation

The worker runs in the background. The human can attach. The orchestrator should not.

## Extensible source architecture

shiphq treats task descriptions as a common internal work-item model rather than baking GitHub-specific behavior into the whole app.

- source providers fetch and normalize work from each system
- prompt building stays separate, so GitHub issues still get GitHub-specific prompts and Jira tickets still get Jira-specific prompts
- repository target detection is separate from task sources, which keeps future Jira-to-GitLab or Jira-to-Bitbucket workflows clean
- session metadata is stored explicitly instead of being reconstructed from session names

Built-in providers:

- GitHub issue
- GitHub PR
- custom prompt
- Jira ticket stub

## Dependencies

| Tool | Purpose |
|------|---------|
| [WorkTrunk](https://worktrunk.dev/) | Create and manage isolated worktrees |
| [tmux](https://github.com/tmux/tmux) | Persistent local worker sessions |
| [GitHub CLI](https://cli.github.com/) | Fetch GitHub task descriptions and open PRs |
| [tmux-sessionx](https://github.com/omerxx/tmux-sessionx) | Optional fuzzy tmux session switcher |
| AI agent CLI | Run the actual worker inside each session |

Built-in agent support:

- `opencode` (default)
- `claude`
- `codex`

You can also add custom agents through config.

## Installation

```bash
go install github.com/obiMadu/shiphq@latest
```

## Usage

```bash
# GitHub issue -> implement -> push -> open PR
shiphq create --github 456 -t issue

# GitHub PR -> review
shiphq create --github 456 -t pr

# Custom instructions plus issue context
shiphq create --github 456 -t issue --prompt "Start by writing tests"

# Custom prompt task
shiphq create --prompt "Refactor authentication middleware"

# Jira shape (adapter not implemented yet)
shiphq create --jira PROJ-123

# Use a different built-in agent if explicitly requested
shiphq create --github 456 -t issue --agent claude

# Session management
shiphq list
shiphq attach project-github-issue-456
shiphq cleanup --id project-github-issue-456
```

## Custom agents

Create `~/.config/shiphq/config.toml`:

```toml
[agents.aider]
command = "aider"
prompt_flag = "--message"

[agents.custom]
command = "my-agent"
prompt_flag = ""
```

Then use it with `--agent aider` or `--agent custom`.

`prompt_flag` tells shiphq how to pass the generated task prompt:

- `--prompt` -> `agent --prompt "..."`
- `--message` -> `agent --message "..."`
- `""` -> `agent "..."`

## WorkTrunk hooks

WorkTrunk is a big part of the value here. Hooks can make worker startup much faster by copying ignored files and caches into new worktrees.

Example `~/.config/worktrunk/config.toml`:

```toml
[post-start]
copy = "wt step copy-ignored"
```

That lets workers reuse things like `node_modules`, `.env`, or build caches instead of rebuilding everything from scratch.

## Limitations

- Jira support is still a stub, so GitHub issues are the main end-to-end task-description path
- Remote runtimes are not implemented; shiphq is local-first
- shiphq is strongest when you stay in the local tmux/worktree workflow; that is the product, not an implementation detail

## License

MIT
