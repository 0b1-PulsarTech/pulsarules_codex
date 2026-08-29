---
id: agent-orchestration
name: Agent orchestration
---

Split work across agents without corrupting shared state: a read-only planner scopes the slices
before any fan-out, the main session orchestrates and integrates, subagents do bounded work, scouts
are read-only, parallel writers are isolated in worktrees, and history operations stay in the main
session. Use whenever you fan work out to subagents, run agents in parallel, or delegate
implementation.

Plan first because briefs written one at a time hide overlap: the second is written without the
first in view, and the collision shows up only after two agents have edited the same file.

The rules - plan before fanning out, orchestrate-and-integrate, read-only scouts, worktree isolation
verified via `git reflog`, never delegate a git-history op (see `git-history`), verify delegated
results before they land - are the composed agent-orchestration rule.
