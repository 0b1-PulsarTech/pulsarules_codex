---
id: user-from-context
name: Principal (user) from context
description: The per-request principal is resolved from JWT claims by a factory registered in the injector; the use case declares it as a constructor argument; the domain never imports JWT types.
tags:
    - go
    - bootstrap
    - auth
dependencies:
    - dependency-injection
    - authorization
---

# Principal (user) from context

> The per-request principal is resolved from JWT claims by a factory registered in the injector; the
> use case declares it as a constructor argument; no manual threading of the user through every
> call; the domain never imports JWT types.

Reference tools: a JWT middleware that stores claims on context (`jwt.ClaimsFromContext`);
a permission engine whose `Set` is rebuilt from the claim (see [[authorization]]).

{{define "when"}}
- Wiring the per-request principal (operator/user) into use cases.
- Loading identity/permissions from a JWT without re-parsing it in the domain.
{{end}}

{{define "recipe"}}
Register the principal as a factory:

```go
// bootstrap
remy.RegisterConstructorArgs1Err(inj, remy.Factory[entities.Principal], NewPrincipalFromContext)
```

The factory reads claims from context (the only place JWT appears outside middleware):

```go
// internal/domain/usecases/auth/user_from_context.go
func NewPrincipalFromContext(ctx context.Context) (entities.Principal, error) {
    claims, ok := jwt.ClaimsFromContext(ctx)
    if !ok {
        return entities.Principal{}, apperr.Unauthenticated()
    }
    perms, err := permits.Unmarshal(claims.Permissions)
    if err != nil {
        return entities.Principal{}, fmt.Errorf("unmarshal permissions: %w", err)
    }
    return entities.Principal{
        ID:          claims.Subject,
        TenantID:    claims.TenantID,
        Username:    claims.Username,
        Permissions: perms,
    }, nil
}
```

The use case takes the principal as a constructor arg:

```go
type UseCase struct {
    repo            Repository
    associatedUser  entities.Principal // request-scoped
}

func New(associatedUser entities.Principal, repo Repository) UseCase {
    return UseCase{repo: repo, associatedUser: associatedUser}
}
```

Register as a factory (it carries the request-scoped principal):

```go
remy.RegisterConstructorArgs2(inj, remy.Factory[UseCase], New)
```
{{end}}

{{define "forbidden"}}
- Returning a nullable principal; importing JWT types into the domain.
- Re-parsing the JWT inside a use case; threading the user through every function argument.
- Caching the principal in the token (permissions reload per request).
{{end}}

{{define "validation"}}
- [ ] Principal registered as `Factory[entities.Principal]` from context.
- [ ] Use cases take the principal as a constructor arg; no manual threading.
- [ ] JWT types confined to the factory + middleware; domain imports none.
- [ ] Permissions unmarshalled per request (not cached in the token).
{{end}}
