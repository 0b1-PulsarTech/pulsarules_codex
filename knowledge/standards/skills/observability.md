---
id: observability
name: Observability
---

Governs tracing: an injected `Tracer` (never a package-level global), spans opened in use cases
with `defer span.End()`, and errors recorded once on the originating span. Reach for this when
instrumenting a use case with spans, choosing the concrete tracer at bootstrap (no-op, basic, or
OTel), or correlating logs with a trace. OTel imports stay confined to the cmd/infra adapter; domain
code never imports `go.opentelemetry.io` directly. Distinct from errors-logging, which governs how
the error itself is wrapped and logged - this skill governs the span wrapped around it.

The rules below are the composed observability rule.
