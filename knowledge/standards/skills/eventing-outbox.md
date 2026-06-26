---
id: eventing-outbox
name: Eventing & outbox
---

Eventing and outbox governs any side effect that leaves a use case's boundary: it goes through
the transactional outbox, never an inline webhook call or a fire-and-forget goroutine. Reach for
it when a use case needs to notify outside its boundary, implementing the relay worker or a
sink, defining a domain event or the event catalog, projecting the audit trail, or choosing
between the durable outbox and the ephemeral weak-pointer bus for live, best-effort fan-out. It
shares the same-transaction requirement with transactions - the event row commits with the
business write or neither does.

The rules below are the composed eventing-outbox rule.
