---
id: code-placement
name: Code placement
description: Deployable binaries vs reusable libs vs tooling vs build outputs; one-way apps-depend-on-libs; non-API code under internal/.
tags:
    - go
    - layout
linters:
    - depguard
---

# Code placement

> Deployable binaries vs reusable libraries vs tooling vs build outputs; one-way apps-depend-on-libs
> rule (no cross-app imports, libs never import apps); non-API code under `internal/`.

Applies to: monorepo and single-module Go project layout. Two source layouts converge on the same
principle: the `apps/` + `libs/` + `tools/` + `build/` workspace (monorepo) and the
`cmd/` + `internal/` + `pkg/` layout (single module). Both enforce the same dependency direction.

{{define "when"}}
- Creating a new module, app, or lib.
- Deciding where a file or package belongs.
- Adding a cross-module dependency.
- Reviewing layout for boundary violations.
{{end}}

{{define "must"}}
1. New code goes only in allowed top-level dirs. Monorepo: `apps/`, `libs/`, `tools/`, `build/`,
   `database/`, `docs/`, `test/`. Single module: `cmd/`, `internal/`, `pkg/`, `build/`. No new
   top-level dirs.
2. Deployable binaries have their own `go.mod` (monorepo) or live under `cmd/<binary>/` (single
   module), with `main.go` + `internal/`. Reusable modules live under `libs/` (monorepo) or
   `pkg/` (single module), expose a public API + keep the rest under `internal/`, and never start
   servers or run migrations.
3. Dependency direction is one-way: binaries may import libs/pkg; libs/pkg never import apps/cmd;
   no app imports another app. Share via libs/pkg.
4. `tools/` (Taskfile, linter config, codegen) is never a runtime dependency. `build/` holds
   Dockerfiles/entrypoints/init - not schema/migrations/codegen.
5. Schema authoring, migrations, and query files live in a dedicated `database/` (monorepo) or
   `build/{entschema,migrations}/` dir; generated query code lands in the infra module. Cross-app
   integration tests live in a separate `test/` module.
6. App internal layout: `internal/bootstrap/`, `internal/domain/{entities,usecases/<feature>/,interop/}`,
   `internal/infra/repositories/<dbms>/<feature>repo/`, `internal/transport/{rest,grpc}/<feature>*/`.
7. Repositories export nothing beyond their constructor; keep non-API code under `internal/` (the
   compiler enforces the boundary).
8. Register every new module in the root `go.work` (workspace) or `go.mod` (single module).
9. Domain package names scream the business capability, not the framework or technical layer: a
   feature under `internal/domain/usecases/<feature>/` is named for what it does (`billing`,
   `onboarding`), never `usecases/http`, `usecases/db`, or a framework name. The tree reveals the
   domain at a glance (see [[module-boundaries]] on workflow partitioning).
{{end}}

{{define "forbidden"}}
- `apps/foo` importing `apps/bar` (extract to `libs/`); `cmd/a` importing `cmd/b`.
- `libs/x` importing `apps/...`; `pkg/` importing `cmd/...`.
- Exported API in a module root that should be under `internal/`.
- New top-level directories outside the allowed set.
- Schema/migrations/codegen placed in `build/` (in the monorepo layout).
- A framework- or layer-named domain package (`usecases/http`, `usecases/db`) instead of a
  capability-named one.
{{end}}

{{define "validation"}}
- [ ] New file/package lives under an allowed top-level dir.
- [ ] No binary importing another binary; no lib importing a binary.
- [ ] Non-API code is under `internal/`.
- [ ] Domain/infra/transport placed per the standard app layout.
- [ ] `go.work`/`go.mod` updated for new modules.
- [ ] Domain packages named for the business capability, not a framework or technical layer.
{{end}}
