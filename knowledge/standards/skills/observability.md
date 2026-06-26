---
id: observability
name: Observability
---

## Mandatory workflow

1. Inject the `Tracer` (never a global); start spans in use cases with
   `ctx, span := uc.tracer.Start(ctx, "name", attrs)` and always `defer span.End()`.
2. Record errors on the originating span (`span.RecordError(err)`; `span.SetStatus(false, "action")`) and return the
   wrapped error.
3. Correlate logs with the span: `slog.InfoContext(tracing.WithSpan(ctx, logger), "msg", ...)`. Record an error once,
   not at every layer (see [[errors-logging]]).
4. Choose the concrete tracer at `cmd/infra` bootstrap (`"noop"`, `"basic"` slog-backed, or `"otel"`), and confine OTel
   imports to the adapter: `oteladapter.NewTracer(...)` only there.
5. Register the chosen tracer as a singleton; pass it through constructors.
6. For OTel-only features (span links, baggage), `Unwrap()` the span in the `cmd/infra` layer only - never in domain
   code.
7. Propagate context across goroutines so no span is lost.

## Validation checklist

- [ ] `Tracer` injected; no global tracer or `init()` exporter.
- [ ] Spans started with `Start` and always `defer span.End()`.
- [ ] Errors recorded on the originating span; status set on failure.
- [ ] Logs correlated via `WithSpan`; error recorded once, not at every layer.
- [ ] Concrete tracer chosen at cmd/infra; OTel imports confined to the adapter.
- [ ] `Unwrap()` used only in cmd/infra, never in the domain.

## Forbidden actions

- Importing `go.opentelemetry.io/*` outside the cmd/infra OTel adapter.
- A global/package-level tracer or `init()`-registered exporter.
- Calling `Unwrap()` in domain code.
- Logging and recording the same error at every layer.
- Losing a span across goroutines by not propagating context.

## Expected outputs

- An injected `Tracer`; spans with `defer span.End()`; errors recorded on the originating span.
- Logs correlated to spans via `WithSpan`; the concrete tracer chosen at `cmd/infra`.
- OTel imports confined to the adapter; `Unwrap()` only in `cmd/infra`.
