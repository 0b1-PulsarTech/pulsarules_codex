---
id: embedded-migrations
name: Embedded migrations & runner
description: SQL migrations embedded via //go:embed and applied by an in-house runner at startup (gated). Atlas generates files at build time; the runtime applier is the in-house migrator.
tags:
    - go
    - database
dependencies:
    - database
    - startup
---

# Embedded migrations & runner

> SQL migrations are embedded into the binary with `//go:embed` and applied by an in-house runner at
> startup (gated by a flag). Atlas generates the SQL files at build time; the runtime applier is the
> in-house migrator, not Atlas.

Reference tools: Atlas (generation); `embed`; an in-house `migrator` with a dialect abstraction.

{{define "when"}}
- Embedding SQL migrations into the binary.
- Applying migrations at startup.
{{end}}

{{define "recipe"}}
```go
// build/migrations/embed_migrations.go
package migrations

import "embed"

//go:embed *.sql
var fs embed.FS

func VersionedMigrationsFS() fs.FS { return fs }
```

The runner (dialect-abstracted):

```go
// internal/migrator/migrator.go
type Dialect interface {
    Placeholder() string
    CurrentRevision(ctx context.Context, db *sql.DB) (string, error)
    Apply(ctx context.Context, db *sql.DB, name, stmt string) error
}

type Migrator struct {
    fs      fs.FS
    dialect Dialect
}

func (m Migrator) Apply(ctx context.Context, db *sql.DB) error {
    // read *.sql in order; skip already-applied (CurrentRevision); apply each; record revision
}
```

Gate at startup (migrations are a side effect; only run when asked):

```go
if conf.RunMigrations {
    if err := migrator.New(migrations.VersionedMigrationsFS(), mysqlDialect{}).Apply(ctx, db); err != nil {
        return fmt.Errorf("apply migrations: %w", err)
    }
}
```
{{end}}

{{define "forbidden"}}
- Running Atlas at runtime - it generates files at build time only.
- Applying migrations unconditionally at startup (gate by config/flag).
- Editing applied migration files in place; add a new file instead.
{{end}}

{{define "validation"}}
- [ ] Migrations embedded via `//go:embed`; applied by an in-house runner.
- [ ] Runner dialect-abstracted; migration application gated by config.
- [ ] Atlas used at build time only; applied files never edited in place.
{{end}}
