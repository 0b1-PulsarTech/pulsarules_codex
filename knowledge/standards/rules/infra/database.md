---
id: database
name: Database
description: ent schema -> Atlas migrations -> sqlc typed queries -> repositories that convert rows to domain DTOs via goverter; no raw SQL in handlers; no tenant_id predicate when isolation is connection-level.
references:
    - ent
    - atlas
    - sqlc
    - goverter
tags:
    - go
    - database
dependencies:
    - errors
    - code-placement
linters:
    - gosec
    - depguard
    - sqlc
analyzers:
    - golangci-lint
---

# Database

> Schema-as-code (`ent`) -> Atlas migrations -> `sqlc` typed queries -> repositories that convert
> rows to domain DTOs via `goverter` (no generated row types escape) -> errors mapped by a DB error
> wrapper. No raw SQL in handlers/use cases; no `tenant_id` predicate when isolation is
> connection-level.

Applies to: persistence. Canonical reference stack: `entgo.io/ent`, `atlasgo.io`, `sqlc.dev`,
`github.com/jmattheis/goverter`.

{{define "when"}}
- Adding or modifying a database entity (table).
- Writing a `.sql` query or generating migrations.
- Creating or editing a repository.
- Mapping generated rows to domain entities.
{{end}}

{{define "must"}}
1. Schema: author entities as `type X struct{ ent.Schema }` with `Fields()`, `Edges()`, `Indexes()`,
   and `Mixin()` (UUID, Timestamp, SoftDelete). Do NOT add a `tenant_id` field when isolation is
   connection-level (a tenant DB resolver).
2. Migrations (Atlas, ent is input): generate the diff, review the `.sql` (add a RENAME step for
   column renames), refresh the migration hash, then apply locally. Never edit deployed migrations
   or hand-edit the migration sum file.
3. Queries: write one query per `-- name: QueryName :one|:many|:exec|:execrows` header, parameterized
   with `?`. IDs are binary(16) overridden to a hex-ID type; generate UUIDv7 in Go when the DB has no
   native generator.
4. Regenerate the typed query code; never edit it by hand.
5. Repository: hold an unexported `*Queries` (`New(db)`) and a converter. Assert
   `var _ <feature>.Repository = (*Repo)(nil)`.
6. Mapping (goverter): hand-write the converter interface with `//goverter:converter`, output
   directives, and `ToEntity`/`FromEntity` methods (use `//goverter:map`/`//goverter:ignore` for
   custom/ignored fields and a `//go:generate goverter gen .`). Regenerate the impl; never edit it.
7. Every exported repository method converts rows to domain DTOs before returning; wrap errors with
   a DB error mapper (yields not-found / constraint errors; a duplicate-entry helper for idempotent
   writers).
8. The `*sql.DB` comes from the tenant DB resolver when multi-tenant; query logic is identical
   regardless, so no `tenant_id` predicate appears in any `.sql`. Single write needs no explicit tx;
   multi-write goes through [[transactions]].
9. Tests: real DB via a test factory, `//go:build integration`, same package - never a SQL mock.
10. A repository method serving a list MUST batch into one query (`IN (...)`) rather than a use case
    looping per-entity repo calls (N+1). Any request-scoped cache stays request-scoped - never a
    package-level cache, which leaks data across requests and users.
11. NULL stops at the repository. A nullable column maps to a value-typed field (ent `Optional()`
    without `.Nillable()`); reach for `.Nillable()` or `sql.Null*` only where NULL means something
    the zero value does not, and convert it before returning - no `*string`/`sql.Null*` reaches a
    use case or entity. This is [[effective-go]]'s zero-value-over-pointer rule at the one boundary
    that genuinely produces nulls.
{{end}}

{{define "forbidden"}}
- A `tenant_id` column/mixin or `WHERE tenant_id = ?` on new tables (when isolation is
  connection-level).
- Editing generated query code, generated mapper code, deployed migrations, or the migration sum.
- Returning generated row types from an exported repository method.
- `*sql.DB.QueryContext` or string-built SQL outside the generated layer (gosec G201); importing
  generated query code outside `internal/infra/repositories/`.
- A SQL mock or any in-memory `database/sql` faker.
- Cross-feature joins into another use case's tables (call via a facade).
- Importing the ent client at runtime (schema is authoring-only).
- A use case looping per-entity repo calls where a batched `IN (...)` query would serve the list.
- A package-level or process-global cache for request-scoped data.
- A `.Nillable()` field or `sql.Null*` value escaping the repository into a use case or entity.
{{end}}

{{define "validation"}}
- [ ] Entity uses mixins; no `tenant_id` field/mixin on new tables (connection-level isolation).
- [ ] Migration generated via Atlas tasks; sum refreshed, not hand-edited.
- [ ] Query files have proper `-- name: ... :kind` headers; binary(16) IDs overridden.
- [ ] Generated code regenerated, never hand-edited.
- [ ] Repository holds unexported `*Queries`; no generated type escapes a public method.
- [ ] SQL errors wrapped via the DB error mapper; repo asserts the consumer interface.
- [ ] No `tenant_id` predicate in any query; repo tests use a real-DB factory with the integration
  tag.
- [ ] List-serving repository methods batch into one `IN (...)` query; no per-entity-call loop (N+1).
- [ ] Any request-scoped cache stays request-scoped; no package-level cache.
{{end}}
