---
id: config-layout
name: Config layout
description: Typed Config struct loaded from TOML + env; a defaults function; an env updater with private name constants; conf.example.toml committed, real config gitignored.
tags:
    - go
    - config
dependencies:
    - config
---

# Config layout

> Typed `Config` struct loaded from TOML + env; a defaults function; an env updater that binds
> package-private env-name constants to fields; shared cross-app types in a shared infra-config
> module; `conf.example.toml` committed, real config gitignored.

Reference tools: `BurntSushi/toml`; a typed env binder (`confloader`).

{{define "when"}}
- Adding a config field.
- Loading configuration at boot.
- Sharing config types across apps.
{{end}}

{{define "recipe"}}
```go
// internal/bootstrap/config.go
type Config struct {
    HTTP     HTTPConfig
    Database DatabaseConfig
    Tracing  string
    RunMigrations bool
}

type HTTPConfig struct {
    Addr string
}

type DatabaseConfig struct {
    Driver   string // "mysql" | "sqlite"
    DSN      string
    MaxConns int
}
```

Defaults + env updater:

```go
func defaults() Config {
    return Config{
        HTTP:     HTTPConfig{Addr: ":8080"},
        Database: DatabaseConfig{Driver: "mysql", MaxConns: 10},
        Tracing:  "noop",
    }
}

const (
    envHTTPAddr = "APP_HTTP_ADDR"
    envDBDriver = "APP_DB_DRIVER"
    envDBDSN    = "APP_DB_DSN"
)

func envUpdater() configload.Updater {
    return func(c *Config) error {
        configload.BindEnv(c, envHTTPAddr, &c.HTTP.Addr)
        configload.BindEnv(c, envDBDriver, &c.Database.Driver)
        configload.BindEnv(c, envDBDSN, &c.Database.DSN)
        return nil
    }
}
```

Load once:

```go
conf, err := configload.Load("conf.toml", defaults(), envUpdater())
```

`conf.example.toml`:

```toml
http.addr = ":8080"
database.driver = "mysql"
database.dsn = "user:pass@tcp(127.0.0.1:3306)/app"
tracing = "noop"
run_migrations = false
```
{{end}}

{{define "forbidden"}}
- `os.Getenv` outside the config package.
- Mutating `Config` after boot; committing real `conf.toml` with secrets.
- Public env-name constants (package-private).
{{end}}

{{define "validation"}}
- [ ] Typed `Config`; defaults function; env updater with private name constants.
- [ ] Loaded once via the loader; treated read-only after boot.
- [ ] `conf.example.toml` committed; real config gitignored.
{{end}}
