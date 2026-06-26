---
id: feature-development
name: Feature development
description: The mandatory router entry sequence for any code change - classify, load skills in composition order, implement, validate, commit.
tags:
    - go
    - workflow
steps:
    - classify the task against the router dispatch table; union matching rows
    - load the baseline (go-style, errors-logging, code-placement, code-minimalism)
    - load dispatch matches in composition order (placement, bootstrap, domain, persistence, eventing, transport, cross-cutting, tests, commits)
    - implement applying each loaded skill's rules
    - run each loaded skill's validation checklist; confirm no forbidden action
    - gate on task tools:fmt / tools:lint / main:test
    - commit per the commits rule
composes_rules:
    - code-placement
    - commits
    - minimalism
composes_patterns:
    - bootstrap-and-di
    - usecase-layout
---

# Feature development workflow

> The mandatory entry sequence for any code change: classify the task -> load the applicable skills
> in composition order -> implement -> validate against each loaded skill's checklist -> commit.
> This is the human/machine expression of the `project-router` skill.

## When to use

- Starting any non-trivial code change (new feature, endpoint, entity, query, worker, test).

## Steps

1. **Classify the task.** Match it against the router dispatch table (see
   `generated/skills/project-router/SKILL.md`). A task often matches several rows - union them.
2. **Load the baseline (every Go change):** `go-style`, `errors-logging`, `code-placement`,
   `code-minimalism`. Load conditionally: `concurrency` (goroutines/channels/workers), `security`
   (secrets/PII/SQL/JWT/containers), `design-patterns` (architectural decision).
3. **Load dispatch matches** in composition order:
    1. `code-placement` - decide where it lives.
    2. `app-bootstrap` - wiring/DI (if a new app or new registration).
    3. Domain: `usecase-layout` + the relevant domain skill.
    4. Persistence: `database-persistence`, then `transactions` if multi-write.
    5. Eventing: `eventing-outbox` (+ `retry-backoff`) for side effects.
    6. Transport: `rest-adapter` / `grpc-adapter` + `transport-interop` + `authorization`.
    7. Cross-cutting: `observability`; baseline throughout.
    8. `integration-tests` - prove it.
    9. `commits` - last, before committing.
4. **Implement**, applying each loaded skill's rules as you go.
5. **Validate:** run each loaded skill's Validation checklist; confirm no Forbidden action was
   taken; respect the stop signs (no globals, no `init()` side effects, no cross-app imports, no
   `panic` outside `main`, no raw SQL in handlers, no generated row types out of repos, no direct
   cross-module calls, no transport imports in domain, no silent error swallowing).
6. **Gate:** `task tools:fmt`, `task tools:lint`, `task main:test` pass.
7. **Commit:** apply the `commits` skill to the message.

## Conflict resolution

The MOST SPECIFIC skill wins on a detail it owns (e.g. `transactions` owns the multi-write defer
pattern). The baseline skills are never skipped. If two skills seem to conflict: the domain skill
defines WHAT, the pattern skill defines HOW, the rule/baseline skill is the non-negotiable floor.

## References

- rules: [[code-placement]], [[commits]], [[minimalism]]
- patterns: [[bootstrap-and-di]], [[usecase-layout]]
- generated skill: `project-router`
