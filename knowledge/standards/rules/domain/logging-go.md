---
id: logging-go
name: Logging - log/slog
description: log/slog only, typed attributes (never positional), lowercase snake_case-keyed messages, slog.SetDefault only in main(). The Go instantiation of the neutral logging rule.
tags:
    - go
    - logging
linters:
    - sloglint
    - forbidigo
---

# Logging - log/slog

> `log/slog` only, typed attributes (never positional), lowercase no-trailing-punctuation messages
> with `snake_case` keys, `slog.SetDefault` only in `main()`. This is the Go instantiation of
> [[logging]]'s neutral contract.

Applies to: all production logging in a Go project. Canonical reference: `log/slog` directly, no
wrapper logger.

{{define "when"}}
- Adding any log statement in Go production code.
- Choosing a logger source (global vs injected) in Go.
{{end}}

{{define "must"}}
1. Use `log/slog` only. Convenience funcs (`slog.Info`) only at the top of `main()`/boot; inside
   libraries and handlers take a `*slog.Logger` from the constructor or context (see [[logging]]).
2. Always typed attributes: `slog.String`, `slog.Int`, `slog.Any`, `slog.Group`. Never positional
   `slog.Info("msg", "key", v)` (sloglint rejects it).
3. Log an error exactly once, at the top of the call chain: `slog.Error("msg", slog.String("error",
   err.Error()))`.
4. Lowercase, short, no-trailing-punctuation messages; `snake_case` attribute keys
   (`request_id`, `user_id`).
5. Set `slog.SetDefault` in `main()` before any other call; never in a library.
{{end}}

{{define "forbidden"}}
- `fmt.Println`/`log.Printf`/third-party loggers in production code.
- Positional slog args; a global logger inside libraries instead of an injected/context logger.
- `slog.SetDefault` in a library.
{{end}}

{{define "validation"}}
- [ ] Logging uses `slog` typed attributes only; no positional kv; no `fmt.Println`.
- [ ] Messages lowercase, no trailing punctuation; keys `snake_case`.
- [ ] Libraries take a `*slog.Logger`; `slog.SetDefault` set in `main()` first.
{{end}}
