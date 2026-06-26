---
id: usecase-layout
name: Use-case layout
---

## Mandatory workflow

1. Create the use-case package under `internal/domain/usecases/<feature>/`. The `UseCase` struct holds its
   consumer-declared `Repository` port and the per-request `associatedUser entities.Principal`.
2. Declare the `Repository` (and any other ports) in the use-case package - the smallest method set the use case needs (
   consumer-declared). One port interface per dependency.
3. Write one action per file (`create_thing.go`, `list_thing.go`), each taking a typed `Input` struct and returning
   `(entity, error)`. Do not use positional arguments.
4. Keep the use case PURE: do NOT check permissions inside it. Authorization is enforced by the access gate
   (`Guard`/`GuardQuery`/`GuardCommand`) at the CALL SITE (see authorization). The principal is read only for
   identity/audit (e.g. stamping `CreatedBy`), never for the authz decision; don't keep a principal field the use case
   never reads.
5. Map infra errors to the domain-error type at the use-case boundary (
   `fmt.Errorf("create thing: %w", apperr.ErrForbidden)`). Return entities, never infra/proto/transport types.
6. Provide a factory constructor `New(associatedUser entities.Principal, repo Repository) UseCase`; register it as
   `Factory[UseCase]` (it carries the request-scoped principal).
7. Generate the repository mock with
   `//go:generate mockgen -source=<feature>.go -destination=<feature>_mock_test.go -package=<feature>`. Write a
   name-matched `usecase_test.go`.
8. Keep the domain transport-agnostic: the `Input`, use case, and `Output`/entities import no `net/http`, `grpc`, proto,
   or web framework. Cross-module calls go through a facade port, never a direct import.

## Validation checklist

- [ ] `UseCase` holds consumer-declared ports; the principal only if read for identity/audit.
- [ ] `Repository` interface small and declared in the use-case package.
- [ ] One action per file; each takes a typed `Input` and returns entity + error.
- [ ] Use case is pure (no permission check); authorization gated at the call site; infra errors mapped to the
  domain-error type.
- [ ] No infra/proto/transport types in public signatures.
- [ ] Constructor registered as `Factory[UseCase]`; `mockgen` directive present.

## Forbidden actions

- Package-level state or globals; using the injector inside the use case.
- A permission check inside the use case; an `associatedUser`/principal field the use case never reads.
- Handlers calling a repository directly (must route through the use case).
- Returning infra types (generated rows, proto, transport DTOs) from use-case methods.
- Importing another use case's package to call it (use a facade port).
- Positional arguments instead of a typed `Input` struct.

## Expected outputs

- A use-case package with a `UseCase` struct, consumer-declared ports, and one action per file.
- A pure use case (authorization gated at the call site); infra errors mapped to the domain-error type.
- A factory constructor; a `mockgen` directive and a colocated `usecase_test.go`.
