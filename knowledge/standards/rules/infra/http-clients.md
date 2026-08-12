---
id: http-clients
name: HTTP clients
description: All outbound HTTP through one infra gateway; no http.DefaultClient; per-request timeouts; no TLS skip-verify; cache at the call site; retries are the caller's responsibility.
tags:
    - go
    - http
dependencies:
    - security
linters:
    - depguard
    - gosec
analyzers:
    - golangci-lint
---

# HTTP clients

> All outbound HTTP goes through one infra gateway package; no `http.Get`, no `http.DefaultClient`,
> no per-package HTTP setup; per-request timeouts; no TLS skip-verify; cache decisions at the call
> site, not in the client; retries are the caller's responsibility.

Applies to: any outbound HTTP call.

{{define "when"}}
- Making an outbound HTTP call.
- Adding a fetcher/provider/adapter that calls an external API.
- Deciding where HTTP setup lives.
{{end}}

{{define "must"}}
1. All outbound HTTP goes through a single `internal/infra/httpdatasource/` (or equivalent) gateway.
   No `http.Get`, no top-level `http.DefaultClient`, no per-package `var client = &http.Client{…}`.
2. Define a `Client` interface (`Do(ctx, Request) (Response, error)`) with a `Request` carrying
   Method, URL, Headers, Body, Timeout, and an optional CacheKey. The default impl uses `net/http`
   with TLS 1.2+ and per-request timeouts.
3. Every `Do` call respects `ctx` cancellation and a per-request `Timeout`. Default 30s; override
   per-call when an upstream is known to be slow.
4. The client does not retry. Retries with backoff are the caller's responsibility (a fetcher) so the
   policy stays visible (see retry-backoff pattern).
5. Cache decisions are at the call site, not in the client: the fetcher consults a cache before
   issuing the request and writes through after. The HTTP gateway itself never caches. Two layers,
   two responsibilities.
6. A non-default TLS backend (e.g. libcurl) is opt-in behind a `//go:build curl` tag; default builds
   are CGO-free and do not require it.
{{end}}

{{define "forbidden"}}
- `http.DefaultClient`; `http.Get`.
- A package-level `var client = &http.Client{…}` outside the HTTP gateway.
- HTTP calls outside the fetcher/provider/discord-facade layer (the gateway's own users).
- TLS skip-verify. Ever. If a cert chain fails, fix the chain.
- Hiding retries inside the gateway (policy stays visible at the caller).
- `go run` to launch a daemon that makes HTTP calls - use the built binary.
{{end}}

{{define "validation"}}
- [ ] One HTTP gateway package; no `http.DefaultClient`/`http.Get`; no per-package clients.
- [ ] `Client` interface with per-request timeout; `ctx` honored.
- [ ] Client does not retry; retries at the caller with backoff.
- [ ] Cache at the call site, not in the gateway.
- [ ] No TLS skip-verify; non-default TLS behind a build tag; default builds CGO-free.
{{end}}
