---
id: retry-backoff
name: Retry & backoff
---

Governs retrying a transiently-failing port call: an injected `Policy` wrapped as a Decorator over
the port, retrying only retryable errors (429/5xx/network, never 4xx/validation) with backoff that
waits on context. Reach for this when wrapping a provider adapter, webhook delivery, or any external
call that can fail transiently; only retry a write when it carries an idempotency key. Distinct from
http-clients, which governs the HTTP transport plumbing itself and pushes retry policy out to the
caller - this skill is that caller's policy.

The rules below are the composed retry-backoff rule.
