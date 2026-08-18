---
id: usecase-layout
name: Use-case layout
description: A UseCase struct with a consumer-declared Repository port and the per-request principal; one action per file with a typed Input; infra errors translated to the domain-error type; a factory constructor.
tags:
    - go
    - domain
dependencies:
    - transport
    - authorization
    - dependency-injection
composes:
    - user-from-context
---

# Use-case layout

> A `UseCase` struct holding its consumer-declared `Repository` port and the per-request principal;
> one action per file with a typed `Input` and an entity+error return; infra errors translated to
> the domain-error type; `mockgen` mocks; a factory constructor.

{{define "when"}}
- Creating a feature use-case package.
- Adding an action (method) to a use case.
- Declaring the `Repository` (or other ports) the use case depends on.
{{end}}

{{define "recipe"}}
```
internal/domain/usecases/<feature>/
├── <feature>.go            # UseCase struct + Repository port + New()
├── create_<thing>.go       # one action per file
├── list_<thing>.go
├── <feature>_mock_test.go  # mockgen-generated
└── usecase_test.go
```

```go
// internal/domain/usecases/thing/thing.go
package thing

type Repository interface { // consumer-declared, smallest method set
    Insert(ctx context.Context, l entities.Thing) (entities.Thing, error)
    Get(ctx context.Context, id entities.ID) (entities.Thing, error)
}

type UseCase struct {
    repo           Repository
    associatedUser entities.Principal
}

func New(associatedUser entities.Principal, repo Repository) UseCase {
    return UseCase{repo: repo, associatedUser: associatedUser}
}
```

One action per file, typed `Input`:

```go
// internal/domain/usecases/thing/create_thing.go
type CreateInput struct {
    Name    string
    Contact string
}

func (uc UseCase) Create(ctx context.Context, in CreateInput) (entities.Thing, error) {
    // The use case is PURE: authorization is enforced by the access gate at the CALL SITE
    // (see authorization). The principal is read only for identity/audit, never for the authz check.
    l, err := uc.repo.Insert(ctx, entities.Thing{
        Name:      in.Name,
        Contact:   in.Contact,
        CreatedBy: uc.associatedUser.ID, // identity/audit, not authorization
    })
    if err != nil {
        return entities.Thing{}, fmt.Errorf("insert thing: %w", err)
    }
    return l, nil
}
```

Register as a factory; generate the mock:

```go
// di.go
remy.RegisterConstructorArgs2(inj, remy.Factory[UseCase], New)
```

```go
//go:generate mockgen -source=thing.go -destination=thing_mock_test.go -package=thing
```
{{end}}

{{define "forbidden"}}
- Package-level state or globals; using the injector inside the use case.
- A permission check inside the use case (gate at the call site via the access gate; keep the use case
  pure); an `associatedUser`/principal field the use case never reads.
- Handlers calling a repository directly (must route through the use case).
- Returning infra types (generated rows, proto, transport DTOs) from use-case methods.
- Importing another use case's package to call it (use a facade port).
- Positional arguments instead of a typed `Input` struct.
{{end}}

{{define "validation"}}
- [ ] `UseCase` holds consumer-declared ports; the principal only if read for identity/audit.
- [ ] `Repository` interface small and declared in the use-case package.
- [ ] One action per file; each takes a typed `Input` and returns entity + error.
- [ ] Use case is pure (no permission check); authorization gated at the call site; infra errors mapped
  to the domain-error type.
- [ ] No infra/proto/transport types in public signatures.
- [ ] Constructor registered as `Factory[UseCase]`; `mockgen` directive present.
{{end}}

{{define "outputs"}}
- A factory constructor; a `mockgen` directive and a colocated `usecase_test.go`.
{{end}}
