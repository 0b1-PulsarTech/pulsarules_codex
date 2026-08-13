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
   `GOWORK=off`). Keep per-module `replace` directives in ONE grouped `replace ( ... )` block per
   `go.mod` with clean `../` relative paths (tabs, not leading spaces; never `./../`). `go work sync`
   needs the replaces, and a malformed `go.mod` silently breaks GoLand/gopls symbol resolution.
7. Container images are multi-stage, distroless-based, CGO-free; the Containerfile is at the repo
   root; base images pinned by digest.
8. Tool dependencies come from a `tools/go.mod` (or `//go:build tools` file); no build-time
   `curl | sh` downloads.
9. A rule this knowledge base ships is a suggestion, not a guardrail, until an analyzer enforces it
   (see [[skill-authoring]]). `internal/analyzer/delegation/golangcilint` runs golangci-lint against
   a TARGET project using THAT project's own discovered config, so a linter enabled only in this
   repo's `.golangci.yml` never governs a consumer. The delegation runner forces a small baseline
   via `-E` on every delegated run - `nolintlint`, `paralleltest`, `tparallel`, `thelper`,
   `forcetypeassert`, `nilerr` - on top of whatever the target's own config already enables
   (`golangci-lint run -E foo` adds `foo`, it does not replace the config's enabled set; verified
   against golangci-lint v2.12.2). The forcing has ONE hole, and it is not "non-negotiable": a
   target that names a forced linter under `linters.disable` WINS over `-E`, and the run then
   reports zero findings for it with no indication the guard was switched off (verified on v2.12.2
   under both `default: standard` and `default: all`). Merely omitting the linter from an `enable:`
   list does NOT do this - `-E` still adds it. So the baseline is forced against silence, not
   against a consumer who explicitly opted out; the delegation runner reads the target's config and
   WARNS when it finds a forced linter disabled, because a guard that reports nothing must never be
   indistinguishable from a guard that found nothing. This is how `nolintlint` enforces "every `//nolint`
   names its linter and carries a reason" (this rule, above) and how the testing rule's
   `t.Parallel()`/`t.Helper()` obligations become machine-checked in every consumer, not just here.
   `nolintlint`'s own `require-explanation`/`require-specific` settings cannot be forced through
   `-E` - it only enables the linter at its permissive defaults - so a consumer wanting that
   stricter enforcement sets those two settings itself; without them, the forced `nolintlint` still
   catches unused and malformed `//nolint` directives, just not a bare unqualified one.
   `bodyclose` and `sqlclosecheck` are RECOMMENDED, not forced: forcing them can flood a legacy
   project that never ran them, and a gate that floods is a gate people switch off. `revive` is
   RECOMMENDED with its `defer` rule and the `loop` argument, which is what catches the
   defer-inside-a-loop clause in `[[safety]]`. It cannot be forced: `-E` enables a linter but its
   settings come from the target's own config, and `defer` is not in revive's defaults, so forcing
   revive would list a guard that catches nothing. Enable it locally to get that clause enforced.
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
  `replace ( ... )` block per `go.mod` with tab-indented clean `../` paths.
- [ ] Container multi-stage, distroless, digest-pinned; tools in `tools/go.mod`.
- [ ] The golangci-lint delegation runner forces `nolintlint`, `paralleltest`, `tparallel`,
  `thelper`, `forcetypeassert`, `nilerr` via `-E` on every target project, and WARNS when the
  target's own config disables one of them (the one case `-E` loses); `bodyclose` and
  `sqlclosecheck` are documented as recommended, not forced.
{{end}}
