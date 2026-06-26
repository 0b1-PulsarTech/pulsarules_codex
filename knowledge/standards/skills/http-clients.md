---
id: http-clients
name: HTTP clients
---

## Mandatory workflow

1. Route all outbound HTTP through a single `internal/infra/httpdatasource/` (or equivalent) gateway. No `http.Get`, no
   top-level `http.DefaultClient`, no per-package `var client = &http.Client{...}`.
2. Define a `Client` interface (`Do(ctx, Request) (Response, error)`) with a `Request` carrying Method, URL, Headers,
   Body, Timeout, and an optional CacheKey. The default impl uses `net/http` with TLS 1.2+ and per-request timeouts.
3. Make every `Do` call respect `ctx` cancellation and a per-request `Timeout`. Default 30s; override per-call when an
   upstream is known to be slow.
4. Keep the client free of retry. Retries with backoff are the caller's responsibility (a fetcher/provider) so the
   policy stays visible (see [[retry-backoff]]).
5. Put cache decisions at the call site, not in the client: the fetcher consults a cache before issuing the request and
   writes through after. The gateway itself never caches. Two layers, two responsibilities.
6. Make a non-default TLS backend (e.g. libcurl) opt-in behind a `//go:build curl` tag; default builds are CGO-free and
   do not require it.
7. Call the gateway only from the fetcher/provider/discord-facade layer (the gateway's own users).

## Validation checklist

- [ ] One HTTP gateway package; no `http.DefaultClient`/`http.Get`; no per-package clients.
- [ ] `Client` interface with per-request timeout; `ctx` honored.
- [ ] Client does not retry; retries at the caller with backoff.
- [ ] Cache at the call site, not in the gateway.
- [ ] No TLS skip-verify; non-default TLS behind a build tag; default builds CGO-free.

## Forbidden actions

- `http.DefaultClient`; `http.Get`.
- A package-level `var client = &http.Client{...}` outside the HTTP gateway.
- HTTP calls outside the fetcher/provider/discord-facade layer.
- TLS skip-verify. Ever. If a cert chain fails, fix the chain.
- Hiding retries inside the gateway (policy stays visible at the caller).
- `go run` to launch a daemon that makes HTTP calls - use the built binary.

## Expected outputs

- One HTTP gateway with a `Client` interface and per-request timeouts honoring `ctx`.
- Retries and caching at the call site; the gateway neither retries nor caches.
- CGO-free default builds; no TLS skip-verify.
