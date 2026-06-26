---
id: repository-layout
name: Repository layout
description: sqlc + a repo struct holding an unexported *Queries + a goverter converter; per-dialect packages register both concrete and interface bindings; rows converted to domain DTOs; errors wrapped by a DB error mapper.
tags:
    - go
    - database
dependencies:
    - database
composes:
    - sqlc-queries
    - goverter-mapping
---

# Repository layout

> sqlc + a repo struct holding an unexported `*Queries` + a goverter converter; per-dialect packages
> register both the concrete and the interface binding the use case consumes; rows converted to
> domain DTOs before returning; errors wrapped by a DB error mapper.

Reference tools: `sqlc`, `goverter`, a DB error wrapper.

{{define "when"}}
- Creating or editing a repository.
- Mapping generated rows to domain entities.
- Wiring a per-dialect repo package.
{{end}}

{{define "recipe"}}
Per dialect (`internal/infra/repositories/<dialect>/<feature>repo/`):

```go
package thingrepo

type Repo struct {
    queries *dbgen.Queries
    cvt     mappers.Converter
}

func New(db *sql.DB) *Repo {
    return &Repo{queries: dbgen.New(db), cvt: mappers.Converter{}}
}

var _ thing.Repository = (*Repo)(nil) // compile-time port assertion

func (repo *Repo) Insert(ctx context.Context, entity entities.Thing) (entities.Thing, error) {
    row, err := repo.queries.InsertThing(ctx, dbgen.InsertThingParams{
        ID:    ids.New(),
        Name:  entity.Name,
        Owner: entity.Owner,
    })
    if err != nil {
        return entities.Thing{}, dberr.WrapSQLError(err, "thing", entity.ID)
    }
    return repo.cvt.ToEntity(row), nil
}
```

`di.go` registers both bindings (the interface binding is what use cases consume):

```go
func Register(inj remy.Injector, db *sql.DB) {
    remy.RegisterConstructor(inj, remy.Singleton[*dbgen.Queries],
        func() *dbgen.Queries { return dbgen.New(db) })
    remy.RegisterConstructorArgs1(inj, remy.Factory[*Repo], New)
    remy.RegisterConstructorArgs1(inj, remy.Factory[thing.Repository],
        func(repo *Repo) thing.Repository { return repo })
}
```

goverter converter (hand-written; impl is generated):

```go
// internal/mappers/converter.go
//go:generate goverter gen .
//
//goverter:converter
//goverter:output:file converter_gen.go
type Converter interface {
    ToEntity(row dbgen.Thing) entities.Thing
    FromEntity(thing entities.Thing) dbgen.InsertThingParams
}
```
{{end}}

{{define "forbidden"}}
- Returning `dbgen.*` row types from an exported repository method.
- Importing `dbgen` outside `internal/infra/repositories/`.
- String-built SQL; a `switch dialect` inside this package (bootstrap owns it).
- Editing generated query or mapper code by hand.
{{end}}

{{define "validation"}}
- [ ] Repo holds unexported `*dbgen.Queries` + converter; asserts the consumer interface.
- [ ] Every exported method converts rows to domain DTOs before returning.
- [ ] SQL errors wrapped via the DB error mapper.
- [ ] `di.go` registers concrete + interface bindings; no dialect switch here.
- [ ] Converter interface hand-written; impl generated, not hand-edited.
{{end}}
