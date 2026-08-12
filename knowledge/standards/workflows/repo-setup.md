---
id: repo-setup
name: Repo setup
description: Bootstrap a new Go project (workspace or single module) with the standard layout, Taskfile, golangci v2 config, docs tree, and container build.
tags:
    - go
    - workflow
    - bootstrap
steps:
    - choose layout (workspace apps/libs/tools/build or single cmd/internal/pkg)
    - pin Go version; CGO_ENABLED=0 default
    - add the Taskfile with standard entrypoints
    - add golangci-lint v2 config with the enforceable subset + gofumpt/goimports/golines
    - add the docs tree (rules/patterns; domain specs in the app repo); AGENTS stop signs
    - set up the data layer (ent, atlas, sqlc, goverter)
    - container build (multi-stage, distroless, digest-pinned)
    - install the standards skills (pulsarules_cli install --project .)
    - initial commit
composes_rules:
    - code-placement
    - build
    - database
    - commits
composes_patterns:
    - app-skeleton
    - bootstrap-and-di
    - config-layout
    - embedded-migrations
---

# Repo setup workflow

> Bootstrap a new Go project (monorepo workspace or single module) with the standard layout,
> Taskfile, golangci-lint v2 config, docs tree, and container build.

## When to use

- Starting a new repository that will follow these standards.

## Steps

1. **Choose the layout.** Monorepo (workspace `go.work` + `apps/` + `libs/` + `tools/` + `build/` +
   `database/` + `test/`) or single module (`go.mod` + `cmd/` + `internal/` + `pkg/` + `build/`).
   (rule: [[code-placement]])
2. **Pin the Go version** to the newest stable; set `CGO_ENABLED=0` as the default. (rule: [[build]])
3. **Add the Taskfile** with the standard entrypoints: `task` (vet+test), `task tools:fmt`,
   `task tools:lint`, `task tools:vuln`, `task tools:mocks`, `task build:bin`, `task build:image`,
   `task gen:*`. Per-directory Taskfiles included by the root. (rule: [[build]])
4. **Add golangci-lint v2 config** with the enforceable subset (errcheck, govet, staticcheck,
   unused, cyclop, funlen, gosec, importas, interfacebloat, ireturn, modernize, nakedret, nestif,
   noctx, prealloc, predeclared, sloglint, tagalign, unconvert, unparam, whitespace, wrapcheck) and
   formatters gofumpt + goimports + golines. `testpackage` disabled (tests are same-package).
   (rule: [[build]])
5. **Add the docs tree:** `docs/rules/`, `docs/patterns/` (seed from this standards repo; keep
   domain specs under `docs/specs/` in the app repo). Add `AGENTS.md` stop signs. (rule:
   [[code-placement]])
6. **Set up the data layer** (if persistent): `ent` schema dir, Atlas migrations, `sqlc.yaml`,
   goverter. (rule: [[database]])
7. **Container build:** multi-stage, distroless, CGO-free `Containerfile` at the repo root; base
   images pinned by digest. (rule: [[build]])
8. **Install the standards skills** into the project:
   `standards-install install --project .` (or `--router-only` to start). (see INSTALL.md)
9. **Initial commit** per [[commits]]: `:sparkles: feat: bootstrap <project> skeleton`.

## References

- rules: [[code-placement]], [[build]], [[database]], [[commits]]
- patterns: [[app-skeleton]], [[bootstrap-and-di]], [[config-layout]], [[embedded-migrations]]
