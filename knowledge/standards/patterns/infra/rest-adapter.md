---
id: rest-adapter
name: REST adapter
description: Expose a use case over REST with typed Input/Output DTOs driving OpenAPI; a parse-call-map-respond handler; a router contract with a permission guard; domain errors mapped to HTTP status by middleware.
tags:
    - go
    - transport
    - rest
dependencies:
    - transport
    - authorization
---

# REST adapter

> Expose a use case over REST with typed `Input`/`Output` DTOs that drive OpenAPI generation; a
> handler that parses -> calls the use case -> maps -> responds; routes registered through a router
> contract with a permission guard middleware; domain errors mapped to HTTP status by the
> middleware.

Reference tools: `Fuego` (OpenAPI-first web framework) via a thin web adapter.

{{define "when"}}
- Exposing a use case over REST.
- Building request/response DTOs for OpenAPI.
- Registering routes with a per-route permission guard.
{{end}}

{{define "recipe"}}
The handler holds a service constructor field, not the use case's dependencies. It resolves the
request-scoped service from the context (so the use case carries the caller's principal) - this
keeps the handler decoupled from how the service is built. Tests wrap a mock in the same
constructor. The type lives in `internal/transport/rest/thinghandler/thinghandler.go`:

```go
package thinghandler

type Handlers struct {
    newSvc func(ctx context.Context) (thing.Service, error)
    perms  thing.Perms
}

func New(newSvc func(context.Context) (thing.Service, error), perms thing.Perms) Handlers {
    return Handlers{newSvc: newSvc, perms: perms}
}

func (handlers Handlers) Create(req web.Request[CreateRequest]) web.Response[CreateResponse] {
    svc, err := handlers.newSvc(req.Context())
    if err != nil {
        return web.Response[CreateResponse]{Error: err}
    }
    created, err := svc.Create(req.Context(), thing.CreateInput{
        Name:  req.Body().Name,
        Owner: req.Body().Owner,
    })
    if err != nil {
        return web.Response[CreateResponse]{Error: err}
    }
    return web.Response[CreateResponse]{StatusCode: http.StatusCreated, Body: toDTO(created)}
}
```

Router contract (typed routes preserve OpenAPI generics):

```go
var _ web.RouterContract = Handlers{}

func (handlers Handlers) GroupName() string { return "things" }
func (handlers Handlers) Middlewares() []any {
    return []any{access.Require(handlers.perms.View)}
}

func (handlers Handlers) RegisterRoutes(reg web.RouteRegistrar) {
    web.POST[CreateRequest, CreateResponse](reg, "/things", handlers.Create)
    web.GET[ListRequest, ListResponse](reg, "/things", handlers.List)
}
```

DI - the handler gets a service constructor that resolves the request-scoped service from context:

```go
remy.RegisterConstructorArgs1(inj, remy.Singleton[thinghandler.Handlers],
    func(perms thing.Perms) thinghandler.Handlers {
        return thinghandler.New(func(ctx context.Context) (thing.Service, error) {
            return remy.GetWithContext[thing.Service](inj, ctx) // factory reads the principal
        }, perms)
    })
// routers built locally by the app's Routers() builder and mounted at the composition root.
```

A test wraps its mock in the same constructor shape:

```go
handlers := thinghandler.New(func(context.Context) (thing.Service, error) { return mockSvc, nil },
    thing.NewPerms(thing.Schema()))
```

Authorization is enforced by the access gate at the call site (the service layer), not in the handler;
the route guard is only a coarse fail-fast pre-check.
{{end}}

{{define "forbidden"}}
- Leaking proto/transport types into the domain; business logic in the handler.
- Permission decisions made only in the handler (the access gate at the call site is the authority).
- Hand-writing HTTP statuses for domain errors (return the domain-error type).
- Global routers/middleware; untyped route registration that loses OpenAPI generics.
{{end}}

{{define "validation"}}
- [ ] Handler package is the only importer of the REST framework adapter.
- [ ] `Input`/`Output` are transport DTOs with validate tags; entities mapped in `mappers.go`.
- [ ] Handler only parses, calls the use case, maps, and responds.
- [ ] Errors returned as the domain-error type; no status codes hand-written.
- [ ] Routes registered typed; `RouterContract` asserted; permission guard applied.
{{end}}
