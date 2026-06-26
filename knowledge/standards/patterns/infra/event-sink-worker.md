---
id: event-sink-worker
name: Event sink & worker
description: A relay worker claims due outbox rows and fans out to idempotent Sink[T] consumers; dedupe is a DB unique-index hit; the worker is supervised; ephemeral fan-out uses a weak-pointer bus.
tags:
    - go
    - eventing
    - concurrency
dependencies:
    - eventing
    - concurrency
composes:
    - observer-weakptr
    - retry-backoff
---

# Event sink & worker

> The relay worker `Run(ctx)` claims due outbox rows on a ticker + `ctx.Done()` and fans out to
> idempotent `Sink[DomainEvent]` consumers; dedupe is a DB unique-index hit on
> `unique(sink, idempotency_key)` recorded in the same tx as the enrich/insert; the worker is
> supervised (`recover()` only at that boundary). Ephemeral fan-out uses a weak-pointer bus.

{{define "when"}}
- Building a relay/worker that drains an outbox.
- Implementing an idempotent sink (notification, enrichment, projection).
- Choosing durable outbox vs ephemeral in-process fan-out.
{{end}}

{{define "recipe"}}
The relay loop (ticker + `ctx.Done()`):

```go
func (relay *Relay) Run(ctx context.Context) error {
    ticker := relay.clock.NewTicker(relay.interval)
    defer ticker.Stop()
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C():
            if err := relay.drain(ctx); err != nil {
                relay.logger.Error("drain outbox", slog.String("error", err.Error()))
            }
        }
    }
}

func (relay *Relay) drain(ctx context.Context) error {
    rows, err := relay.outbox.ClaimDue(ctx, relay.batchSize)
    if err != nil {
        return fmt.Errorf("claim due: %w", err)
    }
    for _, row := range rows {
        relay.deliver(ctx, row)
    }
    return nil
}
```

An idempotent sink:

```go
type NotifierSink struct {
    repo   NotificationRepo
    sender Sender
}

var _ eventing.Sink[eventing.DomainEvent] = (*NotifierSink)(nil)

func (sink *NotifierSink) Handle(ctx context.Context, evt eventing.DomainEvent) error {
    // Dedupe by unique(sink, idempotency_key) in the same tx, so it survives a restart.
    return sink.repo.InsertIfNotProcessed(ctx, evt.IdempotencyKey, evt.Payload)
}
```

Supervise the worker (`recover()` only here):

```go
func Supervise(ctx context.Context, run func(context.Context) error) error {
    for {
        err := run(ctx)
        if ctx.Err() != nil {
            return ctx.Err()
        }
        var perr panicError
        if errors.As(err, &perr) {
            slog.Error("worker panicked; restarting", slog.String("error", perr.Error()))
            continue
        }
        return err
    }
}
```

Ephemeral fan-out (weak-pointer bus) - see [[observer-weakptr]] for the registry mechanics.
{{end}}

{{define "forbidden"}}
- A non-idempotent sink; deduping with an in-memory `seen` set.
- Swallowing a sink error to mark a row sent.
- `time.Sleep` pacing; an unsupervised worker without a `ctx.Done()` exit.
- Routing a durable side effect through the ephemeral bus.
{{end}}

{{define "validation"}}
- [ ] Relay loop selects on `ctx.Done()` + ticker; claims due rows; marks sent/retry with backoff.
- [ ] Every sink is idempotent, enrich/notify-only, returns an error for retry.
- [ ] Sink dedupe is a DB unique index recorded in the sink's tx; survives restart.
- [ ] Worker supervised; `recover()` only at the supervisor; one owner per goroutine.
{{end}}
