---
id: errors-logging
name: Errors & logging
---

## Mandatory workflow

1. Return `(T, error)` as the last value. `panic` only in `cmd/<app>/main.go` boot paths.
2. Wrap errors crossing a package boundary: `fmt.Errorf("action: %w", err)`. Lead with a verb (`get`, `insert`,
   `publish`), not "error while".
3. Define sentinels as exported vars (`var ErrNotFound = errors.New(...)`); compare with `errors.Is`/`errors.As`, never
   on `err.Error()` strings.
4. For HTTP/gRPC, return the domain-error type (`apperr.NotFound`, `apperr.Forbidden`, `apperr.Conflict`,
   `apperr.BadRequest`, `apperr.InternalError`) carrying a code and status. The use case returns it; transport
   middleware maps it to a status via the Go 1.26 typed extraction `errors.AsType[statusCoder](err)` (not a manual
   `errors.As` + type switch). Never write an HTTP status from a use case.
5. Use `errors.Join` to fold multiple errors (e.g. rollback + primary). Use `fmt.Errorf` + `errors.New` + `errors.Join`
   only; never `github.com/pkg/errors`.
6. Log with `log/slog` only, typed attributes (`slog.String`, `slog.Int`, `slog.Any`, `slog.Group`); never positional
   `slog.Info("msg", "key", v)`.
7. Log an error exactly once, at the top of the call chain. Do not log AND return the same error (log OR return).
8. Messages lowercase, short, no trailing punctuation; attribute keys `snake_case` (`request_id`, `user_id`). Never log
   secrets, tokens, Authorization headers, raw bodies, or PII - log IDs and a redacted summary.
9. Inside libraries and handlers, take a `*slog.Logger` from the constructor or context. `slog.Info` convenience funcs
   only at the top of `main()`; `slog.SetDefault` in `main()` before any other call, never in a library.

## Validation checklist

- [ ] All boundary-crossing errors wrapped with `%w` and an action-verb prefix.
- [ ] Sentinels compared via `errors.Is`/`errors.As`, not string match.
- [ ] Transport failures return the domain-error type; status mapped via `errors.AsType[statusCoder]`, not a manual
  type switch; no status codes written from use cases.
- [ ] `errors.Join` used to fold multiple errors; no `pkg/errors`.
- [ ] Logging uses `slog` typed attributes only; no positional kv; no `fmt.Println`.
- [ ] Each error logged once at the chain top, never logged-and-returned.
- [ ] No secrets/tokens/PII/raw bodies in logs; messages lowercase, keys `snake_case`.
- [ ] Libraries take a `*slog.Logger`; `slog.SetDefault` set in `main()` first.

## Forbidden actions

- `(nil, nil)`-style soft failures; `panic` in libraries; `recover()` to mask bugs (only at a worker-supervisor
  boundary).
- Comparing errors on `.Error()` strings; writing an HTTP/gRPC status from inside a use case.
- `github.com/pkg/errors`; logging an error AND returning it.
- `fmt.Println`/`log.Printf`/third-party loggers; positional slog args.
- A global logger inside libraries; logging secrets, JWTs, raw bodies, or PII.
- `slog.SetDefault` in a library.

## Expected outputs

- Errors wrapped with action-verb context and compared by identity; domain errors mapped to status by middleware.
- One structured `slog` line per error at the chain top, with typed `snake_case` attributes and no PII.
