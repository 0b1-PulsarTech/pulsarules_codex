---
id: concurrency
name: Concurrency
---

## Mandatory workflow

1. Give every goroutine exactly one owner that knows when it stops (`WaitGroup`/`errgroup`/result channel or a
   supervisor). No fire-and-forget.
2. Make `ctx context.Context` the first parameter of every blocking/I/O function. Never store a `Context` on a struct
   field.
3. Propagate the inbound context; never pass `context.Background()` from a request handler. Use `context.WithCancel`/
   `WithTimeout` and always `defer cancel()`.
4. Fan out with `errgroup.WithContext(ctx)` + `g.Go(func() error {...})`; it cancels on first error and waits for the
   rest.
5. Long-running workers: `for { select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: ... } }`. Pace with a
   ticker, never `time.Sleep`. Inject the clock as an interface so timing is testable under `testing/synctest`.
6. Document who closes each channel (sender closes; receiver never). Read with the two-value form `v, ok := <-ch`.
   Default to unbuffered channels.
7. Embed `sync.Mutex` as an unexported field; lock the smallest scope; never call a user callback while holding the
   lock. Use `sync.RWMutex` only when reads vastly outnumber writes.
8. Restart a supervised worker loop on panic at its parent (the only sanctioned `recover()` site).
9. Run `go test -race ./...`; CI runs with `-race`. Any race is a bug.
10. To cancel work that must outlive its request, keep a `map[key]context.CancelFunc` owned by the worker; store the
    `CancelFunc`, NEVER the `ctx`.

## Validation checklist

- [ ] Every goroutine has a known owner and stop condition.
- [ ] `ctx` is the first param of blocking calls; not stored on structs.
- [ ] Inbound context propagated; `WithCancel`/`WithTimeout` paired with `defer cancel()`.
- [ ] Fan-out uses `errgroup`/`WaitGroup`, not hand-rolled channel coordination.
- [ ] Worker loops select on `ctx.Done()` with a ticker; no `time.Sleep` pacing.
- [ ] Channel ownership documented; two-value receive used.
- [ ] Tests pass under `-race`.
- [ ] Work that outlives its request is cancelled via a stored `CancelFunc`, never a stored `ctx`.

## Forbidden actions

- `go f()` with no owner/stop tracking (leak).
- `time.Sleep` to wait for an event in production code.
- Storing `Context` on a struct; passing `context.Background()` from a handler.
- Storing a `ctx` (on a struct or in a map) to cancel later; store the `CancelFunc`.
- Omitting `defer cancel()` for `WithCancel`/`WithTimeout`; `select{}` to block forever.
- Silencing the race detector; shipping any `-race` finding.

## Expected outputs

- Owned, cancellable goroutines that propagate context and stop on `ctx.Done()`.
- Fan-out via `errgroup`; worker loops paced by an injected clock; no leaks.
- A green `go test -race ./...` run.
