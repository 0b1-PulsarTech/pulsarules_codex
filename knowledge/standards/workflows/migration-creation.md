---
id: migration-creation
name: Migration creation
description: Change the schema, generate the migration, regenerate typed queries + mappers, update the repository, add an integration test.
tags:
    - go
    - workflow
    - database
steps:
    - author/edit the ent schema (no tenant_id when connection-level)
    - generate the migration via Atlas; review the SQL (RENAME for column renames)
    - refresh the migration hash; apply locally
    - author/edit sqlc queries (one per -- name header, parameterized)
    - regenerate typed query code (never hand-edit)
    - update the goverter converter; regenerate the mapper
    - update the repository (convert rows to DTOs, wrap errors, assert the port)
    - add an integration test (real DB, integration tag)
    - gate and commit
composes_rules:
    - database
    - testing
    - commits
composes_patterns:
    - ent-schema
    - sqlc-queries
    - goverter-mapping
    - repository-layout
    - integration-tests
---

# Migration creation workflow

> Change the schema, generate the migration, regenerate typed queries + mappers, update the
> repository, add an integration test.

## When to use

- Adding/changing a database entity (table, column, index, edge).

## Steps

1. **Author/edit the schema** (`database/entschema/<entity>.go` or equivalent): `Fields()`,
   `Edges()`, `Indexes()`, `Mixin()`. No `tenant_id` when isolation is connection-level.
   (pattern: [[ent-schema]], rule: [[database]])
2. **Generate the migration** via Atlas: `task -d database migration:gen DIFF_NAME=<name>`. Review
   the `.sql` (add a RENAME step for column renames). (pattern: [[ent-schema]])
3. **Refresh the migration hash:** `task -d database migration:hash` (never hand-edit the sum).
4. **Apply locally:** `task -d database migration:apply`.
5. **Author/edit queries** (`database/queries/<feature>.sql`): one query per
   `-- name: ... :kind` header; parameterized; binary(16) IDs overridden. (pattern: [[sqlc-queries]])
6. **Regenerate typed query code:** `task -d database sqlc`. Never edit it by hand.
7. **Update the goverter converter** if columns/fields changed; regenerate the mapper:
   `task -d apps/<name> mappers`. (pattern: [[goverter-mapping]])
8. **Update the repository** (`internal/infra/repositories/<dialect>/<feature>repo/`): convert rows
   to domain DTOs; wrap errors with the DB error mapper; assert `var _ <feature>.Repository =
   (*Repo)(nil)`. (pattern: [[repository-layout]])
9. **Add an integration test** (real DB, `//go:build integration`): cover the round-trip and one
   edge case. (rule: [[testing]], pattern: [[integration-tests]])
10. **Gate & commit:** `task tools:lint`, `task main:test` (with `-tags=integration` for the repo
    test); commit with `:sparkles: feat(<feature>): add <entity> migration` or similar.

## References

- rules: [[database]], [[testing]]
- patterns: [[ent-schema]], [[sqlc-queries]], [[goverter-mapping]], [[repository-layout]],
  [[integration-tests]]
