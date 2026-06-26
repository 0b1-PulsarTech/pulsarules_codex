---
id: retry-backoff
name: Retry & backoff
---

## Mandatory workflow

1. Define a `Policy{MaxAttempts, BaseDelay, MaxDelay, Jitter, IsRetryable}` injected at bootstrap, never a package-level
   default. Bound `MaxAttempts` and cap `MaxDelay`.
2. Make `IsRetryable` true for transient errors only (429/5xx/network timeout, `context.DeadlineExceeded`); false for
   4xx/validation/permanent errors.
3. Implement retry as a Decorator over the port (`retrySender{inner, policy}`), so the use case is unaware of retry.
4. Backoff by waiting on context, never `time.Sleep`:
   `select { case <-ctx.Done(): return ctx.Err(); case <-time.After(delay): }`. Honor cancellation.
5. Retry writes only when the operation carries an idempotency key (so a replay is safe). Never retry a non-idempotent
   write without one.
6. Test with a fake clock or microsecond `BaseDelay`: success-on-N, permanent-error short-circuit, exhaustion (
   `ErrExhausted`), and cancellation propagation - all table-driven.

## Validation checklist

- [ ] `Policy` injected at bootstrap; `MaxAttempts`/`MaxDelay` bounded.
- [ ] `IsRetryable` false for 4xx/validation, true for transient errors.
- [ ] Retry implemented as a decorator over the port; use case unaware of retry.
- [ ] Backoff waits on context; cancellation honored.
- [ ] Writes retried only with an idempotency key.
- [ ] Table-driven tests cover success-on-N, permanent error, exhaustion, cancellation.

## Forbidden actions

- Retrying a non-idempotent write without an idempotency key.
- Retrying permanent errors (4xx/validation).
- A global/package-level default policy; unbounded retries or uncapped `MaxDelay`.
- `time.Sleep` for backoff; a busy-loop without checking `ctx.Done()`.

## Expected outputs

- An injected, bounded `Policy` applied as a Decorator over the port.
- Backoff that waits on context; only transient errors retried; writes retried only with an idempotency key.
- Table-driven tests covering success, permanent-error, exhaustion, and cancellation.
