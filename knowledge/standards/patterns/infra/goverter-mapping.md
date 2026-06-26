---
id: goverter-mapping
name: goverter mapping
description: Convert generated rows to domain DTOs with a compile-time-checked goverter converter; no generated row type escapes a repository.
tags:
    - go
    - database
dependencies:
    - database
---

# goverter mapping

> Convert generated rows to domain DTOs with a compile-time-checked goverter converter. Hand-write
> the converter interface; the implementation is generated. No generated row type escapes a
> repository.

Reference tools: `github.com/jmattheis/goverter`.

{{define "when"}}
- Mapping generated rows to domain entities (and back).
- Avoiding forgotten-field bugs at the persistence boundary.
{{end}}

{{define "recipe"}}
```go
// internal/mappers/converter.go
//go:generate goverter gen .
//
//goverter:converter
//goverter:output:file converter_gen.go
//goverter:output:package mappers
type Converter interface {
    ToEntity(row dbgen.Thing) entities.Thing
    FromEntity(l entities.Thing) dbgen.InsertThingParams

    //goverter:map Stage Stage // field name differs between row and entity
    //goverter:ignore InternalNote // row-only column not surfaced to the domain
}
```

Regenerate:

```sh
task -d apps/<name> mappers   # produces converter_gen.go
```

Use inside the repository:

```go
func (r *Repo) Get(ctx context.Context, id entities.ID) (entities.Thing, error) {
    row, err := r.q.GetThing(ctx, id)
    if err != nil {
        return entities.Thing{}, dberr.WrapSQLError(err, "thing", id)
    }
    return r.cvt.ToEntity(row), nil
}
```
{{end}}

{{define "forbidden"}}
- Editing `converter_gen.go` by hand.
- Returning `dbgen.*` row types from an exported repository method.
- Hand-rolling mapping that goverter can generate (forgotten-field risk).
{{end}}

{{define "validation"}}
- [ ] Converter interface hand-written with `//goverter:*` directives; impl generated.
- [ ] `converter_gen.go` regenerated via the mappers task, never hand-edited.
- [ ] Every exported repository method converts rows to domain DTOs before returning.
{{end}}
