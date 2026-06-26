---
id: sqlc-queries
name: sqlc query authoring
description: .sql query files with one query per -- name header, parameterized; binary(16) IDs overridden; regenerate typed query code; never edit by hand.
tags:
    - go
    - database
dependencies:
    - database
---

# sqlc query authoring

> Write `.sql` query files with one query per `-- name: QueryName :kind` header, parameterized with
> `?`; binary(16) IDs overridden to a hex-ID type; regenerate the typed query code; never edit it by
> hand.

Reference tools: `sqlc.dev`.

{{define "when"}}
- Writing or editing a `.sql` query.
- Regenerating typed query code.
{{end}}

{{define "recipe"}}
```sql
-- database/queries/thing.sql

-- name: InsertThing :one
INSERT INTO things (id, name, owner, stage, created_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListThingsByOwner :many
SELECT * FROM things
WHERE owner = ? AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: UpdateThingStage :execrows
UPDATE things SET stage = ? WHERE id = ?;
```

`sqlc.yaml`:

```yaml
version: "2"
sql:
  - engine: mysql
    schema: build/migrations
    queries: database/queries
    gen:
      go:
        package: dbgen
        out: internal/infra/dbgen
        overrides:
          - column: "things.id"
            go_type: "ids.HexID"
```

Regenerate:

```sh
task -d database sqlc   # writes internal/infra/dbgen/
```

Generate UUIDv7 in Go (`ids.New()`) when the DB has no native generator. IDs are `binary(16)`
overridden to `ids.HexID`/`NullHexID`.
{{end}}

{{define "forbidden"}}
- Editing generated query code by hand.
- String-built SQL; `*sql.DB.QueryContext` outside the generated layer.
- A `tenant_id` predicate in any query (connection-level isolation).
- Cross-feature joins into another use case's tables.
{{end}}

{{define "validation"}}
- [ ] Query files have proper `-- name: ... :kind` headers; parameterized with `?`.
- [ ] binary(16) IDs overridden to the hex-ID type.
- [ ] Generated code regenerated via the sqlc task; never hand-edited.
- [ ] No `tenant_id` predicate.
{{end}}
