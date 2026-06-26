---
id: ent-schema
name: ent schema authoring
description: Add/edit entities as ent.Schema with Fields/Edges/Indexes/Mixin; no tenant_id when isolation is connection-level; ent is authoring-only.
tags:
    - go
    - database
dependencies:
    - database
---

# ent schema authoring

> Add/edit entities as `type X struct{ ent.Schema }` with `Fields()`, `Edges()`, `Indexes()`, and
> `Mixin()` (UUID, Timestamp, SoftDelete). No `tenant_id` field when isolation is
> connection-level. ent is authoring-only; never import the ent client at runtime.

Reference tools: `entgo.io/ent`, Atlas (migration generation).

{{define "when"}}
- Adding or changing a database entity (table).
- Generating a migration from a schema change.
{{end}}

{{define "recipe"}}
```go
// database/entschema/thing.go
type Thing struct{ ent.Schema }

func (Thing) Fields() []ent.Field {
    return []ent.Field{
        field.String("name").NotEmpty(),
        field.String("owner").Optional(),
        field.Enum("stage").Values("new", "contacted", "won", "lost").Default("new"),
    }
}

func (Thing) Edges() []ent.Edge {
    return []ent.Edge{
        edge.To("events", Event.Type),
    }
}

func (Thing) Indexes() []ent.Index {
    return []ent.Index{
        index.Fields("owner", "stage"),
    }
}

func (Thing) Mixin() []ent.Mixin {
    return []ent.Mixin{
        mixins.UUID{},
        mixins.Timestamp{},
        mixins.SoftDelete{},
    }
}
```

Generate a migration (ent is the input; Atlas produces the SQL):

```sh
task -d database migration:gen DIFF_NAME=add_thing_owner_index
task -d database migration:hash   # refresh atlas.sum; never hand-edit it
task -d database migration:apply
```

For a column rename, add an explicit RENAME step to the generated `.sql` before hashing.
{{end}}

{{define "forbidden"}}
- A `tenant_id` field/mixin on new tables when isolation is connection-level.
- Importing the ent client at runtime; ent runtime hooks/privacy policies.
- Hand-editing deployed migrations or `atlas.sum`.
- Placing schema files in `build/`.
{{end}}

{{define "validation"}}
- [ ] Entity uses the mixins package (UUID/Timestamp/SoftDelete); no `tenant_id` (connection-level).
- [ ] Migration generated via Atlas tasks; `atlas.sum` refreshed, not hand-edited.
- [ ] No ent client import at runtime; schema is authoring-only.
{{end}}
