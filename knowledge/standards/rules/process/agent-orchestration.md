---
id: agent-orchestration
name: Agent orchestration
description: Delegate to agents safely - the main session orchestrates and integrates, implementation runs the staged loop (implement, review, fix, test) inside the cheap tier before it returns, scouts and reviewers are read-only, parallel writers get isolated worktrees, git-history operations are never delegated, and delegated results are verified before they land.
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
7. Run the STAGED LOOP in the cheap model tier, with a SEPARATE agent per stage - implement,
   review, fix, test. A fresh agent brings an empty context, so it neither inherits the previous
   agent's blind spots nor re-reads its rationalisations; the stages are adversarial by design, the
   reviewer's job being to find what the implementer missed. Never let one agent review its own
   work. Only a converged result returns to the orchestrator.
8. The orchestrator's audit is a SECOND review, never a rubber stamp on the agent's own report: one
   reviewer misses things, and an agent grading itself is the reviewer most likely to.
9. Brief the agent with the INTENT behind each task, not just the instruction, and say which side is
   authoritative when a test and the code disagree - "fix the failing test" invites baking the bug
   into the expectation.
10. Tell the agent which commit to work from, and verify the base yourself: a worktree cut from the
    wrong base silently produces work against code that no longer exists.
{{end}}

{{define "recipe"}}
The staged loop for one bounded slice of work:

1. ORCHESTRATOR: bound the slice, name the files the agent owns (disjoint from every other
   concurrent writer), and state the return contract - commits, per-task status, the test that
   proves each item, and the verbatim output of the verification commands.
2. IMPLEMENTER (fresh agent): implement the slice.
3. REVIEWER (fresh agent, read-only, told nothing of the implementer's reasoning): review the diff
   against the loaded skills' validation checklists and return findings as data. Its brief says to
   find what is wrong, not to confirm the work.
4. FIXER (fresh agent): apply the review's findings. TESTER (fresh agent): re-run the gates and
   prove each fix with a test that fails without it. Repeat 3-4 with NEW agents each round until a
   review round returns nothing; only then return.
5. ORCHESTRATOR: audit independently - re-read the diff, re-run build, race tests, linters and the
   governance gate on the integrated tree, and confirm each claimed fix has a test that fails
   without it. Treat "tests pass" in a report as a claim to check, not a result.
6. ORCHESTRATOR: integrate. Every git-history operation (cherry-pick, rebase, fold, reset) happens
   here, never in the agent.

Fixture-green is not correct: an agent can pass build, vet, race tests, lint and its own fixtures and
still be wrong against a real repository. End every brief with a proof run against real data,
reported verbatim.
{{end}}

{{define "forbidden"}}
- Delegating any git-history rewrite to an agent.
- Running two write-capable agents against the same working tree concurrently.
- Giving a scout or reviewer write tools.
- Merging a subagent's output without re-reading the diff and running the build and tests.
- Letting one agent review, fix, or test its own implementation - each stage gets a fresh agent.
- Returning a first draft to the orchestrator before a review round came back empty.
- Accepting an agent's "tests pass" as verification instead of re-running them on the integrated
  tree.
- Delegating a slice without naming the files it owns, or letting two writers own the same file.
{{end}}

{{define "validation"}}
- [ ] The main session kept the decisions and integration; only bounded work was delegated.
- [ ] Every scout/reviewer was read-only.
- [ ] Parallel writers had isolated worktrees; the tree was verified via `git status`/`git reflog`.
- [ ] No git-history operation was delegated.
- [ ] Delegated results were verified (diff re-read, build and tests run) before landing.
- [ ] The staged loop ran with a separate fresh agent per stage; nobody reviewed their own work.
- [ ] The orchestrator audited independently rather than trusting the agent's own report.
- [ ] Every agent had an explicit file-ownership list and a stated base commit, both verified.
{{end}}
