---
id: code-placement-inner-modules
name: Code placement - single repo with inner modules
description: One repository with several inner Go modules joined by a root go.work - cmd/internal/pkg inside each, deps via facade ports, replace directives for local cross-module deps; concrete directory map for the code-placement invariants.
tags:
    - go
    - layout
    - modules
linters:
    - depguard
analyzers:
    - arch-boundary
    - import-cycle
    - package-file-count
---

# Code placement - single repo with inner modules

> One repository containing several inner Go modules (each its own `go.mod`) joined by a root
> `go.work`, each module using the `cmd/` + `internal/` + `pkg/` layout. This is the inner-modules
> concrete instantiation of [[code-placement]]'s language-neutral invariants.

Applies to: a single repo that hosts a handful of inner modules (e.g. `modules/<name>/go.mod`, or a
root module plus a few nested ones) wired by one `go.work`, selected by the `inner-modules`
code-layout profile in `profiles.yaml`. That profile replaces the default `code-placement`
composition wholesale, so this file restates [[code-placement]]'s invariants in their concrete
inner-modules form rather than relying on it being composed alongside.

{{define "when"}}
- Creating a new inner module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for boundary violations.
{{end}}

{{define "must"}}
1. Each inner module is a self-contained Go module with its own `go.mod` and the standard
   `cmd/` + `internal/` + `pkg/` layout; deployable binaries live under `<module>/cmd/<binary>` -
   [[code-placement]]'s entry point, kept separate from the module's library code.
2. Public API lives at a module's root packages or `pkg/`; everything else stays under `internal/` so
   the compiler enforces the boundary across modules.
3. A binary module may import a library module; a library module never imports a binary module;
   resolve cross-module calls through a consumer-declared facade port - not a deep import - whose
   contract carries only Connascence of Name + Type (named interface + typed DTOs), per
   [[module-boundaries]].
4. A single root `go.work` lists every inner module; it declares no out-of-repo packages (private deps
   via `GOPRIVATE`). Local cross-module deps use a grouped `replace ( ... )` block per `go.mod` with
   clean `../` relative paths (see [[build]]).
5. Keep the module set small and intentional - split a new inner module out only when a real boundary
   (separate deploy, separate ownership, or a genuine reuse seam) justifies its own `go.mod`.
6. App internal layout still applies inside a module: `internal/bootstrap/`,
   `internal/domain/{entities,usecases/<feature>/,interop/}`, `internal/infra/...`,
   `internal/transport/...`. Name each `<feature>` for its business capability, not a framework or
   technical layer (see [[code-placement]]).
7. Keep a Go package small: past 10 non-test `.go` files, split it by responsibility into
   subdirectories (`package-file-count`, default 10 in code; not counted: generated files, an ent
   schema directory, `//go:build ignore` files, files named `_*` or `.*`, minority-package files,
   and directories Go ignores).
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
- [ ] Cross-module deps via grouped `replace ( ... )` blocks; a library never imports a binary module.
- [ ] Cross-module calls go through facade ports; non-API under `internal/`.
- [ ] Module set is minimal and boundary-justified.
{{end}}
