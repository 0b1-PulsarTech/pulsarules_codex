---
id: code-review
name: Code review
description: Review a change for correctness, rule compliance, and the stop signs before merge.
tags:
    - go
    - workflow
    - review
    - architecture & placement; no cross-app imports; non-API under internal/; source deps point inward (no infra/transport/driver in domain)
    - stop signs (no globals, no init side effects, no panic outside main, no raw SQL in handlers, no dbgen out of repos, no direct cross-module calls, no transport in domain, no silent error swallowing)
    - style (formatter, imports, doc comments, casing, any, consumer interfaces)
    - errors & logging (%w wrapping, errors.Is/As, no status from use cases, slog typed attrs, no PII)
    - persistence (ent/sqlc/atlas/goverter, no tenant_id predicate, real-DB repo tests)
    - concurrency (goroutine ownership, ctx-first, errgroup, ctx.Done loops, -race clean)
    - security (secrets via config, boundary validation, sqlc-only SQL, JWT at middleware, digest-pinned images)
    - tests (colocated, table-driven, integration tag, real DB, no SQL mock, no time.Sleep)
    - gate on lint + test; each new architectural invariant backed by a fitness function
    - commit hygiene (one logical change, emoji format, no attribution trailer)
composes_rules:
    - code-placement
    - dependency-rule
    - fitness-functions
    - startup
    - dependency-injection
    - database
    - interop
    - transport
    - errors
    - effective-go
    - naming
    - imports
    - types
    - logging
    - concurrency
    - security
    - testing
    - commits
composes_patterns:
    - repository-layout
---

# Code review workflow

> Review a change for correctness, rule compliance, and the stop signs before merge.

## When to use

- Reviewing a PR/diff.
- Self-review before requesting review.

## Steps

1. **Architecture & placement:** does the code live in the right layer? No cross-app imports, no
   libs importing apps, non-API code under `internal/`; source dependencies point inward (no
   infra/transport/driver/framework import in the domain). (rules: [[code-placement]],
   [[dependency-rule]])
2. **Stop signs:** no global mutable state; no `init()`/package-load side effects; no `panic`
   outside `main`; no raw SQL in handlers/use cases; no generated row types returned from
   repositories; no direct cross-module calls (use facades); no transport imports in domain; no
   silent error swallowing. (rules: [[startup]], [[dependency-injection]], [[database]],
   [[interop]], [[transport]], [[errors]])
3. **Style:** formatter run; three-group imports; doc comments on exported symbols; correct
   acronym casing; `any` not `interface{}`; consumer-declared small interfaces. (rule:
   [[effective-go]], [[naming]], [[imports]], [[types]])
4. **Errors & logging:** boundary errors wrapped with `%w` + verb prefix; sentinels compared with
   `errors.Is`/`As`; no status codes from use cases; `slog` typed attrs only; no secrets/PII in
   logs. (rules: [[errors]], [[logging]])
5. **Persistence:** schema via ent; queries via sqlc; migrations via Atlas; goverter mapping; no
   `tenant_id` predicate (connection-level); repo tests use a real DB. (rule: [[database]],
   pattern: [[repository-layout]])
6. **Concurrency:** every goroutine owned; `ctx`-first; `errgroup`/`WaitGroup`; worker loops on
   `ctx.Done()`; `-race` clean. (rule: [[concurrency]])
7. **Security:** secrets via config only; inputs validated at the boundary; SQL via generated
   queries; JWT verified at middleware; images pinned by digest. (rule: [[security]])
8. **Tests:** colocated, table-driven, `t.Parallel()`; repo/E2E behind `//go:build integration`
   with a real DB; no SQL mock; no `time.Sleep`. (rule: [[testing]])
9. **Gate:** `task tools:lint` and `task main:test` pass; every new architectural invariant is
   backed by an automated fitness function, not just review. (rule: [[fitness-functions]])
10. **Commit hygiene:** one logical change per commit; emoji Conventional Commit format; no
    tool-attribution trailer. (rule: [[commits]])

## References

- generated skill: `project-router` (enforcement checklist)
