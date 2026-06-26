---
id: frameworks-as-plugins
name: Frameworks and the database are plugins
description: Frameworks, the database engine, and the web are details kept behind ports; their imports live only in cmd, adapter, and infra packages, the concrete is chosen at the composition root, and no framework base-type or annotation leaks into the domain.
tags:
    - go
    - architecture
    - modularity
linters:
    - depguard
---

# Frameworks and the database are plugins

> Treat every framework, database engine, and the web itself as a replaceable detail behind a port,
> not as the thing your domain is built on. Framework and driver imports are confined to `cmd/`,
> adapter, and `internal/infra/**` packages; the concrete choice is made only at the composition
> root; the domain embeds no framework base-type and carries no framework annotation.

Applies to: introducing or wiring a web framework, ORM, DB driver, message broker, or other
third-party SDK. The boundary this protects is enforced by [[dependency-rule]] (inward-only) and the
Adapter discipline in [[design-patterns]]; the wiring mechanism is [[dependency-injection]]. See
[[database]], [[observability]] (the tracer is already chosen at `cmd`/infra).

{{define "when"}}
- Adding a web framework, ORM/ent runtime, DB driver, broker client, or vendor SDK.
- Deciding which package may import that dependency.
- Wiring the concrete implementation at bootstrap.
- Reviewing whether a framework has leaked into the domain.
{{end}}

{{define "must"}}
1. A framework/DB/driver is a plugin behind a consumer-declared port: the domain depends on the port,
   the adapter (in `internal/infra/**`) imports the framework and implements the port.
2. Framework, ORM runtime, and driver imports appear ONLY in `cmd/**`, adapter packages, and
   `internal/infra/**`. Entities and use cases import none of them (see [[dependency-rule]]).
3. The concrete framework/DB/driver is selected at the ONE composition root (`cmd/<app>`); the rest of
   the code names the port, never the concrete, so the plugin can be swapped without touching policy.
4. No domain type embeds a framework base-type (no `gorm.Model`, no framework `BaseController`/context
   embedding) and no domain struct carries framework-driven annotations/struct tags that bind it to a
   library; persistence/transport tags live on the adapter's DTOs, not on entities.
5. Defer the framework decision: keep the core runnable and testable with no framework on the import
   path, so the choice can be made (or changed) late and proven by a plugin-swap test.
{{end}}

{{define "forbidden"}}
- Importing a web framework, ORM runtime, DB driver, or vendor SDK inside `internal/domain/`.
- A domain entity embedding a framework base-type or carrying framework annotations/struct tags.
- Selecting the concrete framework/DB anywhere but the composition root (a `switch driver` outside
  bootstrap; see [[dependency-injection]]).
- A use case that cannot run without the framework being present.
{{end}}

{{define "validation"}}
- [ ] Framework/ORM/driver imports confined to `cmd/**`, adapters, and `internal/infra/**`.
- [ ] Domain depends on a port; the adapter imports the framework and implements it.
- [ ] Concrete framework/DB chosen only at the composition root.
- [ ] No framework base-type embedded in, and no framework annotation on, a domain type.
- [ ] The core builds and tests with no framework on its import path.
{{end}}
