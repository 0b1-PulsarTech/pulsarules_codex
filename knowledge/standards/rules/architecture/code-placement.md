---
id: code-placement
name: Code placement
description: One-way dependency direction between a deployable and the libraries it uses; non-API code kept unreachable from outside the boundary that owns it; a package/directory named for the business capability, never the technical layer; a deployable's entry point kept separate from its library code. Language-neutral; see code-placement-go, code-placement-monorepo, and code-placement-inner-modules for the concrete directory maps.
tags:
    - architecture
    - layout
analyzers:
    - arch-boundary
    - import-cycle
---

# Code placement

> Four invariants hold regardless of language or repo layout: a deployable may depend on a library,
> never the reverse, and no deployable depends on another; code that is not public API stays
> unreachable from outside the boundary that owns it; a package or directory is named for the
> business capability it serves, never the technical layer or framework; and a deployable's entry
> point stays separate from the library code it wires together.

Applies to: deciding where a file, module, or package belongs, and reviewing a diff for a placement
violation, in any language or repo layout. The horizontal counterpart to [[dependency-rule]]'s
vertical layer direction - this rule is about which top-level boundary a file lives in and which way
a dependency between boundaries may point; [[dependency-rule]] is about the domain's inward-only
layering inside one such boundary. See [[module-boundaries]] for the facade/connascence contract at
a crossed boundary. The concrete directory map is language- and layout-specific:
[[code-placement-go]] (single Go module), [[code-placement-monorepo]], and
[[code-placement-inner-modules]].

{{define "when"}}
- Creating a new module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for a boundary violation, in any language.
{{end}}

{{define "must"}}
1. Dependency direction between a deployable and a library is one-way: a deployable (an app, binary,
   or process entry point) may depend on a library; a library never depends on a deployable; no
   deployable depends on another deployable - share code by extracting a library instead.
2. Code that is not public API stays unreachable from outside the boundary that owns it, enforced by
   whatever guarantee the language/toolchain gives (Go's `internal/`, a Dart package's un-exported
   `lib/src/`, a module with no barrel export) - never by convention alone where the language offers
   a stronger guarantee.
3. A package or directory is named for the business capability it serves (`billing`, `onboarding`),
   never for a technical layer or framework (`usecases/http`, `usecases/db`); the tree reveals the
   domain at a glance (see [[module-boundaries]]).
4. A deployable's entry point (the composition root that wires dependencies and starts the process)
   stays separate from its library code; a library never starts a server, opens a listening socket,
   or runs a migration on its own - only the deployable's entry point does.
{{end}}

{{define "forbidden"}}
- A library depending on a deployable, or one deployable depending on another instead of sharing a
  library.
- Non-API code reachable from outside its owning boundary when the language offers a stronger
  guarantee than convention.
- A framework- or layer-named package/directory (`usecases/http`, `usecases/db`) instead of a
  capability-named one.
- A library that starts a server, opens a socket, or runs a migration outside a deployable's entry
  point.
{{end}}

{{define "validation"}}
- [ ] No library depends on a deployable; no deployable depends on another deployable.
- [ ] Non-API code is unreachable from outside its boundary via the language's own guarantee.
- [ ] Packages/directories are named for the business capability, not a framework or layer.
- [ ] The deployable's entry point is separate from its library code; only it starts the process.
{{end}}
