---
id: grpc
name: gRPC
description: Proto contracts in a sibling repo imported as a module; proto aliased <service>v<n>; consumer port wraps the client; server is a thin shell; proto never leaks into domain.
tags:
    - go
    - grpc
dependencies:
    - transport
    - interop
linters:
    - importas
    - depguard
---

# gRPC

> Proto contracts live in a sibling proto repo, imported as a Go module (no `.proto` files or copied
> generated code here); proto packages aliased `<service>v<n>`; a consumer-declared port wraps the
> generated client; the gRPC server is a thin parse-call-map shell; proto types never leak into the
> domain.

Applies to: consuming and serving gRPC. Canonical reference: a sibling `<project>proto` module
imported as `github.com/<org>/<project>proto/gen/<service>-go`.

{{define "when"}}
- Consuming a gRPC service.
- Implementing a gRPC server an app exposes.
- Importing proto packages/stubs.
- Mapping proto messages to domain entities.
- Choosing a `codes.*` status for an RPC error.
- Making an outbound gRPC client call.
- Configuring the gRPC server for production.
{{end}}

{{define "catalog"}}
| Code | When |
|---|---|
| `InvalidArgument` | request fails validation (malformed/missing/out-of-range field). |
| `NotFound` | the referenced resource does not exist. |
| `AlreadyExists` | a create collides with an existing unique resource. |
| `PermissionDenied` | the caller is authenticated but lacks the permission. |
| `Unauthenticated` | the caller has no/invalid credentials. |
| `FailedPrecondition` | the system is not in a state the request requires (retrying the same request will not help). |
| `ResourceExhausted` | rate limit or quota exceeded. |
| `Unavailable` | transient failure, retryable; e.g. a dependency is down. |
| `Internal` | an invariant was violated / an unexpected server-side error. |
| `DeadlineExceeded` | the call did not complete before its deadline. |
{{end}}

{{define "must"}}
1. Proto contracts live in the sibling proto repo and are imported as its Go module. Never commit
   `.proto` files here; never copy generated Go - import the module and pin its version in
   `go.mod`.
2. Alias proto packages `<service>v<n>` (`foov1`, `foov1grpc`); `importas` enforces it.
3. Consuming: declare a port interface in the use-case/domain package with domain-typed methods (no
   proto types in signatures).
4. Implement the adapter in `internal/infra/grpc/<service>grpc/` wrapping the generated client; per
   method: map domain to proto, call the client, map proto to domain, wrap errors. Keep mapping in a
   small pure `mappers.go`.
5. Serving: handler in `internal/transport/grpc/<service>server/` embeds the generated
   `Unimplemented...Server`, takes the use case, and is a thin shell: parse proto -> call use case ->
   map entities to proto response.
6. Business logic and invariants stay in the use case; do not rely on `protoc-gen-validate` alone.
7. DI: register the gRPC client connection as a singleton; the adapter as a singleton (or factory if
   request-scoped). The use case depends only on the port.
8. Map every RPC error to a `codes.*` value from the catalog above, not a bare `codes.Unknown`/
   `codes.Internal` for everything.
9. Register a panic-recovery unary/stream interceptor on the server as a NAMED EXCEPTION to the
   general no-`recover()` rule (see [[errors]], [[concurrency]]): recover, log with `debug.Stack()`
   (the `bootstrap/scheduler.go` template), return `codes.Internal`, and let the request die alone -
   grpc-go does not recover handler panics itself.
10. Every outbound gRPC client call carries a deadline (`context.WithTimeout`/an inbound deadline
    propagated via the context); never call with an undeadlined context.
11. Reflection (`reflection.Register`) is enabled only in non-production builds/environments; it is
    disabled in production.
{{end}}

{{define "forbidden"}}
- Committing `.proto` files or copying generated Go from the proto repo.
- Leaking proto types into domain code; calling a proto client directly from a use case.
- Business logic in the adapter or the gRPC server.
- Unaliased proto imports; relying only on `protoc-gen-validate` for business rules.
- An outbound gRPC call with no deadline on its context.
- `reflection.Register` enabled in a production build/environment.
- A `recover()` in the gRPC server outside the panic-recovery interceptor.
{{end}}

{{define "validation"}}
- [ ] No `.proto` files or copied generated code; proto module version pinned.
- [ ] Proto packages aliased `<service>v<n>`.
- [ ] Consumer port declared with domain types; adapter in `internal/infra/grpc/<service>grpc/`.
- [ ] gRPC server is a thin shell embedding `Unimplemented...Server`; logic in the use case.
- [ ] Proto types confined to adapter/server + `mappers.go`; domain never imports proto.
- [ ] RPC errors wrapped and mapped to a specific `codes.*` value; client registered as singleton.
- [ ] The server registers a panic-recovery interceptor that logs with `debug.Stack()` and returns
  `codes.Internal`.
- [ ] Every outbound client call carries a deadline.
- [ ] Reflection is disabled in production.
{{end}}
