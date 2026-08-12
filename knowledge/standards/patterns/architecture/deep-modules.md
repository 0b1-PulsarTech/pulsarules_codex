---
id: deep-modules
name: Deep modules
description: Design for depth - a lot of behavior behind a small interface at a real seam, testable through that interface. The deletion test, interface-as-test-surface, and one-adapter-is-hypothetical, mapped onto the house constructs (UseCase behind a Repository port, Facade marker, driver-named adapters, Engine/Pipeline).
tags:
    - architecture
    - go
dependencies:
    - module-boundaries
---

# Deep modules

> A deep module hides a lot of behavior behind a small interface at a clean seam; a shallow module's
> interface is nearly as complex as its body. Aim for depth: leverage for callers, locality for
> maintainers, testability through the seam. This is the vocabulary the `codebase-design` skill uses.

{{define "when"}}
- Designing or reshaping a module's interface (a UseCase, an Engine, a Pipeline, a repository).
- Deciding where a seam goes, or whether a seam is worth introducing at all.
- Judging whether an abstraction earns its keep before adding it.
{{end}}

{{define "must"}}
1. Prefer a small interface over a large one: fewer methods, simpler parameters, more hidden inside.
   In Go that is a consumer-declared port named for its ROLE (`Repository`, `Sender`, `Resolver`),
   never a generic `Port`, and named input structs over long positional parameter lists.
2. Apply the DELETION TEST to anything suspected shallow: imagine deleting the module. If complexity
   vanishes it was a pass-through - inline it. If complexity reappears across N callers, it earns its
   keep - keep and deepen it.
3. Treat the interface as the TEST SURFACE: callers and tests cross the same seam. Needing to test
   past the interface means the module is the wrong shape.
4. Introduce a seam only when something actually varies across it: one adapter is a hypothetical seam,
   two adapters is a real one. Do not add an interface for a single implementation.
5. Accept dependencies, do not construct them inside; return results rather than hiding side effects,
   so the deep module stays verifiable through its surface.
{{end}}

{{define "examples"}}
Deep modules in the house vocabulary:

- A `UseCase` is deep behind a small consumer-declared `Repository` port: one action per file, the
  business invariants inside, the port exposing only what the use case needs.
- A `Facade` (the cross-module boundary, a marker interface with a private method so it is only
  implementable through its blessed constructor) is the seam between feature modules - depth is the
  whole feature behind a couple of methods.
- A driver-named repository (`mysqlrepo`, `repomongo`) is an ADAPTER at the persistence seam; a second
  driver is what makes that seam real rather than hypothetical.
- An `Engine`/`EngineFactory` or a `Pipeline`/`Step` hides a multi-stage algorithm behind a small
  drive method; the stages are internal seams private to the implementation, not part of the surface.
- A type-keyed constructor registry (`BindKey[T]`, `ProviderKey[V]`, a `map[Kind]constructor`) is a
  deep selection module: no `switch`, no reflection at the call site, one small lookup interface.
{{end}}

{{define "forbidden"}}
- Drifting into "component", "service", "API", or "boundary" instead of the glossary terms.
- A shallow pass-through module whose interface is nearly as complex as its body - inline it.
- An interface or seam added for a single, hypothetical implementation.
- Measuring depth as implementation-lines over interface-lines (it rewards padding the body).
- Exposing an internal collaborator only so a test can reach past the interface.
{{end}}

{{define "validation"}}
- [ ] The module was described as module/interface/seam/depth before coding.
- [ ] The interface is as small as the behavior allows; the deletion test was applied to suspects.
- [ ] Every seam has two real adapters, or it was not introduced.
- [ ] Dependencies are accepted not constructed; the interface is the test surface.
{{end}}
