---
id: agent-orchestration
name: Agent orchestration
description: Delegate to agents safely - the main session orchestrates and integrates, scouts and reviewers are read-only, parallel writers get isolated worktrees verified via git reflog, git-history operations are never delegated, and delegated results are verified before they land.
tags:
    - process
    - agents
---

# Agent orchestration

> Split work across agents without corrupting shared state: the main session owns the decisions and
> the integration, subagents do bounded work, scouts are read-only, parallel writers are isolated in
> worktrees, git-history stays in the main session, and every delegated result is verified.

Applies to: any task that fans work out to subagents or runs agents in parallel.

{{define "when"}}
- Delegating implementation or review to subagents.
- Running agents in parallel against a working tree.
- Deciding what a subagent is allowed to do.
- About to hand any git operation to an agent.
{{end}}

{{define "must"}}
1. Orchestrate from the main session and delegate the BOUNDED work; the orchestrator keeps the
   decisions and the final integration.
2. Scope scouts and reviewers READ-ONLY - no write tools; their findings return as data, not edits.
3. Isolate parallel writers: give each concurrent writer its own git worktree (or run them
   sequentially), and verify the tree with `git status` and `git reflog` after any parallel phase.
4. NEVER delegate a git-history operation (rebase, squash, fold, reorder, reset, cherry-pick,
   force-push) - these stay in the main session under direct control (see `[[git-history]]`).
5. Give each subagent a return contract (the structured data you need back) and integrate the
   results yourself.
6. Verify delegated work before acting on it: re-read the diff, run the build and tests, confirm
   findings against the code.
{{end}}

{{define "forbidden"}}
- Delegating any git-history rewrite to an agent.
- Running two write-capable agents against the same working tree concurrently.
- Giving a scout or reviewer write tools.
- Merging a subagent's output without re-reading the diff and running the build and tests.
{{end}}

{{define "validation"}}
- [ ] The main session kept the decisions and integration; only bounded work was delegated.
- [ ] Every scout/reviewer was read-only.
- [ ] Parallel writers had isolated worktrees; the tree was verified via `git status`/`git reflog`.
- [ ] No git-history operation was delegated.
- [ ] Delegated results were verified (diff re-read, build and tests run) before landing.
{{end}}
