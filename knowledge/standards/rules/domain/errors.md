---
id: errors
name: Errors
description: Sentinel errors, %w wrapping across boundaries, a domain-error contract for transport status mapping; never pkg/errors, never string-matching.
tags:
    - go
    - errors
dependencies:
    - logging
linters:
    - errcheck
    - wrapcheck
    - staticcheck
    - forbidigo
analyzers:
    - golangci-lint
---

# Errors

> Sentinel errors, `%w` wrapping across boundaries, a domain-error contract for transport status
> mapping; never `pkg/errors`, never string-matching.

Applies to: any function that returns an error. Canonical reference: a domain-error type carrying
`Code()`/`StatusCode()`/`DetailMsg()`/`Unwrap()` (source repos name it `apperr`).

{{define "when"}}
- A function returns an error.
- Forwarding an error across a package boundary.
- Defining a sentinel error callers branch on.
- Creating a domain error that maps to an HTTP/gRPC status code.
{{end}}

{{define "must"}}
1. Return `(T, error)` as the last value. `panic` only in `cmd/<app>/main.go` boot paths.
2. Wrap errors crossing a package boundary: `fmt.Errorf("action: %w", err)`. Lead with a verb
   (`get`, `insert`, `publish`), not "error while".
3. Define sentinels as exported vars (`var ErrNotFound = errors.New(...)`); compare with
   `errors.Is`/`errors.As`, never on `err.Error()` strings.
4. For HTTP/gRPC, return the domain-error type (`apperr.NotFound`, `apperr.Forbidden`,
   `apperr.Conflict`, `apperr.BadRequest`, `apperr.InternalError`, `apperr.ErrForbidden`) carrying
   a code and status. The use case returns it; transport middleware reads the code and renders the
   status. Map the wrapped error to its status with the Go 1.26 typed extraction
   `errors.AsType[statusCoder](err)` (a `statusCoder` interface the domain-error implements), NOT a
   manual `errors.As` + type switch. Never write an HTTP status from a use case.
5. Use `errors.Join` to fold multiple errors (e.g. rollback + primary).
6. Use `fmt.Errorf` + `errors.New` + `errors.Join`; never `github.com/pkg/errors`.
{{end}}

{{define "forbidden"}}
- `(nil, nil)`-style soft failures instead of an error.
- `panic` in libraries; `recover()` to mask bugs (only at a worker-supervisor boundary).
- Comparing errors on `.Error()` strings.
- Writing an HTTP/gRPC status from inside a use case.
- `github.com/pkg/errors`.
- Logging an error AND returning it (log OR return; see [[logging]]).
{{end}}

{{define "validation"}}
- [ ] All boundary-crossing errors wrapped with `%w` and an action-verb prefix.
- [ ] Sentinels compared via `errors.Is`/`errors.As`, not string match.
- [ ] Transport failures return the domain-error type; no status codes written from use cases.
- [ ] Status mapping uses `errors.AsType[statusCoder](err)`, not a manual `errors.As` + type switch.
- [ ] `errors.Join` used to fold multiple errors.
- [ ] No `pkg/errors`; no `panic` in libraries.
{{end}}
