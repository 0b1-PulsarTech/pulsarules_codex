---
id: code-placement
name: Code placement
---

## Mandatory workflow

1. Place new code only in an allowed top-level dir. Monorepo: `apps/`, `libs/`, `tools/`, `build/`, `database/`,
   `docs/`, `test/`. Single module: `cmd/`, `internal/`, `pkg/`, `build/`. No new top-level dirs.
2. Deployable binaries live under `apps/<name>/` (monorepo, own `go.mod`) or `cmd/<name>/` (single module), with
   `main.go` + `internal/`. Reusable modules live under `libs/` (monorepo) or `pkg/` (single module), expose a public
   API and keep the rest under `internal/`; they never start servers or run migrations.
3. Honour the one-way dependency direction: binaries may import libs/pkg; libs/pkg never import apps/cmd; no app imports
   another app. Share via libs/pkg.
4. Keep `tools/` (Taskfile, linter config, codegen) out of the runtime graph; `build/` holds
   Dockerfiles/entrypoints/init - not schema, migrations, or codegen.
5. Put schema authoring, migrations, and query files in a dedicated `database/` (monorepo) or
   `build/{entschema,migrations}/` dir; generated query code lands in the infra module. Cross-app integration tests live
   in a separate `test/` module.
6. Within an app, lay out `internal/bootstrap/`, `internal/domain/{entities,usecases/<feature>/,interop/}`,
   `internal/infra/repositories/<dbms>/<feature>repo/`, `internal/transport/{rest,grpc}/<feature>*/`.
7. Repositories export nothing beyond their constructor; keep non-API code under `internal/` (the compiler enforces the
   boundary).
8. Register every new module in the root `go.work` (workspace) or `go.mod` (single module).

## Validation checklist

- [ ] New file/package lives under an allowed top-level dir.
- [ ] No binary importing another binary; no lib importing a binary.
- [ ] Non-API code is under `internal/`.
- [ ] Domain/infra/transport placed per the standard app layout.
- [ ] `go.work`/`go.mod` updated for new modules.

## Forbidden actions

- `apps/foo` importing `apps/bar` (extract to `libs/`); `cmd/a` importing `cmd/b`.
- `libs/x` importing `apps/...`; `pkg/` importing `cmd/...`.
- Exported API in a module root that should be under `internal/`.
- New top-level directories outside the allowed set.
- Schema/migrations/codegen placed in `build/` (in the monorepo layout).

## Expected outputs

- A file/package placed in the dir that matches its role and lifetime.
- A dependency graph that flows one way (binaries -> libs -> std); no cycles.
- New modules registered in the workspace/module file.
