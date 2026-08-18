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
- Wiring signal handling and graceful shutdown for a long-running server.
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

Graceful shutdown: derive `ctx` from `signal.NotifyContext` with `SIGINT`+`SIGTERM` only - the two
signals a process can actually intercept; `SIGKILL`/`SIGSEGV` are not interceptable, so listing them
is dead code. Drain the server (HTTP `Shutdown`, gRPC `GracefulStop` raced against a timeout), then
close long-lived singletons in REVERSE construction order. Register each singleton's close via
`defer` immediately after constructing it: LIFO then drains the server first (registered last, so it
runs first) and closes earlier-constructed singletons last. `shutdownTimeout` is a hardcoded named
constant, not a config knob.

```go
const shutdownTimeout = 30 * time.Second // not a config knob

func main() {
    // SIGINT/SIGTERM only: the two signals a process can intercept and act on.
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    var shutdownErr error // accumulated by errors.Join; logged once, at the very end

    conf, err := configload.Load("conf.toml", defaults(), envUpdater())
    if err != nil {
        panic(fmt.Errorf("load config: %w", err))
    }

    db, err := bootstrap.OpenDB(conf)
    if err != nil {
        panic(fmt.Errorf("open db: %w", err))
    }
    defer func() {
        if cerr := db.Close(); cerr != nil {
            shutdownErr = errors.Join(shutdownErr, fmt.Errorf("close db: %w", cerr))
        }
    }()

    inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
    remy.RegisterInstance(inj, conf)
    remy.RegisterInstance(inj, db)
    bootstrap.DoInjections(inj, conf)

    grpcSrv := bootstrap.NewGRPCServer(inj)
    defer gracefulStopGRPC(grpcSrv, shutdownTimeout)

    srv := bootstrap.NewWebServer(conf, inj)
    defer func() {
        shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
        defer cancel()
        if serr := srv.Shutdown(shutdownCtx); serr != nil {
            shutdownErr = errors.Join(shutdownErr, fmt.Errorf("shut down server: %w", serr))
        }
    }()

    if runErr := srv.Run(ctx); runErr != nil && !errors.Is(runErr, http.ErrServerClosed) {
        shutdownErr = errors.Join(shutdownErr, fmt.Errorf("run server: %w", runErr))
    }

    if shutdownErr != nil {
        // main() is the top of the chain: logging (not returning) is correct here (see [[logging]]).
        slog.Error("shutdown", slog.String("error", shutdownErr.Error()))
    }
}

// gracefulStopGRPC races GracefulStop against timeout and falls back to a hard Stop, because
// GracefulStop has no timeout of its own and can hang forever on a stuck stream.
func gracefulStopGRPC(srv *grpc.Server, timeout time.Duration) {
    done := make(chan struct{})
    go func() {
        srv.GracefulStop()
        close(done)
    }()

    select {
    case <-done:
    case <-time.After(timeout):
        srv.Stop()
    }
}
```

`os.Exit(1)`/`log.Fatalf` mid-cleanup SKIPS every remaining deferred close - that is why each step is
wrapped (`fmt.Errorf("<action verb>: %w", err)`) and folded with `errors.Join` instead of exiting or
logging as each step fails: one accumulated error, one `slog.Error` call, and every close still runs.

Background work that must keep draining after the server stops accepting - e.g. an in-flight
scheduler/relay - runs on a context derived with `context.WithoutCancel(ctx)` (see [[concurrency]]),
cancelled only once the server has finished shutting down, and bounded by the same
`shutdownTimeout` so a wedged worker cannot hang the process forever.

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
`Routers(inj, ...)` builder) - that is the root owning wiring, not a collaborator reaching in.
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
- `signal.Notify`/`signal.NotifyContext` on any signal beyond `SIGINT`/`SIGTERM` (e.g. `SIGKILL`,
  which cannot be intercepted - dead code).
- `srv.GracefulStop()` with no timeout fallback.
- A long-lived singleton (DB pool, cache client) left unclosed on shutdown.
- `os.Exit(1)`/`log.Fatalf` mid-cleanup instead of accumulating with `errors.Join` and logging once.
- Logging a shutdown step's error immediately instead of folding it and logging once at the end.
{{end}}

{{define "validation"}}
- [ ] `main()` is thin and follows config -> DB -> migrations -> injector -> server.
- [ ] One injector; infra/domain/interop registered in order via `DoInjections`.
- [ ] Singletons vs factories chosen correctly; constructors take CONCRETE deps (never the injector).
- [ ] One composition root (`cmd/<app>`); apps expose `DoInjections`/`Routers` from their root package.
- [ ] Routers built locally via `app.Routers()`, not registered in the main injector.
- [ ] Per-request use cases resolved with `remy.GetWithContext[T](inj, ctx)`.
- [ ] Config-driven impl selection happens only in bootstrap.
- [ ] `ctx` derives from `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)`
  with `defer stop()`.
- [ ] `shutdownTimeout` is a hardcoded named constant (`30 * time.Second`), not config-driven.
- [ ] The server drains before long-lived singletons close, singletons close in reverse construction
  order, and each close is registered via `defer` right after construction.
- [ ] gRPC shutdown races `GracefulStop()` against `shutdownTimeout` and falls back to `Stop()`.
- [ ] Each shutdown step is wrapped (`fmt.Errorf("<verb>: %w", err)`), folded with `errors.Join`, and
  logged exactly once via `slog.Error` in `main()`.
{{end}}
