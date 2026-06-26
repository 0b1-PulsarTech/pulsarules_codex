---
id: integration-tests
name: Integration tests
---

## Mandatory workflow

1. Place tests in the SAME package as the code under test (no black-box `<pkg>_test`), colocated as `foo_test.go` next
   to `foo.go`, so they can reach unexported helpers. One `_test.go` per source file, name-matched; shared fakes live in
   a `<pkg>_test.go` named after the package.
2. Keep unit tests I/O-free (no DB, network, disk, real sleeps). Depend through an interface and pass a fake; generate
   mocks with `//go:generate mockgen` to `_mock_test.go` (same package, directive placed AFTER the import block - never
   before `package`). Hand-roll a mock only for <= 2 methods. Don't copy a multi-method fake/builder into every test
   package: put a SHARED fixture builder/fake in a `<pkg>test` helper package (e.g.
   `webwraptest.NewReader(ctx).WithQuery(k, v).WithBody(x)`); the >2-methods -> `mockgen` rule still governs behavioral
   dependency mocks.
3. Write table-driven tests: the slice is `testCases`, the row is `testCase` (NEVER `tt`/`tests`); fields `name`, input,
   `want`, `wantErr`. Cover the success path AND every failure/error branch (`errors.Is`/`errors.As`). `t.Parallel()` on
   the outer test and each `t.Run`; helpers call `t.Helper()` first and accept `testing.TB`.
4. Drive timing-dependent behaviour with `testing/synctest` (run the body in
   `synctest.Test(t, func(t *testing.T){ ... })`). NEVER add a `now func() time.Time` clock-injection field merely to
   seam time, and never `time.Sleep` to make a test pass. Code needing richer time takes a small injected `Clock` (
   `Now`/`Since`/`After`/`NewTicker`).
5. Test repositories against a real DB via a test factory, guarded by `//go:build integration`, colocated in the repo
   package. `TestMain` builds the factory once; each test gets a fresh DB via `factory.NewDB(t)` (auto `t.Cleanup`).
   Never a SQL mock.
6. Put E2E tests under a separate `test/integration/` module. `TestMain` builds a test engine factory (connection
   factory + migration runner + test-server builder wiring the injector + bootstrap); each test calls `NewEngine(t)` for
   a fresh DB + server.
7. For gRPC E2E, boot the gRPC server on a `bufconn` listener alongside HTTP; share auth via a shared runner context.
8. Use immutable, chainable fixture builders and composed fixture sets. Stop goroutines via `t.Cleanup`. Run
   `go test -tags=integration ./...` for repo/E2E.
9. Concurrency / race tests need a release barrier or they are vacuous: gate every goroutine on a shared unbuffered
   `start` channel and `close(start)` to release them at once so they genuinely contend, then assert the observable
   invariant (exactly one winner, no double effect) under `go test -race`. Without the barrier the goroutines can run
   serially and the test passes against broken code.

## Validation checklist

- [ ] Tests same-package and colocated; no black-box `_test` packages; one `_test.go` per source file.
- [ ] Unit tests do no I/O; dependencies faked via interfaces/`mockgen`.
- [ ] Table-driven: slice `testCases`, row `testCase` (not `tt`/`tests`); success AND every failure path covered;
  `t.Parallel()` outer + inner; helpers use `t.Helper()`; descriptive locals.
- [ ] Timing-dependent behaviour tested via `testing/synctest` (no `time.Sleep`, no `now func()` clock-injection field);
  richer time needs an injected `Clock`.
- [ ] Repo tests use a real-DB factory + `//go:build integration`; no SQL mock.
- [ ] E2E uses the engine factory + `TestMain`; gRPC on `bufconn`.
- [ ] Fixtures used; per-test DB isolation; `t.Cleanup` for goroutines.
- [ ] Concurrency / at-most-once tests gate goroutines on a `close(start)` barrier and run under `-race`.

## Forbidden actions

- Black-box `<pkg>_test` packages; naming the slice/row `tests`/`tt`; bare loop-index locals (`i`/`n`/`e`).
- A `now func() time.Time` clock-injection field added just to seam time; `time.Sleep` in tests.
- A SQL mock or any in-memory `database/sql` faker; `os.Exit` in tests.
- A race / at-most-once test with no `close(start)` release barrier (it can pass against broken code).
- Copy-pasting a multi-method fake/builder into each test package instead of a shared `<pkg>test` helper; a
  `//go:generate` directive placed before `package`.
- Package-level `var db = ...`; cross-test shared state via globals.
- Tests touching real external systems without isolation (use interfaces + fakes).

## Expected outputs

- Colocated, same-package, table-driven tests covering success and every failure path.
- Deterministic timing via `testing/synctest`; repo tests against a real DB behind the integration tag.
- An E2E harness that boots the real server per test on `bufconn`, with per-test DB isolation.
