# multicloud-operators-foundation Agent Instructions

This repository provides the foundational hub and managed-cluster components for
Red Hat Advanced Cluster Management (ACM). It is a Go/Kubernetes codebase using
controller-runtime, client-go, and Open Cluster Management APIs.

## Repository layout

- `cmd/controller/`: hub-side foundation controller and controller-runtime manager.
- `cmd/agent/`: managed-cluster work-manager agent and its controllers.
- `pkg/controllers/`: hub-side reconcilers for cluster information, cluster sets,
  RBAC, image registries, add-ons, garbage collection, and managed service accounts.
- `pkg/klusterlet/`: agent-side action, view, cluster information, and node collection
  controllers.
- `pkg/webhook/`: admission webhook implementations.
- `hack/`: CRD, code-generation, and protobuf update/verification scripts.
- `deploy/`, `examples/`, and `docs/`: deployment manifests, examples, and component
  documentation.
- `test/`: unit, envtest integration, end-to-end, and performance tests.

For system architecture, data flows, and module boundaries, see
[docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).

## Development commands

Run commands from the repository root:

```bash
make build
go test ./pkg/...
make test-integration
make verify
```

- `make build` builds the Go binaries through the OpenShift build-machinery include.
- `go test ./pkg/...` runs the package unit tests without requiring a cluster.
- `make test-integration` provisions envtest assets, compiles `test/integration`, and
  runs the integration suite.
- `make test-e2e` requires configured Hub and managed clusters and deployment assets.
- `make verify` checks generated CRDs and generated code; use `make update` only when
  intentionally regenerating them.
- `make images` builds the `quay.io/stolostron/multicloud-manager:latest` image.

Changes to generated CRDs or code must be made through the corresponding source and
generation scripts, then verified with `make verify`. Keep controller behavior,
RBAC, health probes, and leader-election semantics covered by tests when changing
reconciliation logic.

## Tool availability

- GitHub operations: GitHub MCP tools are available; the `gh` CLI is not assumed.
- Jira operations: Jira MCP tools are available; the `jira` CLI is not assumed.
- If multiple GitHub organization tokens are configured, use the `GH_TOKEN_<ORG>`
  convention without printing token values.

## Personal configuration

Read personal config at the start of any task that needs an assignee, email, or
project key. Canonical path: `~/.config/user.local.md` (tool-agnostic, global).
If the file does not exist, fall back to agent memory (`user-config`), then
placeholders. Run `make personalize` to generate or update the file when Fleet
Engineering tooling is available; this repository does not currently define that
target.

## Fleet Engineering Skills

