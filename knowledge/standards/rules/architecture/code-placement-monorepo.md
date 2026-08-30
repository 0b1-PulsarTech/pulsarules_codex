---
id: code-placement-monorepo
name: Code placement - monorepo layout
description: The apps/libs/tools/build/database/test monorepo - one go.mod per module joined by a root go.work, one composition root per app, concrete directory map for the code-placement invariants.
tags:
    - go
    - layout
    - monorepo
linters:
    - depguard
analyzers:
    - arch-boundary
    - import-cycle
---

# Code placement - monorepo layout

> The multi-module monorepo: `apps/` + `libs/` + `tools/` + `build/` + `database/` + `test/`, each
> module carrying its own `go.mod`, all joined by a single root `go.work`. This is the monorepo
> concrete instantiation of [[code-placement]]'s language-neutral invariants (one-way
> apps-depend-on-libs, non-API under `internal/`, capability-named packages, the composition root
> kept separate from library code).

Applies to: a monorepo where every deployable app and every shared library is its own Go module,
selected by the `monorepo` code-layout profile in `profiles.yaml`. That profile replaces the default
`code-placement` composition wholesale, so this file restates [[code-placement]]'s invariants in
their concrete monorepo form rather than relying on it being composed alongside.

{{define "when"}}
- Creating a new module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for boundary violations.
{{end}}

{{define "must"}}
1. Top-level dirs are exactly `apps/`, `libs/`, `tools/`, `build/`, `database/`, `docs/`, `test/`.
   No new top-level dirs.
2. Each app is its own module: `apps/<app>/go.mod`, the composition root at `apps/<app>/cmd/<app>`
   ([[code-placement]]'s entry point), the rest under `apps/<app>/internal/`. Shared code is its own
   module under `libs/<lib>/`, exposing a public API with the rest under `internal/`; a lib never
   starts servers or runs migrations.
3. `apps/*` may import `libs/*`; `libs/*` never import `apps/*`; no app imports another app - share
   via a `libs/` module or a consumer-declared facade port whose contract carries only Connascence of
   Name + Type (named interface + typed DTOs), per [[module-boundaries]].
4. Join every module in the root `go.work`; `go.work` declares no out-of-repo packages (private deps
   via `GOPRIVATE`). Cross-module local deps use a grouped `replace ( ... )` block per `go.mod` with
   clean `../` relative paths (see [[build]]).
5. `database/` owns schema authoring, migrations, and query files; `build/` owns
   Dockerfiles/entrypoints; `tools/` (Taskfile, linters, codegen) is never a runtime dependency.
   Cross-app integration tests live in the `test/` module.
6. App internal layout: `internal/bootstrap/`, `internal/domain/{entities,usecases/<feature>/,interop/}`,
   `internal/infra/repositories/<dbms>/<feature>repo/`, `internal/transport/{rest,grpc}/<feature>*/`.
   Keep non-API code under `internal/`; the compiler enforces the boundary. Name each `<feature>` for
   its business capability, not a framework or technical layer (see [[code-placement]] and
   [[module-boundaries]]).
{{end}}

{{define "forbidden"}}
- A new top-level dir outside the allowed set.
- An app module without its own `go.mod`, or an app importing another app's `internal/`.
- A `libs/` module importing an `apps/` module.
- Schema/migrations placed under `build/` instead of `database/`.
- Exported API in a module root that belongs under `internal/`.
{{end}}

{{define "validation"}}
- [ ] Every app and lib is its own module joined by the root `go.work`.
- [ ] Composition root at `apps/<app>/cmd/<app>`; non-API under `internal/`.
- [ ] No app-to-app imports; `libs/*` never imports `apps/*`.
- [ ] `database/` holds schema/migrations; `tools/`/`build/` carry no runtime deps.
- [ ] Domain/infra/transport placed per the standard app layout.
{{end}}
