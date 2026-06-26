---
id: transport-interop
name: Transport & interop
---

Governs two related boundaries: domain code that imports no `net/http`, `grpc`, or proto, so a use
case stays callable from HTTP, gRPC, or a background job unchanged, and cross-module calls that go
through a consumer-declared facade port in `interop/facades` rather than importing another feature's
internals directly. Reach for this when writing a handler, deciding where validation vs business
logic lives, or when one feature needs to call another's use case. rest-adapter and grpc-adapter
build the concrete transport shells on top of this rule; transport-interop is what keeps them thin.

The rules below are the composed transport-interop rule.
