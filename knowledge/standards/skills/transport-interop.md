---
id: transport-interop
name: Transport & interop
---

## Mandatory workflow

1. Keep the domain transport-agnostic: the use case's `Input`, the use case, and its `Output`/entities import no
   `net/http`, `google.golang.org/grpc`, proto package, or web framework.
2. Shape handlers strictly: parse request to validate wire format (required fields, max length) -> call
   `uc.Execute(ctx, input)` -> map entities to a response DTO -> write status/body. No business rules, SQL, or domain
   invariants in the handler.
3. Put wire-format validation at the boundary; enforce business invariants in the use case. The handler knows nothing
   about DB rows; the use case knows nothing about HTTP/gRPC status or JSON.
4. Let domain errors flow out as the domain-error type; middleware maps them to the transport status (
   see [[errors-logging]]). Never write a status code from the use case.
5. Route any call crossing a module boundary through a facade: declare the port in the consuming package with the
   smallest method set it needs; implement the facade struct in `internal/domain/interop/facades/<feature>facade/`
   wrapping the provider's use case. The consumer imports nothing from the provider's internals; the provider is unaware
   of the consumer.
6. Register the facade as a singleton; the injector resolves the consumer's port to it structurally (
   `DuckTypeElements: true`).
7. Test by mocking the consumer-declared port (declared in the consumer package), not the facade struct; generate the
   mock with `mockgen` to `_mock_test.go` in the same package.

## Validation checklist

- [ ] No `net/http`/grpc/proto/web-framework import inside `internal/domain/`.
- [ ] Handler only parses, calls the use case, maps, and responds.
- [ ] Wire-format validation at the boundary; business invariants in the use case.
- [ ] Domain errors flow out as the domain-error type; no status codes written from the use case.
- [ ] Cross-module calls go through a consumer-declared facade port in `interop/facades/`.
- [ ] Facade registered as a singleton; tests mock the port, not the facade struct.

## Forbidden actions

- Importing `net/http`/grpc/proto/web-framework inside the domain.
- Business rules, permission checks, or domain validation in a handler.
- Returning raw generated row types / proto / `*sql.Rows` from a use case.
- A use case that can only be called from one transport.
- A use case importing another use case's package directly to call it.
- A facade exposing more methods than the consuming port declares; circular facade dependencies.
- Mocking the facade struct instead of the consumer-declared port.

## Expected outputs

- A transport-agnostic use case callable unchanged from HTTP, gRPC, CLI, or a background job.
- Thin parse-call-map-respond handlers; domain errors mapped to status by middleware.
- Cross-module calls via consumer-declared facade ports; tests mock the port.
