---
id: dispatch-pool
name: Dispatch pool
description: A bounded, keyed worker pool woken by a coalescing non-blocking trigger - producers Notify without blocking, a drain loop pulls batches from storage and shards each item to a fixed worker by key (same key ordered, different keys parallel), bounded queues give backpressure, and cancel drains in-flight work.
tags:
    - go
    - concurrency
dependencies:
    - concurrency
    - eventing
---

# Dispatch pool

> Decouple "something happened" from "work it off" without blocking the producer. A producer signals
> a coalescing `Notifier` (a non-blocking send onto a buffered-by-one channel); a `Pool` drives N
> workers, drains due items from storage in batches on each wake, and shards each item to a fixed
> worker by a stable key hash so a key's items keep order while different keys run in parallel.
> Bounded per-worker queues are the backpressure; on context cancel the drain loop (the sole sender)
> closes the queues so each worker finishes its buffered items before `Run` returns.

Reference: `terectek_comms` `libs/tereckernel/concorre/pooltirao` (`Pool[T]`, `Handler[T]`) and
`libs/tereckernel/eventflow` (`Notifier`).

{{define "when"}}
- A producer must hand off work (send/receive a message, process an event) WITHOUT blocking on the
  worker - the commit path stays fast and the work drains asynchronously.
- Work must be ordered per entity (per conversation, per contact) but parallel across entities.
- A durable backlog in storage must be drained to empty on each wake, with backpressure when workers
  fall behind.
{{end}}

{{define "recipe"}}
The coalescing, non-blocking trigger - many `Notify` calls collapse to one pending wake, and `Notify`
never blocks the producer:

```go
type Notifier struct{ ch chan struct{} }

func NewNotifier() *Notifier { return &Notifier{ch: make(chan struct{}, 1)} }

func (n *Notifier) Notify() { // non-blocking: a wake already pending is a no-op
    select {
    case n.ch <- struct{}{}:
    default:
    }
}

func (n *Notifier) Signal() <-chan struct{} { return n.ch }
```

The work contract - claim a batch, process one item, name the shard key:

```go
type Handler[T any] interface {
    Pull(ctx context.Context, limit int) ([]T, error) // claim due items; short batch = drained
    Process(ctx context.Context, item T) error        // transient err retried; deterministic skip returns nil
    Key(item T) string                                // items sharing a key run on one worker, in order
}
```

The pool - drain on wake or idle tick, shard with a ctx-guarded (backpressuring) send, drain in-flight
on cancel:

```go
func (p *Pool[T]) Run(ctx context.Context) error {
    queues := make([]chan T, p.workers)
    var workers sync.WaitGroup
    for i := range queues {
        queues[i] = make(chan T, p.batch) // bounded => backpressure
        q := queues[i]
        workers.Go(func() { for item := range q { p.process(ctx, item) } })
    }
    p.loop(ctx, queues) // select: ctx.Done | trigger.Signal() | idle ticker -> drain
    for _, q := range queues {
        close(q) // sole sender closes; workers range-drain their buffers, no send-on-closed race
    }
    workers.Wait()
    return fmt.Errorf("dispatch: run stopped: %w", ctx.Err())
}

// drain: Pull batches until a short batch; ctx-guarded send is the backpressure + cancel point.
for _, item := range items {
    select {
    case queues[p.shard(p.handler.Key(item))] <- item:
    case <-ctx.Done():
        return
    }
}
```

`shard(key)` is a stable hash (`fnv32a(key) % workers`) so a key always lands on the same worker.
`process` wraps `Handler.Process` in the retry policy and a `recover()` so one poison item never kills
the worker.
{{end}}

{{define "forbidden"}}
- A blocking `Notify` (a plain `ch <- struct{}{}` with no `default`) - it couples the producer to the
  worker's pace; use the non-blocking select.
- An unbounded queue (or `Notify` that grows a slice) - backpressure disappears and memory is
  unbounded; the bounded worker channel IS the backpressure.
- A worker as the channel closer - only the drain loop (the sole sender) closes, after `loop` returns.
- Sharding that lets one key's items run on two workers - it loses per-key ordering.
- Dropping items silently on a full queue when the work is durable - block (backpressure) or persist;
  only drop when the signal is genuinely coalescible (like the wake itself).
{{end}}

{{define "validation"}}
- [ ] Producers signal via a non-blocking coalescing `Notify`; the commit path never blocks on a worker.
- [ ] Per-worker queues are bounded; the ctx-guarded send is the only backpressure + cancel point.
- [ ] Same key always shards to the same worker (stable hash); different keys parallelize.
- [ ] On cancel the sole sender closes the queues, workers drain their buffers, then `Run` returns
      the wrapped `ctx.Err()`.
- [ ] Each item is retried by policy and isolated by `recover()`; a short `Pull` batch ends the drain.
{{end}}
