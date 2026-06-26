---
id: testing
name: Testing
description: Colocated same-package tests (no black-box _test packages), one _test.go per source file, table-driven with slice testCases / row testCase (never tt) covering success AND every failure path, deterministic time via testing/synctest (never a now func() clock-injection field, never time.Sleep), repository tests against a real DB behind //go:build integration (never a SQL mock), E2E boots the real server via a test engine factory + TestMain, mockgen mocks.
tags:
    - go
    - testing
---

# Testing

> Colocated same-package tests (no black-box `_test` packages), one `_test.go` per source file,
> table-driven with `testCases`/`testCase` covering success and every failure path, deterministic
> time via `testing/synctest` (never a `now func()` clock-injection field, never `time.Sleep`),
> repository tests against a real DB behind `//go:build integration` (never a SQL mock), E2E via a
> test engine factory + `TestMain`, `mockgen` mocks.

Applies to: all test code.

{{define "when"}}
- Writing unit tests for business logic.
- Writing repository tests against a real database.
- Writing end-to-end tests that exercise the booted server.
- Building fixtures or mocking a dependency.
{{end}}

{{define "must"}}
1. Tests live in the SAME package as the code under test (no black-box `<pkg>_test`), colocated as
   `foo_test.go` next to `foo.go`, so they can reach unexported helpers. One `_test.go` per source
   file, name-matched; shared fakes/helpers live in a `<pkg>_test.go` named after the package.
2. Unit tests do no I/O (no DB, network, disk, real sleeps). Depend through an interface and pass a
   fake; generate mocks with `//go:generate mockgen` to `_mock_test.go` (same package). Hand-roll a
   mock only for <= 2 methods (see colocated-mocks pattern). Do NOT copy a multi-method fake/builder
   into every test package: put a SHARED fixture builder/fake in a `<pkg>test` helper package
   (e.g. `webwraptest.NewReader(ctx).WithQuery(k, v).WithBody(x)`) imported by the tests that need it;
   the >2-methods -> `mockgen` rule still governs behavioral dependency mocks. Place every
   `//go:generate` directive AFTER the import block, never before `package`.
3. UNIT-TEST SHAPE: table-driven by default. The slice is `testCases`, the row variable is `testCase`
   (NEVER `tt`/`tests`); fields are `name`, input, `want`, `wantErr`. Cover the success path AND every
   failure/error branch (`errors.Is`/`errors.As` for sentinels). Name locals descriptively - never
   bare `i`/`n`/`e`. `t.Parallel()` on the outer test and each inner `t.Run`; helpers call
   `t.Helper()` first and accept `testing.TB`.
4. DETERMINISTIC TIME: code under test never reads the wall clock or sleeps just so a test can pass,
   and you NEVER add a `now func() time.Time` (or similar) field merely to seam time in tests. Drive
   timing-dependent behaviour (TTL/expiry, backoff, debounce, ticker, timeout) with `testing/synctest`
   (stable in Go 1.25; target 1.26): run the body in `synctest.Test(t, func(t *testing.T){ ... })`,
   where the bubble's fake clock advances `time.Now`/`time.Sleep`/timers with no real wait and
   `synctest.Wait()` blocks until goroutines settle. Domain/use-case/worker code that needs richer
   time takes a small injected `Clock` interface (`Now`/`Since`/`After`/`NewTicker`), never direct
   `time.*`.
5. Repository tests are real-DB round-trips via a test factory, guarded by `//go:build integration`,
   colocated in the repo package. `TestMain` builds the factory once; each test gets a fresh DB via
   `factory.NewDB(t)` (auto `t.Cleanup`). Never a SQL mock.
6. End-to-end tests live under a separate `test/integration/` module. `TestMain` builds a test engine
   factory with a connection factory, a migration runner, and a test-server builder that wires the
   injector + bootstrap and builds the server. Each test calls `NewEngine(t)` for a fresh DB + server.
