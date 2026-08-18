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
4. The bootstrap (`internal/bootstrap/injections.go` or `register_injections.go`) is the ONLY place
   that switches on a config-driven selector (dialect, driver, provider code). No `switch driver`
   inside a repo `di.go`; no `switch code` inside a provider `di.go`.
5. Lifetime: `Singleton[T]` for stateless/long-lived (DB, HTTP client, mailer, registry);
   `Factory[T]` for per-request (the principal, tx-carrying repos, request logger). Use a `Module`
   when a feature has more than 3 registrations.
6. Constructors take every dependency as a parameter; never read globals mid-method, never store
   the injector for later `Get`.
7. To inject something at resolve time, use typed pairs (`GetWithPairs` with `BindInstance`) over a
   `GetWith(callback)` that hides what is being added.
8. The injector is built once in `cmd/<binary>` (`serve.go`/`main.go`), populated by bootstrap, and
   consulted only to resolve the top-level service.
{{end}}

{{define "forbidden"}}
- `RegisterSingleton(func(ret DependencyRetriever)...)` - an opaque factory that defeats the DI graph.
- `Get[*concreteType](ret)` inside a registered factory; defeats duck-typing and hides the dep.
- Service-locator usage outside bootstrap; application code querying the injector instead of
  receiving deps through constructor parameters.
- Global `var inj = NewInjector(...)`; a `switch driver`/`switch code` outside bootstrap.
- Storing the injector on a struct for later `Get`.
{{end}}

{{define "validation"}}
- [ ] Only `RegisterConstructor*` used; `RegisterSingleton(func)` absent.
- [ ] Use-case packages declare consumer-defined interfaces; `di.go` is one-liner-per-type.
- [ ] Repo `di.go` registers concrete + interface bindings.
- [ ] Impl/dialect/provider selection happens only in bootstrap.
- [ ] Singletons vs factories chosen correctly; constructors take all deps.
- [ ] No injector stored on a struct; no `Get` outside bootstrap/constructor/handler-first-lines.
{{end}}
