---
id: code-placement-inner-modules
name: Code placement - single repo with inner modules
description: One repository with several inner Go modules joined by a root go.work - cmd/internal/pkg inside each, one-way deps via facade ports, replace directives for local cross-module deps, non-API under internal/.
tags:
    - go
    - layout
    - modules
linters:
    - depguard
analyzers:
    - arch-boundary
    - import-cycle
---

# Code placement - single repo with inner modules

> One repository containing several inner Go modules (each its own `go.mod`) joined by a root
> `go.work`, each module using the `cmd/` + `internal/` + `pkg/` layout; one-way dependency direction
> across modules; non-API code under `internal/`.

Applies to: a single repo that hosts a handful of inner modules (e.g. `modules/<name>/go.mod`, or a
root module plus a few nested ones) wired by one `go.work`.

{{define "when"}}
- Creating a new inner module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for boundary violations.
{{end}}

{{define "must"}}
1. Each inner module is a self-contained Go module with its own `go.mod` and the standard
   `cmd/` + `internal/` + `pkg/` layout; deployable binaries live under `<module>/cmd/<binary>`.
2. Public API lives at a module's root packages or `pkg/`; everything else stays under `internal/` so
   the compiler enforces the boundary across modules.
3. Dependency direction is one-way across inner modules: a binary module may import a library module;
   a library module never imports a binary module; resolve cross-module calls through a
   consumer-declared facade port - not a deep import - whose contract carries only Connascence of Name
    + Type (named interface + typed DTOs), per [[module-boundaries]].
4. A single root `go.work` lists every inner module; it declares no out-of-repo packages (private deps
   via `GOPRIVATE`). Local cross-module deps use a grouped `replace ( ... )` block per `go.mod` with
   clean `../` relative paths (see [[build]]).
5. Keep the module set small and intentional - split a new inner module out only when a real boundary
   (separate deploy, separate ownership, or a genuine reuse seam) justifies its own `go.mod`.
6. App internal layout still applies inside a module: `internal/bootstrap/`,
   `internal/domain/{entities,usecases/<feature>/,interop/}`, `internal/infra/...`,
   `internal/transport/...`. Name each `<feature>` for its business capability, not a framework or
   technical layer (see [[module-boundaries]]).
{{end}}

{{define "forbidden"}}
- Inner modules wired by ad-hoc `GOFLAGS`/`GOWORK=off` instead of one root `go.work`.
- A library inner module importing a binary inner module, or a deep import across a module boundary
  that a facade port should mediate.
- Public API in a module root that belongs under `internal/`.
- Splitting a module out with no real boundary (premature modularization).
{{end}}

{{define "validation"}}
- [ ] Each inner module has its own `go.mod` and `cmd/internal/pkg` layout; joined by one root `go.work`.
- [ ] Cross-module deps via grouped `replace ( ... )` blocks; one-way direction preserved.
- [ ] Cross-module calls go through facade ports; non-API under `internal/`.
- [ ] Module set is minimal and boundary-justified.
{{end}}
