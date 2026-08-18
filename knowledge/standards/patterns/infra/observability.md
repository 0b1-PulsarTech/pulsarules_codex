---
id: observability
name: Observability (tracing abstraction)
description: An injected Tracer (never global); spans with defer span.End(); errors recorded on the originating span; the concrete tracer chosen at cmd/infra with OTel confined to the adapter.
tags:
    - go
    - observability
dependencies:
    - dependency-injection
---

# Observability (tracing abstraction)

> An injected `Tracer` (never global); spans started in use cases with `defer span.End()`; errors
> recorded on the originating span; logs correlated via `WithSpan`; the concrete tracer (no-op,
> slog-backed basic, or OTel) chosen at cmd/infra bootstrap with OTel imports confined to the
> adapter.

Reference tools: a tracer abstraction package; OpenTelemetry wired only at the infra adapter.

{{define "when"}}
- Instrumenting a use case with tracing spans.
- Wiring the observability adapter at bootstrap.
- Correlating logs with spans.
{{end}}

{{define "must"}}
1. Inject the `Tracer` (never a global); start spans in use cases with
   `ctx, span := uc.tracer.Start(ctx, "name", attrs)` and always `defer span.End()`.
2. Record errors on the originating span (`span.RecordError(err)`; `span.SetStatus(false, "action")`)
   and return the wrapped error.
3. Correlate logs with the span: `slog.InfoContext(tracing.WithSpan(ctx, logger), "msg", ...)`. Record
   an error once, not at every layer (see [[errors-logging]]).
4. Choose the concrete tracer at `cmd/infra` bootstrap (`"noop"`, `"basic"` slog-backed, or `"otel"`),
   and confine OTel imports to the adapter: `oteladapter.NewTracer(...)` only there.
5. Register the chosen tracer as a singleton; pass it through constructors.
6. For OTel-only features (span links, baggage), `Unwrap()` the span in the `cmd/infra` layer only -
   never in domain code.
7. Propagate context across goroutines so no span is lost.
8. Metric labels MUST be low-cardinality: never a user ID, full URL path, or request ID as a label
   value - use route templates (e.g. `/users/{id}`) instead. Rule of thumb: a label taking more than
   ~100 distinct values is too many.
{{end}}

{{define "recipe"}}
Inject the tracer; start a span; `defer span.End()`:

```go
type UseCase struct {
    repo   Repository
    tracer tracing.Tracer
}

func (uc UseCase) Create(ctx context.Context, in CreateInput) (entities.Thing, error) {
    ctx, span := uc.tracer.Start(ctx, "thing.Create", tracing.String("owner", in.Owner))
    defer span.End()

    l, err := uc.repo.Insert(ctx, entities.Thing{Name: in.Name, Owner: in.Owner})
    if err != nil {
        span.RecordError(err)
        span.SetStatus(false, "insert thing")
        return entities.Thing{}, fmt.Errorf("insert thing: %w", err)
    }
    span.SetAttrs(tracing.String("thing.id", l.ID.String()))
    return l, nil
}
```

Pick the implementation at cmd/infra bootstrap:

```go
var tracer tracing.Tracer
switch conf.Tracing {
case "otel":
    tracer = oteladapter.NewTracer(exporter, resource, propagators) // OTel imports only here
case "basic":
    tracer = tracing.NewBasic(logger)
default:
    tracer = tracing.NewNoop()
}
remy.RegisterInstance(inj, tracer)
```

Correlate logs with the span:

```go
slog.InfoContext(tracing.WithSpan(ctx, logger), "thing created", slog.String("thing.id", id))
```

For OTel-only features (span links, baggage), unwrap in the cmd/infra layer only:

```go
if os, ok := span.Unwrap().(oteltrace.Span); ok {
    os.AddLink(link)
}
```

Low-cardinality metric labels - a route template, not the raw path or ID:

```go
requestsTotal.With(metric.String("route", "/users/{id}")).Inc() // not r.URL.Path or userID
```
{{end}}

{{define "forbidden"}}
- Importing `go.opentelemetry.io/*` outside the cmd/infra OTel adapter.
- A global/package-level tracer or `init()`-registered exporter.
- Calling `Unwrap()` in domain code.
- Logging and recording the same error at every layer.
- Losing a span across goroutines by not propagating context.
- A metric label carrying a user ID, full URL path, or request ID (unbounded cardinality).
{{end}}

{{define "validation"}}
- [ ] `Tracer` injected; no global tracer or `init()` exporter.
- [ ] Spans started with `Start` and always `defer span.End()`.
- [ ] Errors recorded on the originating span; status set on failure.
- [ ] Logs correlated via `WithSpan`; error recorded once, not at every layer.
- [ ] Concrete tracer chosen at cmd/infra; OTel imports confined to the adapter.
- [ ] `Unwrap()` used only in cmd/infra, never in the domain.
- [ ] Metric labels are low-cardinality (route templates, not raw user IDs/paths/request IDs); no
  label exceeds ~100 distinct values.
{{end}}
