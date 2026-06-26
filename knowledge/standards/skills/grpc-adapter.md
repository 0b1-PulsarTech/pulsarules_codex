---
id: grpc-adapter
name: gRPC adapter
---

## Mandatory workflow

1. Keep proto contracts in a sibling proto repo and import them as its Go module. Never commit `.proto` files here;
   never copy generated Go - import the module and pin its version in `go.mod`.
2. Alias proto packages `<service>v<n>` (`foov1`, `foov1grpc`); enforce with `importas`.
3. When consuming: declare a port interface in the use-case/domain package with domain-typed methods (no proto types in
   signatures).
4. Implement the adapter in `internal/infra/grpc/<service>grpc/` wrapping the generated client; per method: map domain
   to proto, call the client, map proto to domain, wrap errors. Keep mapping in a small pure `mappers.go`.
5. When serving: put the handler in `internal/transport/grpc/<service>server/`, embed the generated
   `Unimplemented...Server`, take the use case, and be a thin shell: parse proto -> call use case -> map entities to
   proto response.
6. Keep business logic and invariants in the use case; do not rely on `protoc-gen-validate` alone.
7. DI: register the gRPC client connection as a singleton; the adapter as a singleton (or factory if request-scoped).
   The use case depends only on the port.
8. Confine proto types to the adapter/server + `mappers.go`; the domain never imports proto.

## Validation checklist

- [ ] No `.proto` files or copied generated code; proto module version pinned.
- [ ] Proto packages aliased `<service>v<n>`.
- [ ] Consumer port declared with domain types; adapter in `internal/infra/grpc/<service>grpc/`.
- [ ] gRPC server is a thin shell embedding `Unimplemented...Server`; logic in the use case.
- [ ] Proto types confined to adapter/server + `mappers.go`; domain never imports proto.
- [ ] RPC errors wrapped; client registered as singleton.

## Forbidden actions

- Committing `.proto` files or copying generated Go from the proto repo.
- Leaking proto types into domain code; calling a proto client directly from a use case.
- Business logic in the adapter or the gRPC server.
- Unaliased proto imports; relying only on `protoc-gen-validate` for business rules.

## Expected outputs

- A consumer port with domain types; an adapter wrapping the generated client; a thin gRPC server shell.
- Proto aliased `<service>v<n>` and confined to the adapter/server + `mappers.go`.
- The use case depends only on the port; the gRPC client registered as a singleton.
