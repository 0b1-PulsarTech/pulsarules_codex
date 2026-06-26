---
id: transactions
name: Transactions
---

Governs multi-write atomicity: any use case performing two or more writes - including a business
write plus its outbox event - runs them in exactly one transaction, committed or rolled back
together via `defer` keyed on the named return error. Reach for this when a use case does more than
one write, or writes business state and an outbox event together. Not the same as
database-persistence, which governs the repository/query/mapping chain for a single write -
transactions is specifically the atomicity wrapper around two or more of those writes. Pairs with
eventing-outbox for the event that must commit in the same transaction.

The rules below are the composed transactions rule.
