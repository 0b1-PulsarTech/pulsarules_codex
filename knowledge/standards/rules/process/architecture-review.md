---
id: architecture-review
name: Architecture review
description: Scope an architecture review by churn, explore read-only and name friction rather than following a checklist, apply the deletion test, present candidates as a report with strength badges and no final interfaces, flag ADR conflicts only on real friction, and grill the chosen candidate before landing it.
tags:
    - process
    - architecture
---

# Architecture review

> Scope a review to the parts that change often (or to what the user named), explore read-only and
> name friction instead of following a rigid checklist, apply the deletion test, present candidates
> as a report with strength badges and no final interfaces, flag an ADR conflict only when the
> friction is real, then grill the chosen candidate before landing it via refactoring.

Applies to: reviewing a codebase or subsystem for architectural friction and deepening opportunities.

{{define "when"}}
- Reviewing a codebase or subsystem for architectural friction.
- Looking for refactors that improve testability and locality.
- An area keeps causing bugs or is hard to test through its interface.
{{end}}

{{define "must"}}
1. Scope before you scan (YAGNI). Deepening a module pays off by making future change to it easier,
   so weight the parts that change often. If the user named a module or pain point, take it.
   Otherwise walk back a stretch of `git log --oneline` for the hot spots and start there; read
   `CONTEXT.md` and any relevant ADRs first so you use the domain's real names.
2. Explore with a read-only Explore subagent. Do not follow rigid heuristics - note where you feel
   friction: understanding one concept means bouncing between many small modules; a module is
   shallow (interface nearly as complex as the body); pure functions were extracted for testability
   but the real bugs hide in how they are called (no locality); tight coupling leaks across a seam;
   a module is untested or hard to test through its current interface.
3. Apply the deletion test to each suspected shallow module: would deleting it concentrate
   complexity (deepen - the signal you want) or just move it around?
4. Present candidates as a report, not as code. Use the `archify` skill or a self-contained report
   written outside the repo tree. Each candidate states: files involved, the friction, the
   plain-English change, the benefit in terms of locality and leverage and how tests improve, a
   before/after sketch, and a strength badge (Strong / Worth exploring / Speculative). End with a
   top recommendation. Use `CONTEXT.md` names for the domain and `codebase-design` terms for the
   architecture. Do NOT propose final interfaces yet.
5. Flag an ADR conflict only when the friction is real enough to reopen the decision; mark it
   clearly rather than listing every refactor an ADR forbids.
6. Grilling loop. Once a candidate is chosen, run `grilling` to walk the decision tree (constraints,
   what sits behind the seam, which tests survive). Keep the domain model current via
   `domain-modeling` as concepts crystallize, and offer an ADR when a rejection carries a
   load-bearing reason a future review would need.
{{end}}

{{define "forbidden"}}
- Proposing final interfaces or editing code during the review phase.
- Scanning the whole codebase with no scoping when the churn points somewhere specific.
- Re-litigating a decision an ADR already settled without real, current friction.
- Drifting out of the `codebase-design` vocabulary into "service" / "component" / "boundary".
{{end}}

{{define "validation"}}
- [ ] Scope was chosen from recent churn or an explicit ask before scanning; ADRs and `CONTEXT.md`
  were read first.
- [ ] Exploration used a read-only subagent and named friction, not a rigid checklist.
- [ ] Each candidate was justified with the deletion test and framed in locality/leverage terms.
- [ ] Candidates were presented as a report; no interfaces were finalized before grilling.
- [ ] The chosen candidate went through `grilling` and lands via `refactoring`.
{{end}}
