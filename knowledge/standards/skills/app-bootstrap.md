---
id: app-bootstrap
name: App bootstrap
---

## Mandatory workflow

1. Keep `main()` thin (~15-20 lines) and follow the sequence: load config -> open DB -> run migrations (gated) -> build
   the injector -> wire boot -> run the server. `panic` only here, on impossible boot states.
2. Keep import-time purity: packages only define symbols. No I/O, connections, goroutines, timers, env reads, or
   registrations in `init()`. Allowed exceptions: driver `_` imports in `main.go` with a comment, and compile-time
   `var _ Iface = (*impl)(nil)` assertions.
3. Define a typed `Config` (in `hookconf/` or `internal/bootstrap/config.go`) loaded once from TOML + env via the config
   loader; treat it as read-only after boot. Bind env via package-private name constants; never `os.Getenv` outside
   config. Zero secret source fields after registering them.
4. Wire per-layer `Register*` functions: register infra (repositories, adapters) as singletons, then domain (use cases)
   as factories that carry the per-request principal, then interop (facades) as singletons. The bootstrap (
   `DoInjections`) is the only switchboard.
5. Resolve the per-request principal from JWT claims via a factory registered in the injector (
   `Factory[entities.Principal]` from context). Use cases declare the principal as a constructor argument; the domain
   never imports JWT types. For a per-request use case, resolve from the request context with
   `remy.GetWithContext[T](inj, ctx)` - the factory reads the caller from the ctx-injected principal.
   5a. ONE composition root (`cmd/<app>`): it calls infra `RegisterAndInit` once + each module's `DoInjections`; each
   app
   exposes `DoInjections` AND a `Routers` builder from its ROOT package (imported by app name, e.g.
   `apps/wabapi/msgreceiver`), never a `bootstrap` subpackage, and ships no `main`/DB/server of its own.
   5b. Transport wiring is NOT global state: routers are NOT registered in the main injector. Each app exposes
   `Routers(inj, conf, mws...) ([]webwrap.RouterContract, error)` that builds them locally; `NewHTTPServer` collects
   `app.Routers()` and derives the mux from each router's `GroupName()`. The main injector holds only the
   domain/infra collaborators resolved in many places.
   5c. Collaborator constructors (use cases, services, repos) take CONCRETE deps; never `remy.Injector`/
   `DependencyRetriever`. Resolve at the root. The ONE exception is the composition seam itself (`DoInjections`, an app
   `Routers(inj, …)` builder) - the root owning wiring, not a collaborator reaching in.
6. Choose lifetime: `Singleton[T]` for stateless/long-lived (DB, HTTP client, mailer, registry); `Factory[T]` for
   per-request (the principal, tx-carrying repos, request logger). Use a `Module` when a feature has more than 3
   registrations.
7. Switch on a config-driven selector (dialect, driver, provider code) ONLY in bootstrap. No `switch driver`/
   `switch code` inside a repo or provider `di.go`.
8. Embed SQL migrations via `//go:embed` and apply them with an in-house, dialect-abstracted runner, gated by a config
   flag. Atlas generates the `.sql` at build time only; the runtime applier is the in-house migrator.
9. Set `slog.SetDefault` in `main()` before any other call. Build the injector once in `cmd/<binary>`; consult it only
   to resolve the top-level service.

## Validation checklist

- [ ] `main()` is thin and follows config -> DB -> injector -> migrations -> server.
- [ ] No side effects at import/`init()`; only driver registration / interface assertions.
- [ ] One injector; infra/domain/interop registered in order via `DoInjections`.
- [ ] `Config` typed, loaded once via the loader; env names constants; secrets zeroed; no `os.Getenv` outside config.
- [ ] Principal registered as `Factory[entities.Principal]` from context; use cases take it as a constructor arg.
- [ ] One composition root (`cmd/<app>`); apps expose `DoInjections`/`Routers` from their root package, ship no main.
- [ ] Routers built locally via `app.Routers()` (not in the main injector); per-request use cases via `GetWithContext`.
- [ ] Singletons vs factories chosen correctly; constructors take CONCRETE deps (never the injector).
- [ ] Config-driven impl selection happens only in bootstrap.
- [ ] Migrations embedded via `//go:embed`; applied by an in-house runner, gated by config.
- [ ] No package-level mutable state; `slog.SetDefault` set in `main()` first.

## Forbidden actions

- Side effects in `init()` or at package load; `panic` outside `main()`; package-level mutable state / globals.
- Storing the injector on a struct for later `Get` (service-locator anti-pattern); `remy.Get`/`remy.GetWithContext`
  outside bootstrap, an app-root seam (`DoInjections`/`Routers`), a constructor, or a handler's first lines.
- A collaborator constructor taking `remy.Injector`/`DependencyRetriever` instead of concrete deps.
- An app shipping its own `main`/DB/server, or exposing `DoInjections` from a `bootstrap` subpackage.
- Registering a router (transport-only object) in the main injector.
- `os.Getenv` outside the config package; mutating `Config` after boot; a `switch driver`/`switch code` outside
  bootstrap.
- Importing JWT types into the domain; re-parsing a JWT inside a use case.
- Running Atlas at runtime; applying migrations unconditionally at startup; editing applied migrations in place.
- `slog.SetDefault` in a library.

## Expected outputs

- A thin `main()` that wires config -> DB -> injector -> gated migrations -> server.
- One injector with infra/domain/interop registered in order; config-driven selection only in bootstrap.
- The per-request principal resolved from context as a factory; no JWT in the domain.
