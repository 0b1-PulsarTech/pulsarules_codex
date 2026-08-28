---
id: dependency-injection
name: Dependency injection
description: Constructor-based DI via an injector (remy); RegisterConstructor* only; consumer-declared interfaces; bootstrap is the only switchboard; no service-locator usage.
references:
    - goremy-di
tags:
    - go
    - di
linters:
    - depguard
---

# Dependency injection

> Constructor-based DI via an injector (source repos use `remy`/goremy-di with
> `Config{DuckTypeElements: true}`): `RegisterConstructor*` only, consumer-declared interfaces,
> the bootstrap is the only switchboard, no service-locator usage outside bootstrap.

Applies to: wiring services, repositories, facades, and routers. Canonical reference:
`github.com/wrapped-owls/goremy-di/remy`.

{{define "when"}}
- Registering a service, repository, facade, or router in the injector.
- Deciding singleton vs per-request factory lifetime.
- Wiring a use-case or repository `di.go`.
- Choosing how to inject something at resolve time.
{{end}}

{{define "must"}}
1. Use `RegisterConstructor*` only. If a constructor has N dependencies, use
   `RegisterConstructorArgsN[Err]` so the injector resolves them from the graph.
2. Each use-case package declares the smallest interface it needs (consumer-defined interfaces);
   the use-case `di.go` is one-liner-per-type.
3. A repository `di.go` registers both the concrete binding and the interface binding the use case
   consumes; the interface binding is what use cases actually consume.
4. The bootstrap (`internal/bootstrap/injections.go` or `register_injections.go`) is the only place
   that switches on a config-driven selector (dialect, driver, provider code). No `switch driver`
   inside a repo `di.go`; no `switch code` inside a provider `di.go`.
5. Lifetime: `Singleton[T]` for stateless/long-lived (DB, HTTP client, mailer, registry);
   `Factory[T]` for per-request (the principal, tx-carrying repos, request logger). Use a `Module`
   when a feature has more than 3 registrations.
6. Constructors take every dependency as a parameter; never read globals mid-method, never store
   the injector for later `Get`.
7. To inject something at resolve time, use typed pairs - `GetWithPairs` with
   `remy.NewBindEntry(value)` (or `NewBindEntryTagged(value, tag)` when the bind is tagged) - over
   a `GetWith(callback)` that hides what is being added.
8. The injector is built once in `cmd/<binary>` (`serve.go`/`main.go`), populated by bootstrap, and
   consulted only to resolve the top-level service.
9. Keep `Config{DuckTypeElements: true}` on: it is what resolves a consumer-declared port to the
   concrete that satisfies it structurally, which [[interop]] depends on for every facade. Its scan
   costs nothing on a concrete `Get` or on an interface `Get` that hits a direct bind - only an
   interface resolve that misses one pays it. Add `Config{MultiBinding: true}` alongside it when
   code must also list every bind under a tag, read through the `GetAll` family. `MultiBinding`
   answers "list them"; it is not a cheaper substitute for structural discovery.
10. Layer a cross-cutting concern (retry, tracing, audit) onto a bind whose registration you do not
    own with `remy.Decorate[T](bind, decorator)`; it keeps the wrapped bind's kind and lifecycle, so
    a `Factory` is decorated on every build and a `Singleton` exactly once. Where you do own the
    registration, plain constructor composition - the wrapper struct taking the inner port as a
    constructor parameter - stays the default (see [[retry-backoff]]); reach for `Decorate` when
    rewriting the original registration is not yours to do.
{{end}}

{{define "forbidden"}}
- `RegisterSingleton(func(ret DependencyRetriever)...)` - an opaque factory that defeats the DI graph.
- `Get[*concreteType](ret)` inside a registered factory; defeats duck-typing and hides the dep.
- Service-locator usage outside bootstrap; application code querying the injector instead of
  receiving deps through constructor parameters.
- Global `var inj = NewInjector(...)`; a `switch driver`/`switch code` outside bootstrap.
- Storing the injector on a struct for later `Get`.
- Reaching for `MultiBinding`/`GetAll` to satisfy an interface-discovery `Get`, which is what
  `DuckTypeElements` resolves.
- Re-registering a bind this package does not own to add a cross-cutting concern, instead of
  layering it with `remy.Decorate`.
{{end}}

{{define "validation"}}
- [ ] Only `RegisterConstructor*` used; `RegisterSingleton(func)` absent.
- [ ] Use-case packages declare consumer-defined interfaces; `di.go` is one-liner-per-type.
- [ ] Repo `di.go` registers concrete + interface bindings.
- [ ] Impl/dialect/provider selection happens only in bootstrap.
- [ ] Singletons vs factories chosen correctly; constructors take all deps.
- [ ] No injector stored on a struct; no `Get` outside bootstrap/constructor/handler-first-lines.
- [ ] `DuckTypeElements` stays on wherever a port is resolved structurally; `MultiBinding` is added
  for listing, never substituted for discovery.
- [ ] A bind whose registration this code does not own is layered via `remy.Decorate`, not a
  duplicate registration.
{{end}}
