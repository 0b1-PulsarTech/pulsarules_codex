---
id: architecture-decision-records
name: Architecture decision records
description: Record an architecturally-significant decision as a numbered, versioned ADR in the repo - Title, Status, Context, Decision, Consequences (trade-offs), and a Compliance section naming the fitness function that enforces it; supersede by a new ADR, never an edit.
tags:
    - go
    - workflow
    - architecture
steps:
    - decide whether the choice is architecturally significant (a boundary, dependency, style, or structure others must follow)
    - write a numbered ADR under docs/adr/ with Title, Status, Context, Decision, Consequences, Compliance, Notes
    - state the Decision affirmatively ("we will...") and record the trade-offs in Consequences
    - name the fitness function that enforces the decision in the Compliance section
    - commit the ADR with the change it governs; supersede later by a new ADR linking back, never an edit
composes_rules:
    - commits
---

# Architecture decision records workflow

> Capture every architecturally-significant decision as a short, numbered, versioned markdown file in
> the repo, so the WHY survives and is not re-litigated. An ADR records the context, the decision, and
> its trade-offs, and names the fitness function that enforces it. Decisions are immutable: a changed
> mind is a NEW ADR that supersedes the old one by link, not an edit.

## When to use

- Choosing a boundary, dependency, architecture style, or structure others must follow.
- Adding a significant external dependency (record it alongside [[dependency-addition]]).
- Reversing or superseding a prior architectural decision.

## Steps

1. **Significance check.** Record an ADR only when the decision is architecturally significant - it
   affects structure, a boundary, a cross-cutting dependency, or a convention others must follow.
   Skip ADRs for local, easily-reversed choices.
2. **Write the ADR** as a numbered file (`docs/adr/NNNN-title.md`) with: **Title** (numbered) →
   **Status** (Proposed / Accepted / Superseded) → **Context** (forces at play) → **Decision**
   (affirmative, "we will...") → **Consequences** (the trade-offs, good and bad) → **Compliance**
   (which fitness function enforces it) → **Notes**.
3. **State the decision affirmatively** and be honest in Consequences - per the first law, there is
   always a trade-off; if you cannot name one, you have not looked hard enough.
4. **Compliance.** Name the [[fitness-functions]] check (depguard rule, go-arch-lint test, complexity
   budget) that enforces the decision. A decision with no enforcing check is governed by review only -
   say so explicitly.
5. **Commit and supersede.** Commit the ADR in the change it governs (see [[commits]]). To change a
   decision, add a new ADR that supersedes the old one by link and flip the old Status to Superseded -
   never rewrite history.

## Anti-patterns to reject

- **Covering your assets:** deferring a decision past the last responsible moment.
- **Groundhog Day:** a decision with no recorded context/justification, so it gets re-argued.
- **Email-driven architecture:** the decision lives in chat/email, not the single in-repo record.

## References

- rule: [[commits]], [[fitness-functions]]
- workflow: [[dependency-addition]], [[code-review]]
