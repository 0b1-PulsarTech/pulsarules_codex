---
id: external-provider
name: External provider
---

External provider governs onboarding a new external integration as a Strategy port: a
per-provider package with the fixed file set (fetcher, client, url, dto, mapper, di), unexported
DTOs, and bootstrap switching on a provider code. Reach for it when adding a new external HTTP
fetcher, provider, or integration, or onboarding any Strategy implementation behind a
consumer-declared port. It composes http-clients for the outbound call itself (the shared
gateway, per-request timeouts, no default client) and dependency-injection for wiring the
binding - this skill is specifically about the provider package's shape and the mapper boundary
that keeps its DTOs from leaking into the domain.

The rules below are the composed external-provider rule.
