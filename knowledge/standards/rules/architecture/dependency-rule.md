---
id: dependency-rule
name: The Dependency Rule - inward-only layer dependencies
description: Source-code dependencies point only inward; the domain (entities + use cases) imports no infra, transport, persistence driver, ORM runtime, or framework package, so the business core stays independent of every detail. The vertical layer-direction counterpart to module-boundaries' horizontal facade rule. Language-neutral Clean Architecture; see dependency-rule-go for the Go-concrete instantiation.
tags:
    - architecture
    - layout
analyzers:
    - arch-boundary
---

# The Dependency Rule - inward-only layer dependencies

> Source-code dependencies cross a layer boundary in one direction only: inward, toward the business
> policy. The domain layer (entities + use cases) imports no infrastructure, transport, persistence
> driver, ORM runtime, or framework package; outer layers (transport, infra, the composition root)
> depend on the domain, never the reverse. The business core is independent of every detail that
> surrounds it.

Applies to: deciding which module/package/layer an import may live in, reviewing a cross-layer
import, and keeping the domain testable in isolation, in any language or framework. The vertical
layer-direction counterpart to [[module-boundaries]]'s horizontal facade rule; it generalizes
[[transport]] (which forbids only `net/http`/grpc/proto in the Go domain) to every detail. See
[[interop]], [[code-placement]], [[frameworks-as-plugins]], and `dependency-rule-go` for the Go
package-level denylist and its linter.

{{define "when"}}
- Adding an import/dependency to a domain (entities/use-cases) module.
- Deciding where a type that touches a framework, driver, or transport belongs.
- Reviewing a diff for an inward-pointing dependency violation.
- Wiring the concrete detail at the composition root.
{{end}}

{{define "must"}}
1. Source dependencies point only inward: entities (innermost) depend on nothing outer; use cases
   depend on entities and on consumer-declared ports, never on a concrete adapter; transport/infra
   (outer) depend inward on the domain. An inner layer NEVER names anything in an outer layer.
2. The domain module imports no persistence driver, ORM/query runtime, HTTP/RPC client or server,
   proto/schema package, web framework, or other infrastructure/transport dependency - only the
   platform standard library, the domain itself, and its own ports.
3. Cross a boundary inward through an interface the inner layer declares (a consumer-declared port);
   the outer layer implements it. Control flows outward while the source dependency points inward
   (Dependency Inversion at the boundary).
4. Data crossing a boundary is a plain domain type the inner layer owns - never a generated row
   struct, proto message, ORM entity, or raw transport request/response shaped by an outer layer.
5. Pick the concrete detail (driver, framework, transport) only at the composition root; the domain
   stays unaware of which one was chosen (see [[frameworks-as-plugins]]).
{{end}}

{{define "forbidden"}}
- A domain (entities/use-cases) module importing infra, transport, a DB driver, an ORM runtime, a
  proto/schema package, or a web framework.
- A use case depending on a concrete adapter instead of a consumer-declared port.
- An outer-shaped type (row/proto/ORM entity/raw transport request) crossing inward into the domain.
- An import cycle that an inward-only direction would have prevented.
{{end}}

{{define "validation"}}
- [ ] No infra/transport/driver/ORM/proto/framework import inside the domain module.
- [ ] Cross-layer calls go inward through an inner-declared port; the outer layer implements it.
- [ ] Only domain-owned types cross a boundary; no row/proto/ORM/raw-transport type leaks inward.
- [ ] The concrete detail is chosen at the composition root, not in the domain.
- [ ] The domain's import denylist is machine-checked (a linter/architecture test/verifier).
{{end}}
