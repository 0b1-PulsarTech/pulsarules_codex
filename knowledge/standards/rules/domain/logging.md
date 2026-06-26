---
id: logging
name: Logging
description: log/slog only, typed attributes (never positional), one log per error at the chain top, no secrets/PII.
tags:
    - go
    - logging
linters:
    - sloglint
    - forbidigo
---

# Logging

> `log/slog` only, typed attributes (never positional), one log per error at the chain top, no
> secrets/PII.

Applies to: all production logging. Canonical reference: `log/slog` directly, no wrapper logger.

{{define "when"}}
- Adding any log statement in production code.
- Deciding a log level or whether to log vs return.
- Choosing a logger source (global vs injected).
{{end}}

{{define "must"}}
1. Use `log/slog` only. Convenience funcs (`slog.Info`) only at the top of `main()`/boot; inside
   libraries and handlers take a `*slog.Logger` from the constructor or context.
2. Always typed attributes: `slog.String`, `slog.Int`, `slog.Any`, `slog.Group`. Never positional
   `slog.Info("msg", "key", v)` (sloglint rejects it).
3. Log an error exactly once, at the top of the call chain: `slog.Error("msg", slog.String("error",
   err.Error()))`. Do not log AND return the same error (see [[errors]]).
4. Lowercase, short, no-trailing-punctuation messages; `snake_case` attribute keys
   (`request_id`, `user_id`).
5. Never log secrets, tokens, Authorization headers, raw request/response bodies, or PII. IDs are
   fine; log a redacted summary (`slog.Int("body_size", n)`) when unsure.
6. Set `slog.SetDefault` in `main()` before any other call; never in a library.
{{end}}

{{define "forbidden"}}
- `fmt.Println`/`log.Printf`/third-party loggers in production code.
- Positional slog args; a global logger inside libraries instead of an injected/context logger.
- Logging secrets, JWTs, raw bodies, or customer PII.
- Logging an error then returning it (log OR return).
- `slog.SetDefault` in a library.
{{end}}

{{define "validation"}}
- [ ] Logging uses `slog` typed attributes only; no positional kv; no `fmt.Println`.
- [ ] Each error logged once at the chain top, never logged-and-returned.
- [ ] No secrets/tokens/PII/raw bodies in logs; messages lowercase, keys `snake_case`.
- [ ] Libraries take a `*slog.Logger`; `slog.SetDefault` set in `main()` first.
{{end}}
