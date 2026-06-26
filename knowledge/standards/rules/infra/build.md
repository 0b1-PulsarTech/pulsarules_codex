---
id: build
name: Build & tooling
description: CGO-free default build; Taskfile entrypoints for fmt/lint/test/gen; reproducible codegen; container images pinned by digest; no go run for daemons.
references:
    - golangci-lint
    - taskfile
    - govulncheck
tags:
    - go
    - build
dependencies:
    - code-placement
linters:
    - golangci-lint
    - govulncheck
---

# Build & tooling

> CGO-free default build; build tags for opt-in features; Taskfile entrypoints for fmt/lint/test/
> gen; code generation (sqlc, Atlas, mockgen, seed) is explicit and reproducible; container base
> images pinned by digest; no `go run` to launch daemons; no `curl | sh` build-time downloads.

Applies to: build, tooling, and code generation.

{{define "when"}}
- Adding or changing a Taskfile target, linter config, or build tag.
- Running code generation (sqlc, Atlas, mockgen, seed).
- Building a binary or container image.
- Adding a tool dependency.
{{end}}

{{define "must"}}
1. Default build is `CGO_ENABLED=0` (no native SQLite via CGO; use a pure-Go driver if needed). Target
   the newest stable Go the module pins.
2. Taskfile is the single entrypoint: `task` (vet + tests), `task tools:fmt` (gofumpt + goimports +
   golines), `task tools:lint` (golangci-lint), `task tools:vuln` (govulncheck), `task tools:mocks`
   (regen colocated mocks), `task build:bin` (binaries), `task build:image` (container), plus `gen:*`
   targets for sqlc/Atlas/seed. Per-directory Taskfiles are `includes:`-ed by the root.
3. Build tags: `//go:build curl` (opt into a non-default TLS/HTTP backend), `//go:build tools`
   (tools-only file), `//go:build integration` (integration tests, skipped by default `go test`).
4. Code generation is explicit and reproducible: sqlc -> generated query code; Atlas -> SQL
   migration files (Atlas at build time; the runtime applier is an in-house migrator); mockgen ->
   colocated `<file>_mock_test.go`; seed generators write idempotent SQL.
5. ONE golangci-lint v2 config enables the enforceable subset (errcheck, govet, staticcheck, unused,
   cyclop, funlen, gosec, importas, interfacebloat, ireturn, modernize, nakedret, nestif, noctx,
   prealloc, predeclared, sloglint, tagalign, unconvert, unparam, whitespace, wrapcheck, plus wsl,
   nlreturn, mnd) with formatters gofumpt + goimports + golines. Tests are same-package so
   `testpackage` is disabled. Repo-wide PRE-EXISTING lint debt is a SEPARATE cleanup - do NOT gate new
   work on it; NEW code must be gofumpt + `ireturn` clean. Prefer an INLINE `//nolint:<linter>` on the
   exact reported line/declaration over a global config exclusion; `ireturn` permits a `generic`-typed
   return.
6. Workspace: `go.work` declares NO out-of-repo packages - fetch private modules via `GOPRIVATE` (env),
   never a `go.work use` of an external path. Run the workspace in GOWORK-on mode (do NOT force
   `GOWORK=off`). Keep per-module `replace` directives in ONE grouped `replace ( … )` block per
   `go.mod` with clean `../` relative paths (tabs, not leading spaces; never `./../`). `go work sync`
   needs the replaces, and a malformed `go.mod` silently breaks GoLand/gopls symbol resolution.
7. Container images are multi-stage, distroless-based, CGO-free; the Containerfile is at the repo
   root; base images pinned by digest.
8. Tool dependencies come from a `tools/go.mod` (or `//go:build tools` file); no build-time
   `curl | sh` downloads.
{{end}}

{{define "forbidden"}}
- `go run` to launch a daemon - use the built binary.
- `task` targets that hide implicit network calls.
- Build-time downloads of binaries (`curl | sh`); tools not in `tools/go.mod`.
- CGO unless explicitly required by a build tag.
- Forcing `GOWORK=off`; a `go.work use` of an out-of-repo path; scattered/duplicate `replace`
  directives or `go.mod` indented with leading spaces or `./../` paths.
- Gating new work on pre-existing lint debt; a broad global lint exclusion where an inline `//nolint`
  on the reported line would do.
- `:latest` or unpinned base images; hand-edited generated code.
{{end}}

{{define "validation"}}
- [ ] Default build CGO-free; Go version pinned.
- [ ] Taskfile covers fmt/lint/test/gen/build/image; per-dir Taskfiles included.
- [ ] Build tags used for opt-in features and integration tests.
- [ ] Code generation reproducible; generated code not hand-edited.
- [ ] ONE golangci v2 config (enforceable subset + wsl/nlreturn/mnd + gofumpt/goimports/golines); new
  code gofumpt + `ireturn` clean; inline `//nolint` preferred over global excludes; pre-existing debt
  not gating new work.
- [ ] `go.work` declares no out-of-repo packages (private via `GOPRIVATE`); GOWORK-on; one grouped
  `replace ( … )` block per `go.mod` with tab-indented clean `../` paths.
- [ ] Container multi-stage, distroless, digest-pinned; tools in `tools/go.mod`.
{{end}}
