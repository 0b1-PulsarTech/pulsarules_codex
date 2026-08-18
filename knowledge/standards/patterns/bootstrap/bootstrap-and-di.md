---
id: bootstrap-and-di
name: Bootstrap & DI
description: The composition root wiring main to config to DB to injector to server; per-layer Register* functions; bootstrap is the only switchboard.
tags:
    - go
    - bootstrap
    - di
dependencies:
    - dependency-injection
    - startup
    - config
composes:
    - app-skeleton
    - user-from-context
    - config-layout
    - embedded-migrations
---

# Bootstrap & DI

> The composition root: a thin `main()` wires config -> DB -> migrations -> injector -> server;
> per-layer `Register*` functions register infra (singletons) then domain (factories) then interop
> (facades); the bootstrap is the only switchboard for config-driven impl selection.

Reference tools: `remy` (goremy-di) with `Config{DuckTypeElements: true}`; a config loader
(`confloader`).

{{define "when"}}
- Wiring `main()` to config to DB to injector to server.
- Registering services/repositories/facades in the injector.
- Choosing singleton vs per-request factory lifetime.
- Switching dialect/driver/provider by config.
{{end}}

{{define "recipe"}}
Thin `main()` (~15-20 lines):

```go
func main() {
    slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

    conf, err := configload.Load("conf.toml", defaults(), envUpdater())
    if err != nil {
        panic(fmt.Errorf("load config: %w", err))
    }

    db, err := bootstrap.OpenDB(conf)
    if err != nil {
        panic(fmt.Errorf("open db: %w", err))
    }

    if conf.RunMigrations {
        if err := bootstrap.ApplyMigrations(db); err != nil {
            panic(fmt.Errorf("apply migrations: %w", err))
        }
    }

    inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
    remy.RegisterInstance(inj, conf)
    remy.RegisterInstance(inj, db)

    bootstrap.DoInjections(inj, conf)

    srv := bootstrap.NewWebServer(conf, inj)
    if err := srv.Run(); err != nil {
        panic(fmt.Errorf("run server: %w", err))
    }
}
```

Per-layer orchestration (`internal/bootstrap/register_injections.go`):

```go
func DoInjections(inj remy.Injector, conf hookconf.Config) {
    registerInfra(inj, conf)   // repositories, adapters -> singletons
    registerDomain(inj)        // use cases -> factories (carry the request principal)
    registerInterop(inj)       // facades -> singletons
}
```

Lifetime rules:

```go
// Stateless / long-lived -> singleton.
remy.RegisterInstance(inj, db)
remy.RegisterConstructorArgs1(inj, remy.Singleton[*httpdatasource.Client], httpdatasource.New)

// Per-request -> factory (carries the request-scoped principal / tx state).
remy.RegisterConstructorArgs2(inj, remy.Factory[UseCase], NewUseCase)
```

Config-driven impl selection - ONLY in bootstrap:

```go
switch conf.Database.Driver {
case "mysql":
    repomysql.Register(inj, db)
default:
    reposqlite.Register(inj, db)
}
```

One composition root, app-root seams. The monolith has exactly ONE composition root (`cmd/<app>`): it
calls infra `RegisterAndInit` once, then each module's `DoInjections`. Each app exposes its `DoInjections`
(and a `Routers` builder) from its ROOT package - imported by app name (e.g. `apps/wabapi/msgreceiver`),
NOT a `bootstrap` subpackage - and ships no `main`, DB, or server of its own.

```go
// cmd/<app>/main.go - the only composition root.
inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
infra.RegisterAndInit(inj, conf)        // once
msgreceiver.DoInjections(inj, conf)     // each module's app-root seam
contacts.DoInjections(inj, conf)

srv, err := webserver.NewHTTPServer(conf, inj,
    msgreceiver.Routers, contacts.Routers) // see below
```

Transport wiring is NOT global state. Objects consumed only by the composition root's HTTP server
(routers) are NOT registered in the main injector. Each app exposes a local builder; `NewHTTPServer`
collects every `app.Routers()` and derives the mux from each router's `GroupName()`. The main injector
holds only the domain/infra collaborators resolved in many places.

```go
// apps/wabapi/msgreceiver/routers.go
func Routers(inj remy.Injector, conf hookconf.Config, mws ...webwrap.Middleware) ([]webwrap.RouterContract, error) {
    h, err := remy.Get[Handlers](inj) // build locally; not stored on the injector
    if err != nil {
        return nil, fmt.Errorf("build msgreceiver routers: %w", err)
    }
    return []webwrap.RouterContract{h.WithMiddlewares(mws...)}, nil
}
```

Request-scoped resolution. For a per-request use case, resolve from the request context with
`remy.GetWithContext[T](inj, ctx)`; the use-case factory reads the caller from the ctx-injected principal.

```go
svc, err := remy.GetWithContext[contacts.Service](inj, ctx) // factory reads the principal from ctx
```

Collaborators take CONCRETE deps. A use case / service / repository constructor takes concrete
collaborators (`NewUseCase(repo Repository, clk Clock)`), NEVER `remy.Injector` / `DependencyRetriever`;
those are resolved at the root. The ONE exception is the composition seam itself (`DoInjections`, an app
`Routers(inj, …)` builder) - that is the root owning wiring, not a collaborator reaching in.
{{end}}

{{define "forbidden"}}
- Package-level mutable state / globals; side effects in `init()` or at package load.
- Storing the injector on a struct for later `Get` (service-locator anti-pattern).
- `remy.Get`/`remy.GetWithContext` outside bootstrap, an app-root seam (`DoInjections`/`Routers`), a
  constructor, or a handler's first lines.
- A collaborator constructor that takes `remy.Injector`/`DependencyRetriever` instead of concrete deps
  (only the composition seam may read the injector).
- An app shipping its own `main`/DB/server, or exposing `DoInjections` from a `bootstrap` subpackage
  instead of its root package.
- Registering a router (transport-only object) in the main injector.
- A `switch driver`/`switch code` outside bootstrap.
- `slog.SetDefault` in a library.
{{end}}

{{define "validation"}}
- [ ] `main()` is thin and follows config -> DB -> migrations -> injector -> server.
- [ ] One injector; infra/domain/interop registered in order via `DoInjections`.
- [ ] Singletons vs factories chosen correctly; constructors take CONCRETE deps (never the injector).
- [ ] One composition root (`cmd/<app>`); apps expose `DoInjections`/`Routers` from their root package.
- [ ] Routers built locally via `app.Routers()`, not registered in the main injector.
- [ ] Per-request use cases resolved with `remy.GetWithContext[T](inj, ctx)`.
- [ ] Config-driven impl selection happens only in bootstrap.
{{end}}
