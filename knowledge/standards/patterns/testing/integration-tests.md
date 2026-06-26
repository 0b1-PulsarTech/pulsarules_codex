---
id: integration-tests
name: Integration tests & E2E harness
description: Repo tests against a real DB via a test factory behind //go:build integration; E2E in a separate module boots the real server via an engine factory + TestMain; gRPC on bufconn; no time.Sleep.
tags:
    - go
    - testing
dependencies:
    - testing
---

# Integration tests & E2E harness

> Repository tests against a real DB via a test factory behind `//go:build integration` (never a SQL
> mock); E2E tests in a separate `test/integration/` module that boots the real server via a test
> engine factory + `TestMain`; gRPC E2E on `bufconn`; no `time.Sleep`; per-test DB isolation via
> `t.Cleanup`.

Reference tools: a DB test factory; an E2E engine factory; `bufconn`.

{{define "when"}}
- Writing repository tests against a real database.
- Writing end-to-end tests that exercise the booted HTTP/gRPC server.
- Building fixtures or a test harness.
{{end}}

{{define "recipe"}}
Repository test (real DB, same package, integration tag):

```go
//go:build integration

var factory *dbtest.Factory

func TestMain(m *testing.M) {
    ctx := context.Background()
    factory = dbtest.NewFactory(ctx)
    os.Exit(m.Run())
}

func TestRepo_InsertAndGet(t *testing.T) {
    t.Parallel()
    db := factory.NewDB(t)

    repo := thingrepo.New(db)
    got, err := repo.Insert(context.Background(), entities.Thing{Name: "x"})
    if err != nil {
        t.Fatalf("insert: %v", err)
    }
}
```

E2E harness (`test/integration/`):

```go
var engines *apptest.EngineFactory

func TestMain(m *testing.M) {
    engines = apptest.NewEngineFactory(
        apptest.WithConnectionFactory(dbtest.NewFactory(context.Background())),
        apptest.WithMigrationRunner(migrations.VersionedMigrationsFS()),
        apptest.WithTestServerFromEngine(withTestServer),
    )
    os.Exit(m.Run())
}

func withTestServer(engine *apptest.Engine) (apptest.Runner, error) {
    inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
    remy.RegisterInstance(inj, engine.DB())
    bootstrap.DoInjections(inj, appconf.Config{...})
    return apptest.NewHTTPRunner(engine.DB(), bootstrap.NewWebServer(...)), nil
}

func TestCreateLead_E2E(t *testing.T) {
    engine := engines.NewEngine(t)
}
```

gRPC E2E: boot the gRPC server on a `bufconn` listener alongside HTTP; share auth via a shared
runner context.
{{end}}

{{define "forbidden"}}
- A SQL mock or in-memory `database/sql` faker.
- Black-box `_test` packages.
- `time.Sleep` to wait for readiness; the engine factory returns only when ready.
- Cross-test shared state via globals; missing `t.Cleanup` for started goroutines.
{{end}}

{{define "validation"}}
- [ ] Repo tests use a real-DB factory + `//go:build integration`; no SQL mock.
- [ ] E2E uses the engine factory + `TestMain`; `NewEngine(t)` gives a fresh DB + server.
- [ ] gRPC E2E on `bufconn`; auth shared via runner context.
- [ ] No `time.Sleep`; per-test DB isolation; `t.Cleanup` for goroutines.
{{end}}
