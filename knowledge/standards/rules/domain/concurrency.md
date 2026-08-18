---
id: concurrency
name: Concurrency
description: Goroutine ownership and context discipline; errgroup/WaitGroup over hand-rolled channels; worker loops on ctx.Done(); -race never silenced.
tags:
    - go
    - concurrency
linters:
    - noctx
    - govet
analyzers:
    - golangci-lint
    - time-discipline
---

# Concurrency

> Every goroutine has one owner; `context.Context` is the first param and never stored on a struct;
> `errgroup`/`WaitGroup` over hand-rolled channels; worker loops select on `ctx.Done()`; the race
> detector is never silenced.

Applies to: any concurrent Go code.

{{define "when"}}
- Spawning a goroutine or fanning out work.
- Working with channels, `sync.Mutex`, or `sync.WaitGroup`.
- Designing a long-running worker/relay loop.
- Propagating or honoring context cancellation/timeouts.
- Starting owned background work that must outlive the request that triggered it.
- Defining or reading a context value key.
- Bounding fan-out concurrency with no per-key ordering requirement.
{{end}}

{{define "must"}}
1. Every goroutine has exactly one owner that knows when it stops (`WaitGroup`/`errgroup`/result
   channel or a supervisor). No fire-and-forget.
2. Make `ctx context.Context` the first parameter of every blocking/I/O function. Never store a
   `Context` on a struct field.
3. Propagate the inbound context; never pass `context.Background()` from a request handler. Use
   `context.WithCancel`/`WithTimeout` and always `defer cancel()`.
4. Fan out with `errgroup.WithContext(ctx)` + `g.Go(func() error {...})`; it cancels on first error
   and waits for the rest.
5. Long-running workers: `for { select { case <-ctx.Done(): return ctx.Err(); case <-ticker.C: ...
   } }`. Pace with a ticker, never `time.Sleep`. Inject the clock as an interface so timing is
   testable under `testing/synctest`.
6. Document who closes each channel (sender closes; receiver never). Read with the two-value form
   `v, ok := <-ch`. Default to unbuffered channels.
7. Embed `sync.Mutex` as an unexported field; lock the smallest scope; never call a user callback
   while holding the lock. Use `sync.RWMutex` only when reads vastly outnumber writes.
8. A supervised worker loop is restarted by its parent on panic. `recover()` is sanctioned at
   exactly two boundaries: this worker-supervisor loop, and transport-level panic-recovery
   middleware/interceptor (see [[rest-adapter]], [[grpc]]) - nowhere else.
9. Run `go test -race ./...`; CI runs with `-race`. Any race is a bug.
10. To cancel work that must outlive its request, keep a `map[key]context.CancelFunc` owned by the
    worker; store the `CancelFunc`, NEVER the `ctx`.
11. Hand off work without blocking the producer via a coalescing NON-BLOCKING signal: a buffered-by-one
    channel with `select { case ch <- struct{}{}: default: }` collapses many wakes into one pending
    wake and never blocks the caller. A consumer that must not block on a full sink drops or persists
    (never grows an unbounded buffer). For durable async dispatch use a bounded, key-sharded worker
    pool (see `[[dispatch-pool]]`): bounded queues are the backpressure, per-key sharding keeps a key
    ordered while different keys parallelize, and the sole sender closes the queues on cancel so
    workers drain in-flight items.
12. For owned background work that must outlive the request that started it, derive the worker's
    context with `context.WithoutCancel(ctx)`. Never `context.Background()` (it loses trace/request
    values carried on `ctx`) and never the live request `ctx` (it gets cancelled when the request
    ends, killing the background work with it).
13. Context value keys MUST be an unexported type (`type ctxKey struct{}` or an unexported named
    type), never a bare `string`/`int`, so keys cannot collide across packages.
14. For simple bounded fan-out with no per-key ordering requirement, cap concurrency with
    `errgroup.SetLimit(n)`. Reach for the heavier `[[dispatch-pool]]` only when work must stay
    ordered per key or needs a durable backlog.
{{end}}

{{define "forbidden"}}
- `go f()` with no owner/stop tracking (leak).
- `time.Sleep` to wait for an event in production code.
- Storing `Context` on a struct; passing `context.Background()` from a handler.
- Storing a `ctx` (on a struct or in a map) to cancel later; store the `CancelFunc`.
- Omitting `defer cancel()` for `WithCancel`/`WithTimeout`.
- `select{}` to block forever instead of returning from main.
- A blocking producer signal (`ch <- v` with no `default`) where a coalescing non-blocking wake fits.
- An unbounded in-memory queue/buffer that grows under load instead of bounded-channel backpressure.
- Silencing the race detector; shipping any `-race` finding.
- `context.Background()` for owned background work that must outlive the request (loses
  trace/request values); the live request `ctx` for the same (gets cancelled with the request).
- A bare `string`/`int` context value key instead of an unexported type.
- `recover()` anywhere other than the worker-supervisor loop or transport panic-recovery
  middleware/interceptor.
{{end}}

{{define "validation"}}
- [ ] Every goroutine has a known owner and stop condition.
- [ ] `ctx` is the first param of blocking calls; not stored on structs.
- [ ] Inbound context propagated; `WithCancel`/`WithTimeout` paired with `defer cancel()`.
- [ ] Fan-out uses `errgroup`/`WaitGroup`, not hand-rolled channel coordination.
- [ ] Worker loops select on `ctx.Done()` with a ticker; no `time.Sleep` pacing.
- [ ] Channel ownership documented; two-value receive used.
- [ ] Tests pass under `-race`.
- [ ] Work that outlives its request is cancelled via a stored `CancelFunc`, never a stored `ctx`.
- [ ] Owned background work uses `context.WithoutCancel(ctx)`, not `context.Background()` or the
  live request `ctx`.
- [ ] Context value keys are an unexported type, never a bare `string`/`int`.
- [ ] Bounded fan-out with no per-key ordering uses `errgroup.SetLimit(n)`; ordered/durable
  fan-out uses `[[dispatch-pool]]` instead.
- [ ] `recover()` appears only at a worker-supervisor boundary or transport panic-recovery
  middleware/interceptor.
{{end}}
