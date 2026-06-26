---
id: interop
name: Cross-module interop via facades
description: Any call crossing a feature/module boundary goes through a consumer-declared facade port; the provider is unaware of the consumer.
tags:
    - go
    - interop
dependencies:
    - transport
    - dependency-injection
linters:
    - depguard
---

# Cross-module interop via facades

> Any call crossing a feature/module boundary goes through a consumer-declared facade port,
> implemented in an `interop/facades/` package; the provider is unaware of the consumer; the
> injector resolves the port structurally.

Applies to: cross-feature and cross-module calls.

{{define "when"}}
- One feature/module needs to call another's use case.
- Designing the seam between two bounded contexts.
- Mocking a cross-feature dependency for tests.
{{end}}

{{define "must"}}
1. Any call crossing a module boundary (packages not in the same feature directory) goes through a
   facade interface declared in the **consuming** package, with the smallest method set the consumer
   needs.
2. Implement the facade struct in `internal/domain/interop/facades/<feature>facade/` (app) or
   `libs/<name>/interop/facades/` (lib), wrapping the provider's use case. The consumer imports
   nothing from the provider's internals; the provider is unaware of the consumer.
3. Register the facade as a singleton; the injector resolves the consumer's port to it structurally
   (`DuckTypeElements: true`).
4. Tests mock the port interface (declared in the consumer package), not the facade struct; generate
   the mock with `mockgen` to `_mock_test.go` in the same package (see [[testing]]).
{{end}}

{{define "forbidden"}}
- A use case importing another use case's package directly to call it.
- A facade exposing more methods than the consuming port declares.
- Circular facade dependencies; feature packages importing each other directly.
- Mocking the facade struct instead of the consumer-declared port.
{{end}}

{{define "validation"}}
- [ ] Cross-module calls go through a consumer-declared facade port.
- [ ] Facade struct lives in `interop/facades/`; registered as a singleton.
- [ ] Consumer imports nothing from the provider's internals.
- [ ] Tests mock the port interface, not the facade.
{{end}}
