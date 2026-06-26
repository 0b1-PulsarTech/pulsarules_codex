---
id: retry-backoff
name: Retry & backoff
description: An injected Policy wrapped as a Decorator over a port; retry only retryable errors; backoff waits on context; only retry writes with an idempotency key.
tags:
    - go
    - resilience
dependencies:
    - concurrency
---

# Retry & backoff

> A `Policy{MaxAttempts, BaseDelay, MaxDelay, Jitter, IsRetryable}` injected at bootstrap and
> wrapped as a Decorator over a port; retry only retryable errors (429/5xx/network, not
> 4xx/validation); backoff waits on context, never `time.Sleep`; only retry writes with an
> idempotency key.

Reference tools: a backoff helper package.

{{define "when"}}
- A sender/port can fail transiently (429, 5xx, network timeout).
- Wrapping a provider adapter, webhook delivery, or external call.
- An operation carries an idempotency key (safe to retry writes).
{{end}}

{{define "recipe"}}
```go
type retrySender struct {
    inner  Sender
    policy backoff.Policy
}

func (s retrySender) Send(ctx context.Context, msg Message) (Receipt, error) {
    return backoff.Do(ctx, s.policy, func(ctx context.Context) (Receipt, error) {
        return s.inner.Send(ctx, msg)
    })
}
```

Policy built and injected at bootstrap (never a package-level default):

```go
policy := backoff.Policy{
    MaxAttempts: 5,
    BaseDelay:   200 * time.Millisecond,
    MaxDelay:    10 * time.Second,
    Jitter:      0.2,
    IsRetryable: func(err error) bool {
        var s transportStatus
        if errors.As(err, &s) {
            return s.Code == 429 || s.Code >= 500
        }
        return errors.Is(err, context.DeadlineExceeded) || isNetError(err)
    },
}
remy.RegisterInstance(inj, policy)
remy.RegisterConstructorArgs2(inj, remy.Singleton[Sender],
    func(inner Sender, p backoff.Policy) Sender { return retrySender{inner, p} })
```

Backoff waits on context (inside the helper):

```go
select {
case <-ctx.Done():
    return Receipt{}, ctx.Err()
case <-time.After(delay):
}
```

Test with a fake clock or microsecond `BaseDelay`: assert attempt count, permanent-error
short-circuit, exhaustion (`ErrExhausted`), and cancellation propagation.
{{end}}

{{define "forbidden"}}
- Retrying a non-idempotent write without an idempotency key.
- Retrying permanent errors (4xx/validation).
- A global/package-level default policy; unbounded retries or uncapped `MaxDelay`.
- `time.Sleep` for backoff; a busy-loop without checking `ctx.Done()`.
{{end}}

{{define "validation"}}
- [ ] `Policy` injected at bootstrap; `MaxAttempts`/`MaxDelay` bounded.
- [ ] `IsRetryable` false for 4xx/validation, true for transient errors.
- [ ] Retry implemented as a decorator over the port; use case unaware of retry.
- [ ] Backoff waits on context; cancellation honored.
- [ ] Writes retried only with an idempotency key.
- [ ] Table-driven tests cover success-on-N, permanent error, exhaustion, cancellation.
{{end}}