Fetch and apply the relevant skill when the task matches its domain. The canonical
catalog is [Fleet Engineering skills](https://github.com/OpenShift-Fleet/agentic-sdlc/blob/main/skills/README.md).

| Skill | When to use |
|---|---|
| [bug-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/bug-specialist/SKILL.md) | Bug triage, reproduction, and fix planning |
| [epic-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/epic-specialist/SKILL.md) | Multi-sprint epics with outcomes |
| [feature-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/feature-specialist/SKILL.md) | Large customer-facing capabilities |
| [initiative-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/initiative-specialist/SKILL.md) | Multi-team strategic programs |
| [jira-create](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/jira-create/SKILL.md) | Interactive Jira issue creation |
| [jira-qe-readiness](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/jira-qe-readiness/SKILL.md) | Check whether a Jira ticket is ready for QE |
| [jira-report](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/jira-report/SKILL.md) | Produce Jira portfolio and quality reports |
| [jira-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/jira-specialist/SKILL.md) | General Jira triage and issue management |
| [jira-type-audit](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/jira-type-audit/SKILL.md) | Audit Jira issue types across a hierarchy |
| [outcome-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/outcome-specialist/SKILL.md) | Strategic outcomes tied to OKRs |
| [release-dod](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/release-dod/SKILL.md) | Build a release Definition of Done checklist |
| [risk-report](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/risk-report/SKILL.md) | Detect risk signals and draft a status report |
| [risk-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/risk-specialist/SKILL.md) | Manage risk registers and mitigations |
| [spike-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/spike-specialist/SKILL.md) | Time-boxed research and proof of concepts |
| [story-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/story-specialist/SKILL.md) | User stories and acceptance criteria |
| [supportex-review](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/supportex-review/SKILL.md) | Review support exception requests |
| [task-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/task-specialist/SKILL.md) | Internal technical tasks |
| [ticket-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/ticket-specialist/SKILL.md) | Triage stakeholder requests |
| [backlog-grooming](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/backlog-grooming/SKILL.md) | Scan Jira work for grooming gaps |
| [breaking-changes](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/breaking-changes/SKILL.md) | Detect API, configuration, and behavior breaks |
| [ci-triage](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/ci-triage/SKILL.md) | Diagnose failing pull-request checks |
| [coderabbit-sync](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/coderabbit-sync/SKILL.md) | Maintain the Fleet reference CodeRabbit configuration |
| [cve-triage](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/cve-triage/SKILL.md) | Gather vulnerability evidence and dispositions |
| [cve-sustaining-handoff](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/cve-sustaining-handoff/SKILL.md) | Resolve or hand off CVE tracking work |
| [diagnosing-bugs](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/diagnosing-bugs/SKILL.md) | Reproduce and minimize unclear failures |
| [finish-work](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/finish-work/SKILL.md) | Commit, push, open a PR, and update Jira |
| [github-org-access](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/github-org-access/SKILL.md) | Modify GitHub organization access configuration |
| [init-context-docs](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/init-context-docs/SKILL.md) | Assess and bootstrap repository context docs |
| [opencode-setup](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/opencode-setup/SKILL.md) | Configure OpenCode and its integrations |
| [org-repo-audit](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/org-repo-audit/SKILL.md) | Audit organization repositories for SDLC readiness |
| [pr-fix](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/pr-fix/SKILL.md) | Fix merge conflicts, CI failures, and review comments |
| [pr-hygiene](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/pr-hygiene/SKILL.md) | Manage stale pull requests across repositories |
| [pr-review](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/pr-review/SKILL.md) | Review GitHub pull requests |
| [pr-review-detailed](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/pr-review-detailed/SKILL.md) | Run layered branch or diff analysis |
| [pr-review-fix](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/pr-review-fix/SKILL.md) | Iteratively review and fix local changes |
| [repo-content-audit](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/repo-content-audit/SKILL.md) | Find unlinked or orphaned repository content |
| [repo-setup](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/repo-setup/SKILL.md) | Onboard or refresh a repository for Fleet SDLC |
| [release-notes](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/release-notes/SKILL.md) | Generate categorized release notes from merged PRs |
| [renovate-prs](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/renovate-prs/SKILL.md) | Manage dependency update pull requests |
| [rhacm-addon-wizard](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/rhacm-addon-wizard/SKILL.md) | Guide RHACM add-on development and scaffolding |
| [scored-code-review](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/scored-code-review/SKILL.md) | Deprecated scored code review workflow |
| [session-summary](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/session-summary/SKILL.md) | Summarize session work against Jira and GitHub |
| [start-work](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/start-work/SKILL.md) | Create a Jira sub-task for active work |
| [test-coverage-gap](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/test-coverage-gap/SKILL.md) | Analyze and prioritize coverage gaps |
| [vulnerability-slack-report](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/vulnerability-slack-report/SKILL.md) | Report overdue vulnerability work to Slack |
| [f2f-daily-summary](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/f2f-daily-summary/SKILL.md) | Capture daily face-to-face meeting notes |
| [f2f-epic-specialist](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/f2f-epic-specialist/SKILL.md) | Create and manage face-to-face meeting epics |
| [presentation-task](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/presentation-task/SKILL.md) | Log delivered presentations as Jira tasks |
| [scrum-status](https://raw.githubusercontent.com/OpenShift-Fleet/agentic-sdlc/main/skills/scrum-status/SKILL.md) | Capture scrum bullets and weekly status |
