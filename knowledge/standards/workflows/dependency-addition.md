---
id: dependency-addition
name: Dependency addition
description: Decide whether to add a new external dependency; if yes, wrap it as an Adapter behind a port and allow-list it in the linter.
tags:
    - go
    - workflow
    - dependencies
steps:
    - walk the minimalism ladder (necessity -> stdlib -> platform -> existing dep -> one line -> implement)
    - justify it as a design decision (maintained, narrow, well-fitted)
    - wrap it as an Adapter behind a consumer-declared port
    - allow-list it in depguard and pin in go.mod
    - if an external service, route through the HTTP gateway / proto adapter + retry/backoff
    - if it touches secrets/PII/SQL, apply security
    - add a tools/go.mod entry if it is a build/codegen tool
    - gate and commit
composes_rules:
    - minimalism
    - imports
    - build
    - transport
    - http-clients
    - grpc
    - security
    - commits
composes_patterns:
    - design-patterns
    - external-provider
    - retry-backoff
---

# Dependency addition workflow

> Decide whether to add a new external dependency; if yes, wrap it as an Adapter behind a port and
> allow-list it in the linter.

## When to use

- About to add a third-party dependency.
- Reviewing a PR that adds one.

## Steps

1. **Walk the minimalism ladder first:** necessity -> stdlib -> platform feature -> an
   already-imported dep -> one line -> implement. Add a dependency only when none of those satisfy
   the task. (rule: [[minimalism]])
2. **Justify it as a design decision:** does it pull in a maintained, narrow, well-fitted library?
   Record the rationale (a `// simplification:`/PR note if a corner is cut). (pattern:
   [[design-patterns]], rule: [[minimalism]])
3. **Wrap it as an Adapter** behind a consumer-declared port. The domain never imports the
   dependency directly. (pattern: [[design-patterns]], [[external-provider]], rule: [[transport]])
4. **Allow-list it** in `depguard` (the linter import allow-list) and pin it in `go.mod`
   (`go mod tidy`). (rule: [[build]], [[imports]])
5. **If it is an external service** (HTTP/gRPC), route through the HTTP gateway / a proto adapter;
   add retry/backoff for transient failures. (rules: [[http-clients]], [[grpc]]; pattern:
   [[retry-backoff]])
6. **If it touches secrets/PII/SQL**, apply [[security]].
7. **Add a `tools/go.mod` entry** if it is a build/codegen tool (not a runtime dep).
8. **Gate & commit:** `task tools:lint`, `task main:test`; commit `:package: feat(<scope>): add
   <dep> for <reason>` per [[commits]].

## References

- rules: [[minimalism]], [[imports]], [[build]], [[transport]], [[http-clients]], [[grpc]],
  [[security]], [[commits]]
- patterns: [[design-patterns]], [[external-provider]], [[retry-backoff]]
