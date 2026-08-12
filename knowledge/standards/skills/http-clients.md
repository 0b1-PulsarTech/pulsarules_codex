---
id: http-clients
name: HTTP clients
---

Route every outbound call to a third-party API through a single infra HTTP gateway package, never
`http.DefaultClient` or a per-package client. Reach for this when adding a fetcher, provider, or
adapter that calls an external API, or when deciding where HTTP client setup belongs. The gateway
itself never retries and never caches - caching is a call-site decision, and retrying is
retry-backoff's job as a decorator wrapped around the port. Confusing the two is common: http-clients
governs the transport plumbing, retry-backoff governs the resilience policy layered on top of it.

The rules below are the composed http-clients rule.
