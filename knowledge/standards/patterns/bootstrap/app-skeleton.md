---
id: app-skeleton
name: App skeleton
description: New deployable binary skeleton with own go.mod, main.go, internal layout, Dockerfile, and Taskfile/compose entries.
tags:
    - go
    - bootstrap
    - layout
dependencies:
    - code-placement
composes:
    - bootstrap-and-di
---

# App skeleton

> New deployable binary skeleton: own `go.mod`, `main.go`, `internal/{bootstrap,domain,infra,
> transport}`, `hookconf/`, Dockerfile, Taskfile/compose entries.

{{define "when"}}
- Bootstrapping a new deployable binary under `apps/` (monorepo) or `cmd/<name>/` (single module).
{{end}}

{{define "recipe"}}
```
apps/<name>/                 # or cmd/<name>/
├── go.mod                   # own module (monorepo); add to go.work
├── main.go                  # thin boot (see bootstrap-and-di)
├── hookconf/
│   └── config.go            # typed Config for this app
├── internal/
│   ├── bootstrap/           # composition root; the only switchboard
│   │   ├── config.go
│   │   ├── database.go
│   │   ├── migrations.go
│   │   ├── register_injections.go
│   │   └── web_server.go
│   ├── domain/
│   │   ├── entities/        # domain types
│   │   ├── apperr/          # domain-error contract
│   │   ├── interop/facades/ # cross-module facade impls
│   │   └── usecases/<feature>/
│   ├── infra/
│   │   └── repositories/<dbms>/<feature>repo/
│   └── transport/
│       ├── rest/<feature>handler/
│       └── grpc/<service>server/
├── Dockerfile               # --build-arg APP=<name>
├── Taskfile.build.yml       # build/run targets
└── conf.toml.example
```

Register in `go.work`:

```
go 1.26

use (
    ./apps/<name>
    ./libs/...
)
```

`main.go` follows the bootstrap-and-di recipe. `panic` is allowed only here, on impossible boot
states.
{{end}}

{{define "forbidden"}}
- New top-level directories outside the allowed set (see [[code-placement]]).
- Schema/migrations/codegen placed in `build/`.
- A binary that starts servers or runs migrations from a library.
{{end}}

{{define "validation"}}
- [ ] `apps/<name>/` (or `cmd/<name>/`) skeleton with the six bootstrap files.
- [ ] `conf.toml.example`, Dockerfile, Taskfile/compose entries present.
- [ ] Added to `go.work` (monorepo) or `go.mod` (single module).
{{end}}
