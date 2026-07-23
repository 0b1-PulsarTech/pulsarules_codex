---
id: agent-orchestration
name: Agent orchestration
---

Split work across agents without corrupting shared state: the main session orchestrates and
integrates, subagents do bounded work, scouts are read-only, parallel writers are isolated in
worktrees, and history operations stay in the main session. Use whenever you fan work out to
subagents, run agents in parallel, or delegate implementation.

The rules - orchestrate-and-integrate, read-only scouts, worktree isolation verified via
`git reflog`, never delegate a git-history op (see `git-history`), verify delegated results before
they land - are the composed agent-orchestration rule.

## Mandatory workflow

1. Orchestrate from the main session and delegate only BOUNDED work; the decisions and the final
   integration stay with you.
2. Scope scouts and reviewers READ-ONLY - no write tools. Their findings come back as data, never as
   edits.
3. Isolate parallel writers: one git worktree each, or run them sequentially. Verify the tree with
   `git status` and `git reflog` after any parallel phase.
4. NEVER delegate a git-history operation (rebase, squash, fold, reorder, reset, cherry-pick,
   force-push). Those stay in the main session under direct control (see `[[git-history]]`).
5. Give each subagent a return contract - the structured data you need back - and integrate the
   results yourself.
6. Verify delegated work before acting on it: re-read the diff, run the build and tests, and confirm
   findings against the code.

## Validation checklist

- [ ] The main session kept the decisions and integration; only bounded work was delegated.
- [ ] Every scout/reviewer was read-only.
- [ ] Parallel writers had isolated worktrees; the tree was verified via `git status`/`git reflog`.
- [ ] No git-history operation was delegated.
- [ ] Delegated results were verified (diff re-read, build and tests run) before landing.

## Forbidden actions

- Delegating any git-history rewrite to an agent.
- Running two write-capable agents against the same working tree concurrently.
- Giving a scout or reviewer write tools.
- Merging a subagent's output without re-reading the diff and running the build and tests.

## Expected outputs

- Integrated work the main session verified itself, with git history operations never delegated.
- Read-only scouting results returned as data.
