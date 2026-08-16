# `patterns/`

Implementation recipes - the **how**. Each file shows how to apply the rules with working,
self-contained Go code. Code uses generic placeholders (`Foo`, `Bar`, the caller's domain type);
named tools (`remy`, `ent`, `sqlc`, `atlas`, `goverter`, `Fuego`, `permitek`, ...) appear as
canonical reference implementations, not requirements.

Patterns are known-good recipes for recurring problems - mandatory in shape, adaptable in detail.
They are grouped into themed subdirectories; the ID in each file's frontmatter is the canonical
reference used in `skills.yaml` composition lists.

## `bootstrap/` - wiring, DI, and app skeleton

| Pattern                                                    | Topic                                     |
|------------------------------------------------------------|-------------------------------------------|
| [app-skeleton.md](bootstrap/app-skeleton.md)               | New deployable binary skeleton            |
| [bootstrap-and-di.md](bootstrap/bootstrap-and-di.md)       | Wiring main -> config -> DB -> server     |
| [config-layout.md](bootstrap/config-layout.md)             | Typed Config + env binder                 |
| [embedded-migrations.md](bootstrap/embedded-migrations.md) | Embed SQL + in-house runner               |

## `domain/` - use cases, domain logic, and bounded output shapes

| Pattern                                                  | Topic                                        |
|----------------------------------------------------------|----------------------------------------------|
| [usecase-layout.md](domain/usecase-layout.md)           | Use-case struct + per-action files + port    |
| [user-from-context.md](domain/user-from-context.md)     | Per-request principal from JWT context       |
| [permitek-schema.md](domain/permitek-schema.md)         | Per-module reflection-free permission schema |
| [proposal-window.md](domain/proposal-window.md)         | Sliding-window bounded numeric value         |
| [template-engine.md](domain/template-engine.md)         | Variable-registry render engine              |
| [rule-engine.md](domain/rule-engine.md)                 | Data-driven JSON rule selection + registry   |

## `infra/` - persistence, transport, and external integration recipes

| Pattern                                                      | Topic                                         |
|--------------------------------------------------------------|-----------------------------------------------|
| [ent-schema.md](infra/ent-schema.md)                         | Schema-as-code entity authoring (ent)         |
| [sqlc-queries.md](infra/sqlc-queries.md)                     | Typed query authoring (sqlc)                  |
| [goverter-mapping.md](infra/goverter-mapping.md)             | Compile-time row -> DTO mapping               |
| [repository-layout.md](infra/repository-layout.md)           | sqlc + repo struct + goverter mapper          |
| [grpc-adapter.md](infra/grpc-adapter.md)                     | Wrap a proto client / serve gRPC              |
| [rest-adapter.md](infra/rest-adapter.md)                     | Expose a use case over REST                   |
| [observability.md](infra/observability.md)                   | Tracing abstraction; OTel adapter at infra    |
| [retry-backoff.md](infra/retry-backoff.md)                   | Exponential backoff + jitter decorator        |
| [transactions.md](infra/transactions.md)                     | Mandatory multi-write transactions            |
| [event-sink-worker.md](infra/event-sink-worker.md)           | Relay worker loop + idempotent Sink           |
| [external-provider.md](infra/external-provider.md)           | Strategy port + per-provider package + DI     |

## `testing/` - test harness and mock patterns

| Pattern                                                        | Topic                                       |
|----------------------------------------------------------------|---------------------------------------------|
| [integration-tests.md](testing/integration-tests.md)           | DB-backed test harness + E2E engine factory |
| [colocated-mocks.md](testing/colocated-mocks.md)               | `<source>_mock_test.go` recipe              |

## `architecture/` - structural and behavioural design patterns

| Pattern                                                      | Topic                                         |
|--------------------------------------------------------------|-----------------------------------------------|
| [design-patterns.md](architecture/design-patterns.md)       | Which classic patterns used / avoided         |
| [observer-weakptr.md](architecture/observer-weakptr.md)     | `weak.Pointer` observer registry              |
