---
id: external-provider
name: External provider
---

## Mandatory workflow

1. Model the external integration as a Strategy port declared in the consuming domain; the concrete HTTP implementation
   lives behind it. The provider is selected by code at bootstrap, not by the lib.
2. Lay the per-provider package out under `internal/infra/fetchers/<name>fetch/` (or `internal/infra/providers/<name>/`)
   with the fixed file set: `fetcher.go` (`Fetcher` struct + `Code()`/`DisplayName()`/`Sync`), `client.go` (HTTP via the
   gateway, cache-aware), `url.go` (endpoint builders, URLs from config), `dto.go` (package-local API DTOs, never
   exported), `mapper.go` (dto -> domain), `di.go` (`Register(inj, baseURL, lang)`), `doc.go` (package comment +
   endpoint list).
3. Implement the Strategy port: fetch via the HTTP gateway, consult a cache at the call site before issuing and write
   through after, decode into unexported DTOs, map to a domain `SyncDelta`/value, and return domain types only.
4. Register both the concrete binding and the Strategy binding the consuming set consumes; switch on the provider code
   ONLY in bootstrap.
5. Add config: a `[[providers]]` table (`code`, `enabled`, `base_url`, `language`) consumed by the bootstrap
   switchboard.
6. Wrap the provider's calls with a retry/backoff Decorator over the port when it can fail transiently (
   see [[retry-backoff]]); keep the use case unaware of retry.
7. Write table-driven `mapper.go` tests (happy path + at least one edge case).

## Validation checklist

- [ ] Fixed file set; DTOs unexported; mapping in `mapper.go`; `doc.go` lists endpoints.
- [ ] DI registers concrete + Strategy bindings; bootstrap switches on code.
- [ ] `mapper.go` has table-driven tests (happy path + one edge case).
- [ ] No hardcoded URL/language; HTTP only via the gateway; cache at the call site.
- [ ] Transient failures handled by a retry Decorator over the port, not inside the fetcher.

## Forbidden actions

- Hardcoded URLs or language outside `url.go`/config.
- Exported DTOs; mapping outside `mapper.go`.
- A `switch code` inside the fetcher package (bootstrap owns it).
- HTTP calls outside the gateway; retries hidden inside the gateway/fetcher.
- Leaking provider DTOs into the domain.

## Expected outputs

- A per-provider package with the fixed file set implementing a Strategy port.
- DTOs unexported; mapping in `mapper.go` with table-driven tests; bootstrap switches on code.
- HTTP via the gateway; cache at the call site; retry as a Decorator over the port.
