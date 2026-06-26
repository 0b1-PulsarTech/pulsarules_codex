---
id: rest-adapter
name: REST adapter
---

## Mandatory workflow

1. Expose a use case over REST with typed `Input`/`Output` DTOs that drive OpenAPI generation. Keep entity <-> DTO
   mapping in a small `mappers.go`.
2. Make the handler strictly parse -> call `uc.Execute(ctx, input)` -> map entities to a response DTO -> respond. No
   business rules, SQL, or domain invariants in the handler.
3. Keep the handler package the only importer of the REST framework adapter (e.g. Fuego via a thin web adapter). The
   domain imports none of it.
4. Register routes typed through a router contract (`var _ web.RouterContract = Handlers{}`) with a `GroupName()`,
   `Middlewares()`, and `RegisterRoutes(web.RouteRegistrar)`. Preserve OpenAPI generics; avoid untyped route
   registration.
5. Apply a per-route permission guard middleware as a coarse fail-fast pre-check; the access gate at the call site (the
   service layer) is the authorization authority, not the handler. The use case stays pure.
6. Return domain errors as the domain-error type; let middleware map them to HTTP status. Never hand-write HTTP statuses
   for domain errors in the handler or use case.
7. Give the handler a service-CONSTRUCTOR field `newSvc func(ctx)(Service, error)` (resolved via
   `remy.GetWithContext`), not the use case's deps passed down - this reduces coupling. The handler calls `newSvc(ctx)`
   to get the request-scoped service (carrying the principal); tests wrap their mock in that same constructor. Routers
   are built locally by the app's `Routers()` builder and mounted at the composition root.

## Validation checklist

- [ ] Handler package is the only importer of the REST framework adapter.
- [ ] `Input`/`Output` are transport DTOs with validate tags; entities mapped in `mappers.go`.
- [ ] Handler only parses, calls the use case, maps, and responds.
- [ ] Errors returned as the domain-error type; no status codes hand-written.
- [ ] Routes registered typed; `RouterContract` asserted; permission guard applied.
- [ ] Authorization enforced by the access gate at the call site; the route guard is only fail-fast.

## Forbidden actions

- Leaking proto/transport types into the domain; business logic in the handler.
- Permission decisions made only in the handler (the access gate at the call site is the authority).
- Hand-writing HTTP statuses for domain errors (return the domain-error type).
- Global routers/middleware; untyped route registration that loses OpenAPI generics.

## Expected outputs

- Typed request/response DTOs driving OpenAPI; a parse-call-map-respond handler.
- Routes registered through a typed `RouterContract` with a permission guard.
- Domain errors mapped to HTTP status by middleware; the use case transport-agnostic.