7. gRPC E2E: boot the gRPC server on a `bufconn` listener alongside HTTP; share auth via a shared
   runner context.
8. Use immutable, chainable fixture builders and composed fixture sets.
9. Never `time.Sleep` to wait for readiness (the engine factory returns only when ready); every test
   isolates its own DB; stop goroutines via `t.Cleanup`. Run `go test -tags=integration ./...` for
   repo/E2E.
10. CONCURRENCY / RACE tests need a release barrier or they are vacuous. A test asserting
    race-freedom or at-most-once semantics MUST gate every goroutine on a shared unbuffered `start`
    channel and `close(start)` to release them simultaneously, so they genuinely contend. Without
    the barrier the goroutines can run serially and the test passes even against broken code. Assert
    the observable invariant (exactly one winner, no double effect) and run these under
    `go test -race`.
{{end}}

{{define "examples"}}
Table-driven unit test (slice `testCases`, row `testCase`, `t.Parallel()` outer + inner, success and
every failure branch):

```go
func TestParseStatus(t *testing.T) {
    t.Parallel()

    testCases := []struct {
        name    string
        input   string
        want    Status
        wantErr error
    }{
        {"draft", "draft", StatusDraft, nil},
        {"unknown", "xyz", "", ErrUnknownStatus},
    }
    for _, testCase := range testCases {
        t.Run(testCase.name, func(t *testing.T) {
            t.Parallel()
            got, err := ParseStatus(testCase.input)
            if !errors.Is(err, testCase.wantErr) {
                t.Fatalf("err = %v, want %v", err, testCase.wantErr)
            }
            if got != testCase.want {
                t.Fatalf("got = %v, want %v", got, testCase.want)
            }
        })
    }
}
```

Deterministic time with `testing/synctest` (a fake clock advances only when every goroutine is
blocked; no real sleep, no `now func()` field):

```go
synctest.Test(t, func(t *testing.T) {
    d := NewDebouncer(clock, quiet) // clock is the bubble's fake
    d.Push(ctx, "a")
    d.Push(ctx, "b")
    synctest.Wait() // let the quiet window elapse in fake time
    // assert exactly one coalesced fire
})
```
{{end}}

{{define "forbidden"}}
- Black-box `<pkg>_test` packages.
- Naming the table slice/row `tests`/`tt` (use `testCases`/`testCase`); bare loop-index locals
  (`i`/`n`/`e`).
- A `now func() time.Time` (or other clock-injection) field added just to seam time - drive a fake
  clock with `testing/synctest` instead.
- A SQL mock or any in-memory `database/sql` faker.
- Copy-pasting a multi-method fake/builder into each test package instead of a shared `<pkg>test` helper.
- A `//go:generate` directive placed before `package` (it goes after the import block).
- `time.Sleep` in tests; `os.Exit` in tests.
- A race / at-most-once test with no `close(start)` release barrier - it can pass against broken code.
- Package-level `var db = ...`; cross-test shared state via globals.
- Tests touching real external systems without isolation (use interfaces + fakes).
{{end}}

{{define "validation"}}
- [ ] Tests are same-package and colocated; no black-box `_test` packages; one `_test.go` per source
  file.
- [ ] Unit tests do no I/O; dependencies faked via interfaces/`mockgen`.
- [ ] Table-driven: slice `testCases`, row `testCase` (not `tt`/`tests`); success AND every failure
  path covered; `t.Parallel()` outer + inner; helpers use `t.Helper()`; descriptive locals.
- [ ] Timing-dependent behaviour tested via `testing/synctest` (no `time.Sleep`, no `now func()`
  clock-injection field); richer time needs an injected `Clock`.
- [ ] Repo tests use a real-DB factory + `//go:build integration`; no SQL mock.
- [ ] E2E uses the engine factory + `TestMain`; gRPC on `bufconn`.
- [ ] Fixtures used; per-test DB isolation; `t.Cleanup` for goroutines.
{{end}}
