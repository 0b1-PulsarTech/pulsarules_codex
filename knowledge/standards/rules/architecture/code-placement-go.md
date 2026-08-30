---
id: code-placement-go
name: Code placement - single Go module
description: The cmd/internal/pkg single-module layout - cmd/<binary> as the composition root, pkg/ for a reusable public API, internal/ for everything else, database/build placement for schema and migrations, go.mod registration. The Go concrete instantiation of code-placement.
tags:
    - go
    - layout
linters:
    - depguard
analyzers:
    - arch-boundary
    - import-cycle
---

# Code placement - single Go module

> A single Go module laid out as `cmd/` + `internal/` + `pkg/` + `build/`: a deployable binary lives
> under `cmd/<binary>/` with `main.go` plus its own `internal/`; a reusable library lives under
> `pkg/`, exposing a public API with the rest under `internal/`. This is the Go concrete instantiation
> of [[code-placement]]'s language-neutral invariants.

Applies to: a single-module Go project (one root `go.mod`). Composes with [[code-placement]] and
[[dependency-rule-go]].

{{define "when"}}
- Creating a new module, app, or lib in a single-module Go project.
- Deciding where a file or package belongs.
- Adding a cross-package dependency.
- Reviewing layout for a boundary violation.
{{end}}

{{define "must"}}
1. New code goes only in `cmd/`, `internal/`, `pkg/`, or `build/`. No new top-level dirs.
2. A deployable binary lives under `cmd/<binary>/`, with `main.go` plus its own `internal/` - this is
   the entry point [[code-placement]] requires kept separate from library code. A reusable module
   lives under `pkg/`, exposing a public API with the rest under `internal/`; it never starts servers
   or runs migrations (see [[code-placement]]).
3. `build/` holds Dockerfiles, entrypoints, and init scripts - never schema, migrations, or codegen.
   `tools/` (Taskfile, linter config, codegen), where present, is never a runtime dependency.
4. Schema authoring, migrations, and query files live under `build/{entschema,migrations}/`;
   generated query code lands in the infra package.
5. App internal layout: `internal/bootstrap/`,
   `internal/domain/{entities,usecases/<feature>/,interop/}`,
   `internal/infra/repositories/<dbms>/<feature>repo/`, `internal/transport/{rest,grpc}/<feature>*/`.
   Name each `<feature>` for its business capability (see [[code-placement]]), never a framework or
   technical layer.
6. Repositories export nothing beyond their constructor; non-API code stays under `internal/`, which
   the compiler enforces as the boundary (see [[code-placement]]).
7. Register every new package's module in the root `go.mod`.
{{end}}

{{define "forbidden"}}
- A new top-level directory outside `cmd/`, `internal/`, `pkg/`, `build/`.
- `cmd/a` importing `cmd/b`; `pkg/` importing `cmd/`.
- Exported API in a module root that should be under `internal/`.
- Schema, migrations, or codegen placed in `build/` instead of `build/{entschema,migrations}/`.
- Domain/infra/transport code placed outside the standard `internal/` app layout.
{{end}}

{{define "validation"}}
- [ ] New file/package lives under `cmd/`, `internal/`, `pkg/`, or `build/`.
- [ ] No binary importing another binary; no `pkg/` importing `cmd/`.
- [ ] Non-API code is under `internal/`.
- [ ] Domain/infra/transport placed per the standard app layout, capability-named.
- [ ] `go.mod` registers every new package.
{{end}}
