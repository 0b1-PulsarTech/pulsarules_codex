---
id: api-implementation
name: API implementation
description: Implement a use case, expose it over REST or gRPC, guard it, instrument it, test it end-to-end.
tags:
    - go
    - workflow
    - api
steps:
    - use case (UseCase struct, consumer port, principal, typed Input, factory)
    - persistence (schema/query/migration/repository; transactions if multi-write)
    - authorization (module schema, Has in the use case, route guard is coarse pre-check)
    - transport (REST via rest-adapter or gRPC via grpc-adapter; parse-call-map-respond)
    - cross-module (facade port declared in this package)
    - observability (injected Tracer, Start + defer span.End)
    - resilience (retry decorator for transient external calls)
    - tests (unit with mocks; E2E via engine factory; gRPC on bufconn)
    - gate and commit
composes_rules:
    - transport
    - authorization
    - interop
    - grpc
    - testing
    - commits
composes_patterns:
    - usecase-layout
    - rest-adapter
    - grpc-adapter
    - observability
    - retry-backoff
    - integration-tests
---

# API implementation workflow

> Implement a use case, expose it over REST or gRPC, guard it, instrument it, test it end-to-end.

## When to use

- Adding a REST or gRPC endpoint backed by a use case.

## Steps

1. **Use case** (`internal/domain/usecases/<feature>/`): `UseCase` struct with the consumer-declared
   `Repository` port and the per-request principal; one action per file; typed `Input`; infra errors
   mapped to the domain-error type; register as a factory. (pattern: [[usecase-layout]],
   [[user-from-context]], rule: [[transport]])
2. **Persistence** (if needed): schema/query/migration per the migration-creation workflow; repository
   per [[repository-layout]]; `transactions` if multi-write. (rules: [[database]],
   [[eventing-outbox]] if a side effect)
3. **Authorization:** declare the module's permission schema; check `Has` in the use case; a route
   guard is only a coarse pre-check. (rule: [[authorization]], pattern: [[permitek-schema]])
4. **Transport:** REST via [[rest-adapter]] (typed DTOs, OpenAPI, router contract) or gRPC via
   [[grpc-adapter]] (thin shell, proto confined to adapter). Handler only parses -> calls use case
   -> maps -> responds. (rule: [[transport]], [[grpc]])
5. **Cross-module:** if the use case calls another feature, depend on a facade port declared in this
   package. (rule: [[interop]])
6. **Observability:** inject a `Tracer`; `Start` + `defer span.End()`; record errors on the
   originating span. (pattern: [[observability]])
7. **Resilience:** wrap transient external calls with a retry decorator. (pattern: [[retry-backoff]])
8. **Tests:** unit tests against mocks; E2E via the engine factory booting the real server; gRPC on
   `bufconn`. (rule: [[testing]], patterns: [[colocated-mocks]], [[integration-tests]])
9. **Gate & commit:** `task tools:lint`, `task main:test`; commit per [[commits]].

## References

- rules: [[transport]], [[authorization]], [[interop]], [[grpc]], [[testing]]
- patterns: [[usecase-layout]], [[rest-adapter]], [[grpc-adapter]], [[observability]],
  [[retry-backoff]], [[integration-tests]]
