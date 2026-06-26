---
id: build
name: Build & tooling
---

## Mandatory workflow

1. Default build is `CGO_ENABLED=0` (no native SQLite via CGO; use a pure-Go driver if needed). Target the newest stable
   Go the module pins.
2. Make the Taskfile the single entrypoint: `task` (vet + tests), `task tools:fmt` (gofumpt + goimports + golines),
   `task tools:lint` (golangci-lint), `task tools:vuln` (govulncheck), `task tools:mocks` (regen colocated mocks),
   `task build:bin` / `task build:image`, plus `gen:*` targets for sqlc/Atlas/seed. Per-directory Taskfiles are
   `includes:`-ed by the root.
3. Use build tags for opt-in features: `//go:build curl` (non-default TLS/HTTP backend), `//go:build tools` (tools-only
   file), `//go:build integration` (integration tests, skipped by default `go test`).
4. Keep code generation explicit and reproducible: sqlc -> generated query code; Atlas -> SQL migration files (Atlas at
   build time; the runtime applier is an in-house migrator); mockgen -> colocated `<file>_mock_test.go`; seed generators
   write idempotent SQL. Never hand-edit generated code.
5. Enable ONE golangci-lint v2 config with the enforceable subset (errcheck, govet, staticcheck, unused, cyclop, funlen,
   gosec, importas, interfacebloat, ireturn, modernize, nakedret, nestif, noctx, prealloc, predeclared, sloglint,
   tagalign, unconvert, unparam, whitespace, wrapcheck, plus wsl, nlreturn, mnd) and formatters gofumpt + goimports +
   golines. Tests are same-package so `testpackage` is disabled. Treat repo-wide PRE-EXISTING lint debt as a SEPARATE
   cleanup (do not gate new work on it); NEW code must be gofumpt + `ireturn` clean; prefer an INLINE
   `//nolint:<linter>`
   on the reported line over a global exclusion; `ireturn` permits a `generic`-typed return.
6. Keep `go.work` free of out-of-repo packages - fetch private modules via `GOPRIVATE` (env), never a `go.work use` of
   an external path. Run GOWORK-on (never force `GOWORK=off`); keep per-module `replace` directives in ONE grouped
   `replace ( … )` block per `go.mod` with tab-indented clean `../` paths (no leading spaces, no `./../`) so
   `go work sync` works and GoLand/gopls resolve symbols.
7. Build container images multi-stage, distroless-based, CGO-free; the Containerfile is at the repo root; base images
   pinned by digest.
8. Pull tool dependencies from a `tools/go.mod` (or a `//go:build tools` file). No build-time `curl | sh` downloads.

## Validation checklist

- [ ] Default build CGO-free; Go version pinned.
- [ ] Taskfile covers fmt/lint/test/gen/build/image; per-dir Taskfiles included.
- [ ] Build tags used for opt-in features and integration tests.
- [ ] Code generation reproducible; generated code not hand-edited.
- [ ] ONE golangci v2 config (subset + wsl/nlreturn/mnd); new code gofumpt + `ireturn` clean; inline `//nolint`
  preferred; pre-existing debt not gating new work.
- [ ] `go.work` free of out-of-repo packages (private via `GOPRIVATE`); GOWORK-on; one grouped `replace ( … )` block
  per `go.mod`, tab-indented clean `../` paths.
- [ ] Container multi-stage, distroless, digest-pinned; tools in `tools/go.mod`.

## Forbidden actions

- `go run` to launch a daemon - use the built binary.
- `task` targets that hide implicit network calls.
- Build-time downloads of binaries (`curl | sh`); tools not in `tools/go.mod`.
- CGO unless explicitly required by a build tag.
- Forcing `GOWORK=off`; a `go.work use` of an out-of-repo path; scattered `replace` directives or a `go.mod` indented
  with leading spaces / `./../` paths.
- Gating new work on pre-existing lint debt; a broad global lint exclusion where an inline `//nolint` would do.
- `:latest` or unpinned base images; hand-edited generated code.

## Expected outputs

- A CGO-free default build driven by a single Taskfile entrypoint.
- Reproducible codegen; a digest-pinned, distroless container image.
- A golangci-lint v2 config enforcing the curated linter subset.
