---
id: transactions
name: Transactions
description: Any use case doing two or more writes runs them in one transaction; the outbox event commits in the same transaction.
tags:
    - go
    - database
dependencies:
    - database
    - eventing
---

# Transactions

> Any use case doing two or more writes (including a business write plus its outbox event) runs them
> in one transaction - either all land or none do. A single write uses the implicit per-statement
> transaction. The outbox event commits in the SAME transaction (see [[eventing-outbox]]).

Reference tools: `sqlc` `Queries.WithTx`; a small tx helper package.

{{define "when"}}
- A use case performs more than one write/update/delete.
- Writing business state and an outbox event together.
- Implementing a unit-of-work `Tx` port.
{{end}}

{{define "must"}}
1. Run any use case doing two or more writes (including a business write plus its outbox event) in
   exactly one transaction - either all land or none. A single write uses the implicit per-statement
   transaction.
2. Publish the outbox event in the SAME transaction as the business write (see [[eventing-outbox]]).
3. Form A (simple `Transactioner`, one repo, a couple of writes): the repo's `BeginTx` swaps its
   `*Queries` onto the tx and returns a finisher `func(commit bool) error` that restores the saved
   queries and commits or rolls back. Assert `var _ tx.Transactioner = (*Repo)(nil)`.
4. In the use case, drive commit-on-success / rollback-on-error with `defer` keyed on the named return
   error: `defer func() { err = errors.Join(err, onFinish(err == nil)) }()`.
5. Form B (multi-step flows): the repo's `Begin(ctx)` returns a domain `Tx` interface (declared by the
   consuming use case) exposing only the writes it performs plus `Commit`/`Rollback`.
6. Fold the rollback error with `errors.Join`; ignore `sql.ErrTxDone` on rollback.
7. Register a tx-carrying repo as a `Factory`, never a singleton (it holds swap-in-place or tx state).
8. Cover commit and rollback paths with integration tests against a real DB.
9. Use `BeginTx(ctx, opts *sql.TxOptions)`'s options: `SELECT ... FOR UPDATE` when reading rows the
   same transaction will modify (prevents lost-update races), and `Serializable` for money/ledger
   operations. On a Postgres serialization failure (SQLSTATE `40001`), retry the WHOLE transaction,
   never just the failing statement, using the same bounded `Policy` shape as [[retry-backoff]]
   (`IsRetryable` matching `40001`, backoff waiting on context).
{{end}}

{{define "recipe"}}
Form A - simple `Transactioner` (one repo, a couple of writes). The repo swaps its queries onto the
tx and returns a finisher:

```go
func (r *Repo) BeginTx(ctx context.Context, opts *sql.TxOptions) (tx.OnFinishFunc, error) {
    tx, err := r.db.BeginTx(ctx, opts)
    if err != nil {
        return nil, fmt.Errorf("begin tx: %w", err)
    }
    saved := r.queries
    r.queries = r.queries.WithTx(tx)
    return func(commit bool) error {
        r.queries = saved
        if commit {
            return tx.Commit()
        }
        if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
            return err
        }
        return nil
    }, nil
}

var _ tx.Transactioner = (*Repo)(nil)
```

Use case (defer keyed on the named return error):

```go
func (uc UseCase) Move(ctx context.Context, in MoveInput) (entities.Thing, error) {
    onFinish, err := uc.repo.BeginTx(ctx, nil)
    if err != nil {
        return entities.Thing{}, fmt.Errorf("begin tx: %w", err)
    }
    defer func() { err = errors.Join(err, onFinish(err == nil)) }()

    l, err := uc.repo.UpdateStage(ctx, in.ID, in.Stage)
    if err != nil {
        return entities.Thing{}, fmt.Errorf("update stage: %w", err)
    }
    if err := uc.events.Publish(ctx, thing.Moved(in.ID, in.Stage)); err != nil {
        return entities.Thing{}, fmt.Errorf("publish event: %w", err)
    }
    return l, nil
}
```

Form B - unit-of-work `Tx` port for multi-step flows: the repo's `Begin(ctx)` returns a domain `Tx`
interface (declared by the consuming use case) exposing only the writes it performs plus
`Commit`/`Rollback`. Register a tx-carrying repo as a `Factory`, never a singleton.

Isolation options and retry on serialization failure:

```go
onFinish, err := uc.repo.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
```

```go
policy := backoff.Policy{
    MaxAttempts: 5,
    BaseDelay:   50 * time.Millisecond,
    IsRetryable: func(err error) bool {
        var pgErr *pgconn.PgError
        return errors.As(err, &pgErr) && pgErr.Code == "40001" // retry the WHOLE tx, not the statement
    },
}
_, err := backoff.Do(ctx, policy, func(ctx context.Context) (entities.Ledger, error) {
    return uc.Move(ctx, in) // re-runs BeginTx and every statement inside it
})
```
{{end}}

{{define "forbidden"}}
- Two or more writes without a transaction.
- Emitting an outbox event outside the business transaction.
- Holding a transaction open across an HTTP/gRPC call or slow external I/O.
- Returning `*dbgen.Queries` or `*sql.Tx` from a repository.
- Sharing a swap-in-place `Transactioner` repo as a process singleton.
- Swallowing the rollback error (fold with `errors.Join`; ignore `sql.ErrTxDone`).
- Retrying only the failing statement on a `40001` serialization failure instead of the whole
  transaction.
- Default/`ReadCommitted` isolation for a money/ledger write with no `SELECT ... FOR UPDATE` or
  `Serializable` justification.
{{end}}

{{define "validation"}}
- [ ] Every multi-write flow opens exactly one transaction.
- [ ] Commit-on-success / rollback-on-error via `defer`, keyed on the named return error.
- [ ] Rollback errors folded with `errors.Join`; `sql.ErrTxDone` ignored on rollback.
- [ ] Outbox event published inside the same transaction.
- [ ] Tx-carrying repository registered as a factory, not a singleton.
- [ ] Commit and rollback paths covered by integration tests (real DB).
- [ ] Reads intended for same-tx modification use `SELECT ... FOR UPDATE`; money/ledger operations
  use `Serializable`.
- [ ] A `40001` serialization failure retries the whole transaction, via a bounded policy, not the
  failing statement alone.
{{end}}
