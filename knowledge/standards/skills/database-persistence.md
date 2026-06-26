---
id: database-persistence
name: Database persistence
---

## Mandatory workflow

1. Author schema as `type X struct{ ent.Schema }` with `Fields()`, `Edges()`, `Indexes()`, and `Mixin()` (UUID,
   Timestamp, SoftDelete). Do NOT add a `tenant_id` field when isolation is connection-level. ent is authoring-only;
   never import the ent client at runtime.
2. Generate migrations with Atlas (ent is the input): diff, review the `.sql` (add a RENAME step for column renames),
   refresh `atlas.sum`, apply locally. Never edit deployed migrations or hand-edit the sum file.
3. Write one query per `-- name: QueryName :one|:many|:exec|:execrows` header in `.sql`, parameterized with `?`. IDs are
   `binary(16)` overridden to a hex-ID type; generate UUIDv7 in Go (`ids.New()`) when the DB has no native generator.
   Regenerate the typed query code; never edit it by hand.
4. Build the repository struct holding an unexported `*dbgen.Queries` (`New(db)`) plus a goverter converter; assert
   `var _ <feature>.Repository = (*Repo)(nil)`.
5. Hand-write the goverter converter interface (`//goverter:converter`, output directives, `ToEntity`/`FromEntity`,
   `//goverter:map`/`//goverter:ignore` for custom/ignored fields, `//go:generate goverter gen .`); regenerate the impl,
   never edit it.
6. Convert rows to domain DTOs in every exported repository method before returning; wrap errors with a DB error
   mapper (not-found / constraint errors; a duplicate-entry helper for idempotent writers). No generated row type
   escapes a public method.
7. Take the `*sql.DB` from the tenant DB resolver when multi-tenant; query logic is identical regardless, so no
   `tenant_id` predicate appears in any `.sql`. A single write needs no explicit tx; multi-write goes
   through [[transactions]].
8. Register both the concrete and the interface binding in the repo `di.go` (the interface binding is what use cases
   consume); keep any dialect switch in bootstrap. Test against a real DB via a test factory behind
   `//go:build integration`, never a SQL mock.

## Validation checklist

- [ ] Entity uses the mixins package; no `tenant_id` field/mixin on new tables (connection-level isolation).
- [ ] Migration generated via Atlas tasks; `atlas.sum` refreshed, not hand-edited; RENAME steps added for renames.
- [ ] Query files have proper `-- name: ... :kind` headers; parameterized with `?`; binary(16) IDs overridden.
- [ ] Generated query and mapper code regenerated, never hand-edited.
- [ ] Repo holds unexported `*dbgen.Queries`; asserts the consumer interface; no generated type escapes a public method.
- [ ] SQL errors wrapped via the DB error mapper.
- [ ] No `tenant_id` predicate in any query; repo tests use a real-DB factory with the integration tag.

## Forbidden actions

- A `tenant_id` column/mixin or `WHERE tenant_id = ?` on new tables (when isolation is connection-level).
- Editing generated query code, generated mapper code, deployed migrations, or the migration sum.
- Returning generated row types from an exported repository method.
- `*sql.DB.QueryContext` or string-built SQL outside the generated layer (gosec G201); importing generated query code
  outside `internal/infra/repositories/`.
- A SQL mock or any in-memory `database/sql` faker.
- Cross-feature joins into another use case's tables (call via a facade); importing the ent client at runtime.

## Expected outputs

- An ent schema with mixins; Atlas migrations with refreshed sums; typed `sqlc` queries.
- A repository that converts rows to domain DTOs via goverter and wraps errors with the DB error mapper.
- Per-dialect `di.go` registering concrete + interface bindings; real-DB integration tests.
