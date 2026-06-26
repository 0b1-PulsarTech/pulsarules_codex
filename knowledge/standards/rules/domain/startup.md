---
id: startup
name: Startup
description: Zero side effects at package load / in init(); panic only in main(); all I/O and env reads happen inside the bootstrap composition root.
tags:
    - go
    - startup
dependencies:
    - dependency-injection
linters:
    - gochecknoglobals
---

# Startup

> Zero side effects at package load / in `init()`; `panic` only in `main()` boot; all I/O,
> connections, goroutines, and env reads happen inside `main()` -> the bootstrap composition root.

Applies to: process startup and package initialization.

{{define "when"}}
- Writing `main()` or any `internal/bootstrap/` file.
- Adding an `init()` function.
- Deciding where connections, goroutines, or env reads happen.
{{end}}

{{define "must"}}
1. Import-time purity: packages only define symbols. No I/O, connections, goroutines, timers, env
   reads, or registrations in `init()`. Allowed exceptions: driver `_` imports in `main.go` with a
   comment, and compile-time `var _ Iface = (*impl)(nil)` assertions.
2. Keep `main()` thin (~15-20 lines): load config, open the DB, run migrations (if gated), build
   the injector, wire boot, run the server. `panic` only here on impossible boot states.
3. Connections, goroutines, file reads, and env access happen inside `main()` -> the bootstrap
   composition root, never at package load.
4. `slog.SetDefault` is set in `main()` before any other call.
5. No package-level mutable state used as caches, clients, or config (`gochecknoglobals`). Singletons
   live in the injector (see [[dependency-injection]]). Where state is unavoidable, encapsulate it
   behind a constructor/accessor so every write site is discoverable, and prefer returning a new value
   over mutating shared data in place (see [[concurrency]]).
{{end}}

{{define "forbidden"}}
- Side effects in `init()` or at package load.
- `panic` outside `main()` (libraries return errors).
- Package-level mutable state / globals.
- Connections, goroutines, or env reads at package load.
- `slog.SetDefault` in a library.
{{end}}

{{define "validation"}}
- [ ] No side effects at import/`init()`; only driver registration / interface assertions.
- [ ] `main()` is thin and follows the config -> DB -> injector -> migrations -> server sequence.
- [ ] No package-level mutable state; singletons in the injector.
- [ ] `slog.SetDefault` set in `main()` first.
{{end}}
