---
id: code-placement-monorepo
name: Code placement - monorepo layout
description: The apps/libs/tools/build/database/test monorepo - one go.mod per module joined by a root go.work, one composition root per app, one-way apps-depend-on-libs, non-API under internal/.
tags:
    - go
    - layout
    - monorepo
linters:
    - depguard
---

# Code placement - monorepo layout

> The multi-module monorepo: `apps/` + `libs/` + `tools/` + `build/` + `database/` + `test/`, each
> module carrying its own `go.mod`, all joined by a single root `go.work`; one-way
> apps-depend-on-libs; non-API code under `internal/`.

Applies to: a monorepo where every deployable app and every shared library is its own Go module.

{{define "when"}}
- Creating a new module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for boundary violations.
{{end}}

{{define "must"}}
1. Top-level dirs are exactly `apps/`, `libs/`, `tools/`, `build/`, `database/`, `docs/`, `test/`.
   No new top-level dirs.
2. Each app is its own module: `apps/<app>/go.mod`, the composition root at `apps/<app>/cmd/<app>`,
   the rest under `apps/<app>/internal/`. Shared code is its own module under `libs/<lib>/`, exposing
   a public API with the rest under `internal/`; a lib never starts servers or runs migrations.
3. Dependency direction is one-way: `apps/*` may import `libs/*`; `libs/*` never import `apps/*`; no
   app imports another app - share via a `libs/` module or a consumer-declared facade port whose
   contract carries only Connascence of Name + Type (named interface + typed DTOs), per
   [[module-boundaries]].
4. Join every module in the ROOT `go.work`; `go.work` declares no out-of-repo packages (private deps
   via `GOPRIVATE`). Cross-module local deps use a grouped `replace ( … )` block per `go.mod` with
   clean `../` relative paths (see [[build]]).
5. `database/` owns schema authoring, migrations, and query files; `build/` owns
   Dockerfiles/entrypoints; `tools/` (Taskfile, linters, codegen) is never a runtime dependency.
   Cross-app integration tests live in the `test/` module.
6. App internal layout: `internal/bootstrap/`, `internal/domain/{entities,usecases/<feature>/,interop/}`,
   `internal/infra/repositories/<dbms>/<feature>repo/`, `internal/transport/{rest,grpc}/<feature>*/`.
   Keep non-API code under `internal/`; the compiler enforces the boundary. Name each `<feature>` for
   its business capability, not a framework or technical layer (see [[module-boundaries]]).
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
- [ ] One-way apps-depend-on-libs; no app-to-app imports.
- [ ] `database/` holds schema/migrations; `tools/`/`build/` carry no runtime deps.
- [ ] Domain/infra/transport placed per the standard app layout.
{{end}}
