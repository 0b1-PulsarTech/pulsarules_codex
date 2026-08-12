---
id: module-boundaries
name: Module boundaries - cohesion, coupling, connascence
description: Keep cohesion high inside a module and connascence weak across its facade - cross-module contracts carry only Connascence of Name and Type (named interfaces + typed DTOs), with stronger and dynamic connascence kept behind the boundary; derive modules from workflows, not entities.
tags:
    - go
    - architecture
    - modularity
linters:
    - depguard
analyzers:
    - import-cycle
---

# Module boundaries - cohesion, coupling, connascence

> Maximize cohesion WITHIN a module; minimize connascence ACROSS its facade. A cross-module contract
> carries only the weakest, most static connascence - Connascence of Name and Type (a stable named
> interface plus typed DTOs). Everything stronger (Position, Meaning, Algorithm) and every form of
> dynamic connascence (execution order, timing, shared value, shared identity) stays behind the
> boundary.

Applies to: deciding where a boundary goes, designing or reviewing a cross-module facade, naming and
grouping packages, and judging whether to split or merge a module. Vocabulary from Page-Jones's
connascence and Constantine's cohesion/coupling; see [[code-placement]], [[interop]], [[design-patterns]].

{{define "when"}}
- Designing or reviewing a cross-module call or facade port.
- Deciding whether to split a package/module out or keep it whole.
- Naming a package or grouping code by cohesion.
- Reviewing coupling between modules for boundary violations.
{{end}}

{{define "must"}}
1. Aim for FUNCTIONAL cohesion: every file in a package shares one purpose, named for what it does
   (a workflow/capability), never a `util`/`common`/`helpers` grab-bag or a `Manager` of unrelated
   methods (logical/coincidental cohesion). Do not split a cohesive package - splitting only trades
   cohesion for cross-module coupling.
2. Route every cross-module call through a consumer-declared FACADE port; the provider is unaware of
   the consumer. No deep import across a module boundary that a port should mediate (see [[interop]]).
3. A cross-facade contract carries only WEAK, STATIC connascence: Connascence of Name + Type - a small
   named interface plus typed DTO structs. Keep Position (use a named parameter struct, not positional
   args), Meaning (named constants, not magic values), Algorithm (one owning package both sides call),
   and ALL dynamic connascence (required call order, timing, shared mutable value, shared identity)
   BEHIND the facade.
4. Refactor strong connascence into weaker forms as it appears (Rule of Degree): magic value -> named
   constant; positional args -> a named parameter struct; a duplicated algorithm -> a single shared
   function. The farther apart two pieces are, the weaker their coupling must be (Rule of Locality) -
   tight coupling is fine within one package, not across a module boundary.
5. A facade collapses a consumer's outgoing fan-out to ONE stable port (low instability `Ce/(Ce+Ca)`):
   depend on the port, never on the provider's internals; keep dependency direction one-way (a provider
   never imports its consumer).
6. Derive modules from WORKFLOWS/actions, not database entities - avoid the entity trap of one
   `Manager` per table. Prefer domain/workflow partitioning over technical-layer partitioning so a
   change stays inside one module instead of smearing across layers.
7. Split a module out only when a real boundary justifies its own `go.mod` (separate deploy, separate
   ownership, or a genuine reuse seam); premature modularization just adds cross-facade chatter.
8. Keep packages on the main sequence: instability DECREASES in the dependency direction (Stable
   Dependencies - a more-stable, lower-`Ce/(Ce+Ca)` package never imports a less-stable one), and
   abstractness tracks stability (Stable Abstractions - the stable, high-fan-in domain package exposes
   interfaces/typed contracts, while volatile concretes sit at the leaves). A stable-but-concrete
   package (zone of pain) or an abstract package nothing depends on (zone of uselessness) is a smell to
   refactor.
{{end}}

{{define "forbidden"}}
- A deep import across a module boundary instead of a consumer-declared facade port.
- `util`/`common`/`helpers` grab-bag packages; a `Manager`-per-entity component set (the entity trap).
- Leaking Position, Meaning, Algorithm, or any dynamic connascence across a facade (callers depending
  on argument order, magic numbers, a duplicated algorithm, or a required call sequence).
- Splitting a cohesive package merely to "modularize" - it trades cohesion for coupling.
- A provider module importing its consumer (two-way coupling / an import cycle across the boundary).
{{end}}

{{define "validation"}}
- [ ] Packages are functionally cohesive; no `util`/`common`/`helpers` grab-bags.
- [ ] Cross-module calls go through a consumer-declared facade port; no deep cross-boundary imports.
- [ ] Facade contracts use named interfaces + typed DTOs (CoN/CoT); no positional/magic-value/algorithm
  or dynamic connascence crosses the boundary.
- [ ] One-way dependency direction; no provider-imports-consumer cycle; fan-out collapsed to one port.
- [ ] Modules derived from workflows, not entities; a split is justified by a real boundary.
- [ ] Instability decreases in the dependency direction; the stable domain package is abstract; no
  zone-of-pain (stable+concrete) or zone-of-uselessness (abstract+unused) package.
{{end}}
