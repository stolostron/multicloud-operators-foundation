# multicloud-operators-foundation

@AGENTS.md

## Personal configuration

Read personal config at the start of any task that needs an assignee, email, or project key.
Canonical path: `~/.config/user.local.md` (tool-agnostic, global).
If the file does not exist, fall back to agent memory (`user-config`), then placeholders.
Run `make personalize` to generate or update the file when Fleet Engineering tooling is available.

## Tool availability

- GitHub operations: GitHub MCP tools are available; the `gh` CLI is not assumed.
- Jira operations: Jira MCP tools are available; the `jira` CLI is not assumed.
