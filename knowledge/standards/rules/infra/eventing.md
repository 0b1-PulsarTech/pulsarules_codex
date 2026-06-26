---
id: eventing
name: Eventing & outbox
description: Cross-boundary side effects go through a transactional outbox; deterministic idempotency keys; idempotent sinks deduped by a DB unique index; replay-safe projections.
tags:
    - go
    - eventing
dependencies:
    - concurrency
---

# Eventing & outbox

> Cross-boundary side effects go through a transactional outbox: a use case persists business state
> and a domain-event row in one transaction (never inline webhooks/HTTP); a relay claims due rows and
> fans out to idempotent sinks with backoff; projections (audit, timelines) are replay-safe via a
> deterministic idempotency key + unique index.

Applies to: any side effect that leaves a use-case boundary.

{{define "when"}}
- A use case triggers a side effect outside its boundary (notification, webhook, enrichment,
  integration).
- Building a relay/worker or an idempotent sink.
- Defining a domain event or the event catalog.
- Projecting an audit trail or handling an erasure/DSAR job.
- Choosing durable outbox vs ephemeral in-process fan-out.
{{end}}

{{define "must"}}
1. A side effect leaving the use-case boundary goes through the outbox - never an inline webhook or
   HTTP/gRPC call, never a "publish later" goroutine.
2. In the same transaction as the business write, insert the event. The event carries `TenantID`,
   `Type` (catalog), `AggregateID`, `Payload` (JSON), `IdempotencyKey`, `OccurredAt`. Both rows
   commit or neither (see [[transactions]]). The `IdempotencyKey` is **deterministic** - derived
   from the event's stable identity, never random or timestamp-based.
3. The outbox row has `status`, `attempts`, `next_attempt_at`. The relay `Run(ctx)` loops on a
   ticker + `ctx.Done()`, claims due pending/retryable rows, decodes each, and delivers.
4. Deliver fans out to every registered sink and matching webhook subscription (HMAC-signed POST),
   then projects audit. On success mark sent; on failure mark retry with capped exponential backoff,
   flipping to failed past the cap (see [[concurrency]], retry pattern).
5. A sink implements `Handle(ctx, evt) error`, is idempotent, enriches/notifies only, and returns an
   error to signal retry. Dedupe is a **DB unique-index hit** on `unique(sink, idempotency_key)`,
   recorded in the **same tx** as the enrich/insert - so dedupe survives a restart; never an
   in-memory `seen` set.
6. The worker is supervised: `recover()` only at that boundary; one owner per goroutine; cancel on
   SIGTERM.
7. Audit/timeline projections are projected from the event stream as the relay processes - never
   written ad-hoc. Each projection write is keyed by the deterministic `IdempotencyKey` behind a
   unique index, so it is replay-safe and rebuildable. A rollup that must match a canonical write
   updates in the same tx; cross-boundary projections run async off the relay.
8. For ephemeral, in-process, best-effort fan-out (live subscribers, presence/typing) use a weak
   -pointer bus: `Subscribe` holds a `weak.Pointer`; the caller keeps the strong ref; `Emit` sweeps
   GC'd entries. No durable delivery here.
9. Data erasure/DSAR runs as relay jobs, not inline; erasure records a `data.erased` event.
{{end}}

{{define "forbidden"}}
- Publishing an event outside the transaction, or a fire-and-forget goroutine to "publish later".
- Calling an external webhook / another app directly from a use case or handler.
- A non-idempotent sink; swallowing a sink error to mark a row sent.
- A random/timestamp `IdempotencyKey`, or deduping with an in-memory set that does not survive a
  restart.
- A projection that isn't replay-safe (no unique index on the key); writing audit/timeline outside
  the stream projection.
- A sink mutating canonical state outside its owning use case.
- Routing a durable side effect through the ephemeral bus; holding a durable sink as a weak pointer.
{{end}}

{{define "validation"}}
- [ ] Every cross-boundary side effect goes through the outbox (no inline calls).
- [ ] Event inserted in the same tx as the business write; carries a deterministic `IdempotencyKey`.
- [ ] Sink dedupe is a DB unique index (survives restart), recorded in the sink's tx; not in-memory.
- [ ] Projections are replay-safe via the key + unique index; consistency-critical rollups update in
  the canonical write's tx.
- [ ] Relay loop selects on `ctx.Done()` + ticker; claims due rows; marks sent/retry with backoff.
- [ ] Every sink is idempotent, enrich/notify-only, returns error for retry.
- [ ] Worker supervised; `recover()` only at the supervisor; one owner per goroutine.
- [ ] Durable vs ephemeral tier chosen correctly; ephemeral subscribers are weak refs.
{{end}}
