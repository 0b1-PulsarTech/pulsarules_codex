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
{{end}}

{{define "forbidden"}}
- Committing `.proto` files or copying generated Go from the proto repo.
- Leaking proto types into domain code; calling a proto client directly from a use case.
- Business logic in the adapter or the gRPC server.
- Unaliased proto imports; relying only on `protoc-gen-validate` for business rules.
{{end}}

{{define "validation"}}
- [ ] No `.proto` files or copied generated code; proto module version pinned.
- [ ] Proto packages aliased `<service>v<n>`.
- [ ] Consumer port declared with domain types; adapter in `internal/infra/grpc/<service>grpc/`.
- [ ] gRPC server is a thin shell embedding `Unimplemented...Server`; logic in the use case.
- [ ] Proto types confined to adapter/server + `mappers.go`; domain never imports proto.
- [ ] RPC errors wrapped; client registered as singleton.
{{end}}
