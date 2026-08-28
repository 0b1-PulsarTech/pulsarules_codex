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
    - context-discipline
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
   testable under `testing/synctest`. `time.Tick` is permitted in exactly one case: a loop with no
   shutdown obligation because it lives and dies with the process. Go 1.23 made an unreferenced
   ticker collectable, which removes the GC leak but not the goroutine leak this rule is about - a
   collected ticker does not make a running loop return. Declare that single case with a
   `//pulsarules:process-lifetime <reason>` comment on or directly above the `for`; an unmarked
   ticker loop with no `<-ctx.Done()` arm is a leak, not a style choice.
6. Document who closes each channel (sender closes; receiver never). Read with the two-value form
   `val, ok := <-ch`. Default to unbuffered channels.
7. Embed `sync.Mutex` as an unexported field; lock the smallest scope; never call a user callback
   while holding the lock. Use `sync.RWMutex` only when reads vastly outnumber writes.
8. A supervised worker loop is restarted by its parent on panic. `recover()` is sanctioned at
   exactly two boundaries: this worker-supervisor loop, and transport-level panic-recovery
   middleware/interceptor (see [[rest-adapter]], [[grpc]]) - nowhere else.
9. Run `go test -race ./...`; CI runs with `-race`. Any race is a bug.
10. To cancel work that must outlive its request, keep a `map[key]context.CancelFunc` owned by the
    worker; store the `CancelFunc`, NEVER the `ctx`.
11. Hand off work without blocking the producer via a coalescing non-blocking signal: a
    buffered-by-one channel with `select { case ch <- struct{}{}: default: }` collapses many wakes
    into one pending wake and never blocks the caller. A consumer that must not block on a full
    sink drops or persists (never grows an unbounded buffer). For durable async dispatch use a
    bounded, key-sharded worker pool (see `[[dispatch-pool]]`): bounded queues are the
    backpressure, per-key sharding keeps a key ordered while different keys parallelize, and the
    sole sender closes the queues on cancel so workers drain in-flight items.
12. For owned background work that must outlive the request that started it, derive the worker's
    context with `context.WithoutCancel(ctx)`. Never `context.Background()` (it loses trace/request
    values carried on `ctx`) and never the live request `ctx` (it gets cancelled when the request
    ends, killing the background work with it).
13. Context value keys MUST be an unexported type (`type ctxKey struct{}` or an unexported named
    type), never a bare `string`/`int`, so keys cannot collide across packages.
14. For simple bounded fan-out with no per-key ordering requirement, cap concurrency with
    `errgroup.SetLimit(n)`. Reach for the heavier `[[dispatch-pool]]` only when work must stay
    ordered per key or needs a durable backlog.
15. Register cancellation-triggered cleanup with `context.AfterFunc(ctx, cleanup)`, never a goroutine
    whose entire body is `<-ctx.Done()` followed by the cleanup. That goroutine is the
    fire-and-forget shape #1 already forbids, in its most common disguise; `AfterFunc` runs the work
    when the context ends and returns a `stop` function that both unregisters it and reports whether
    it had already fired.
16. When a caller must distinguish why a context ended, cancel with a cause -
    `context.WithCancelCause(ctx)` plus `cancel(err)`, or `WithTimeoutCause`/`WithDeadlineCause` -
    and read it with `context.Cause(ctx)`. `ctx.Err()` collapses every reason into `Canceled` or
    `DeadlineExceeded`, which is too coarse to wrap into a domain error (see [[errors]]) or to branch
    on. Plain `WithCancel`/`WithTimeout` stay correct where no caller inspects the reason.
{{end}}

{{define "forbidden"}}
- `go f()` with no owner/stop tracking (leak).
- `time.Sleep` to wait for an event in production code.
- Storing `Context` on a struct; passing `context.Background()` from a handler.
- Storing a `ctx` (on a struct or in a map) to cancel later; store the `CancelFunc`.
- Omitting `defer cancel()` for `WithCancel`/`WithTimeout`.
- `select{}` to block forever instead of returning from main.
- A goroutine whose only job is to wait on `ctx.Done()` and then clean up; use `context.AfterFunc`.
- A `time.Tick` loop, or any ticker loop, with no `<-ctx.Done()` arm and no
  `//pulsarules:process-lifetime` marker declaring it process-scoped.
- Reporting a cancellation reason through `ctx.Err()` where a caller must tell the causes apart.
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
- [ ] Cancellation cleanup uses `context.AfterFunc`, not a goroutine dedicated to `<-ctx.Done()`.
- [ ] Any ticker loop without a `<-ctx.Done()` arm carries the `//pulsarules:process-lifetime`
  marker and a reason; none is assumed.
- [ ] Where the reason matters, cancellation carries a cause read via `context.Cause`, not
  `ctx.Err()`.
{{end}}
