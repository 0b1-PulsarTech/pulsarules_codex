---
id: usecase-layout
name: Use-case layout
---

Governs the shape of a use case: a `UseCase` struct holding a consumer-declared `Repository` port
and the per-request principal, one action per file with a typed `Input`, infra errors mapped to the
domain-error type, and a factory constructor. Reach for this when creating a new use-case package or
adding an action to one. The use case stays pure - authorization is gated at the call site, not
inside it, and the principal is read only for identity/audit. Not rest-adapter or grpc-adapter,
which are the thin transport shells that call into this; usecase-layout is the business logic
itself.

The rules below are the composed usecase-layout rule.
