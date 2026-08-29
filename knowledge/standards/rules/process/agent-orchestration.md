---
id: agent-orchestration
name: Agent orchestration
description: Delegate to agents safely - a read-only planner scopes the slices before any fan-out, the main session orchestrates and integrates, implementation runs the staged loop (implement, review, fix, test) inside the cheap tier before it returns, scouts and reviewers are read-only, parallel writers get isolated worktrees, git-history operations are never delegated, and delegated results are verified before they land.
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
1. Orchestrate from the main session and delegate the bounded work; the orchestrator keeps the
   decisions and the final integration.
2. Scope scouts and reviewers read-only - no write tools; their findings return as data, not edits.
3. Isolate parallel writers: give each concurrent writer its own git worktree (or run them
   sequentially), and verify the tree with `git status` and `git reflog` after any parallel phase.
4. NEVER delegate a git-history operation (rebase, squash, fold, reorder, reset, cherry-pick,
   force-push) - these stay in the main session under direct control (see `[[git-history]]`).
5. Give each subagent a return contract (the structured data you need back) and integrate the
   results yourself.
6. Verify delegated work before acting on it: re-read the diff, run the build and tests, confirm
   findings against the code.
7. Run the staged loop in the cheap model tier, with a separate agent per stage - implement,
   review, fix, test. A fresh agent brings an empty context, so it neither inherits the previous
   agent's blind spots nor re-reads its rationalisations; the stages are adversarial by design, the
   reviewer's job being to find what the implementer missed. Never let one agent review its own
   work. Only a converged result returns to the orchestrator.
8. The orchestrator's audit is a second review, never a rubber stamp on the agent's own report: one
   reviewer misses things, and an agent grading itself is the reviewer most likely to.
9. Brief the agent with the intent behind each task, not just the instruction, and say which side is
   authoritative when a test and the code disagree - "fix the failing test" invites baking the bug
   into the expectation.
10. Tell the agent which commit to work from, and verify the base yourself: a worktree cut from the
    wrong base silently produces work against code that no longer exists.
11. Plan before you fan out. Any time more than one agent will run, a single read-only planner runs
    first and returns the slice plan: the slices, the files each one owns, the base commit, the
    return contract per slice, and any ordering the slices impose on each other. The orchestrator
    launches from that plan rather than inventing slices while writing briefs. Writing briefs one at
    a time hides overlap, because the second brief is written without the first in view, and the
    collision surfaces only when two agents have already edited the same file. One read-only pass
    over the whole surface costs a fraction of an aborted parallel phase.
12. The plan is the orchestrator's to accept, amend or reject. A planner proposes slices; it does
    not authorise them, and a slice that the orchestrator cannot state the purpose of does not get
    launched.
{{end}}

{{define "recipe"}}
planner (fresh agent, read-only), whenever more than one agent will run:

0. Planner: survey the work and return the slice plan - the slices, the files each owns, the base
   commit, the per-slice return contract, and the ordering between slices. It writes nothing and
   decides nothing; it reports. The orchestrator reads the plan, checks that the ownership lists are
   genuinely disjoint, and amends what it disagrees with before anything else starts.

Then the staged loop, for each bounded slice:

1. Orchestrator: bound the slice from the plan, name the files the agent owns (disjoint from every
   other concurrent writer), and state the return contract - commits, per-task status, the test that
   proves each item, and the verbatim output of the verification commands.
2. Implementer (fresh agent): implement the slice.
3. Reviewer (fresh agent, read-only, told nothing of the implementer's reasoning): review the diff
   against the loaded skills' validation checklists and return findings as data. Its brief says to
   find what is wrong, not to confirm the work.
4. Fixer (fresh agent): apply the review's findings. Tester (fresh agent): re-run the gates and
   prove each fix with a test that fails without it. Repeat 3-4 with new agents each round until a
   review round returns nothing; only then return.
5. Orchestrator: audit independently - re-read the diff, re-run build, race tests, linters and the
   governance gate on the integrated tree, and confirm each claimed fix has a test that fails
   without it. Treat "tests pass" in a report as a claim to check, not a result.
6. Orchestrator: integrate. Every git-history operation (cherry-pick, rebase, fold, reset) happens
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
- Fanning out to more than one agent with no planner pass first, so the slices are invented one brief
  at a time and overlap is found only after work has been done.
- Giving the planner write tools, or treating its plan as an authorisation rather than a proposal.
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
- [ ] A read-only planner ran before any multi-agent fan-out, and its ownership lists were checked
  for overlap by the orchestrator rather than trusted.
{{end}}
