---
id: transactions
name: Transactions
---

## Mandatory workflow

1. Run any use case doing two or more writes (including a business write plus its outbox event) in exactly one
   transaction - either all land or none. A single write uses the implicit per-statement transaction.
2. Publish the outbox event in the SAME transaction as the business write (see [[eventing-outbox]]).
3. Form A (simple `Transactioner`, one repo, a couple of writes): the repo's `BeginTx` swaps its `*Queries` onto the tx
   and returns a finisher `func(commit bool) error` that restores the saved queries and commits or rolls back. Assert
   `var _ tx.Transactioner = (*Repo)(nil)`.
4. In the use case, drive commit-on-success / rollback-on-error with `defer` keyed on the named return error:
   `defer func() { err = errors.Join(err, onFinish(err == nil)) }()`.
5. Form B (multi-step flows): the repo's `Begin(ctx)` returns a domain `Tx` interface (declared by the consuming use
   case) exposing only the writes it performs plus `Commit`/`Rollback`.
6. Fold the rollback error with `errors.Join`; ignore `sql.ErrTxDone` on rollback.
7. Register a tx-carrying repo as a `Factory`, never a singleton (it holds swap-in-place or tx state).
8. Cover commit and rollback paths with integration tests against a real DB.

## Validation checklist

- [ ] Every multi-write flow opens exactly one transaction.
- [ ] Commit-on-success / rollback-on-error via `defer`, keyed on the named return error.
- [ ] Rollback errors folded with `errors.Join`; `sql.ErrTxDone` ignored on rollback.
- [ ] Outbox event published inside the same transaction.
- [ ] Tx-carrying repository registered as a factory, not a singleton.
- [ ] Commit and rollback paths covered by integration tests (real DB).

## Forbidden actions

- Two or more writes without a transaction.
- Emitting an outbox event outside the business transaction.
- Holding a transaction open across an HTTP/gRPC call or slow external I/O.
- Returning `*dbgen.Queries` or `*sql.Tx` from a repository.
- Sharing a swap-in-place `Transactioner` repo as a process singleton.
- Swallowing the rollback error (fold with `errors.Join`; ignore `sql.ErrTxDone`).

## Expected outputs

- One transaction per multi-write flow, with the outbox event committed inside it.
- Commit/rollback driven by a `defer` on the named return error; rollback errors folded, never swallowed.
- A tx-carrying repo registered as a factory; commit and rollback covered by real-DB tests.
