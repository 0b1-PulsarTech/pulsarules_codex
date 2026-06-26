---
id: config
name: Config
description: Typed Config loaded once from TOML + env via a config loader; env names are package-private constants; no os.Getenv outside config; secrets zeroed after load.
tags:
    - go
    - config
dependencies:
    - startup
linters:
    - forbidigo
    - depguard
---

# Config

> Typed `Config` loaded once from TOML + env via a config loader; env names are package-private
> constants; no `os.Getenv` outside the config package; secrets zeroed after load.

Applies to: application configuration. Canonical reference tools: `BurntSushi/toml` + a typed env
binder (source repos use `confloader`).

{{define "when"}}
- Adding a config field.
- Loading configuration at boot.
- Binding an env variable to a typed field.
{{end}}

{{define "must"}}
1. Define a typed `Config` struct (in `internal/bootstrap/config.go` or a `config/` package); shared
   cross-app types (e.g. `Database`, `Mailer`) in a shared infra-config module.
2. Load once at boot: `configload.Load(filename, defaults, updater)`. Treat `Config` as read-only
   after boot.
3. Bind env via the loader's `BindEnv`/`BindField` using package-private name constants; never
   `os.Getenv` outside the config package.
4. Zero secret source fields after registering them into the injector so cleartext does not linger.
5. Provide a `conf.example.toml`/`conf.toml.example`; keep real `.env`/`conf.toml` local-only and
   gitignored.
6. The bootstrap is the only place that switches on a config-driven selector (e.g. `conf.DB.Driver`,
   `conf.Providers[].Code`); no `switch driver` inside a repository's `di.go`.
{{end}}

{{define "forbidden"}}
- `os.Getenv` outside the config package.
- Mutating `Config` after boot.
- Committing real `conf.toml`/`.env` with secrets.
- A `switch driver`/`switch code` outside the bootstrap switchboard.
{{end}}

{{define "validation"}}
- [ ] `Config` typed, loaded once via the loader; env names are constants; secrets zeroed.
- [ ] No `os.Getenv` outside config; no mutation after boot.
- [ ] `conf.example.toml` provided; real config gitignored.
- [ ] Config-driven impl selection happens only in bootstrap.
{{end}}
