---
id: design-patterns
name: Design patterns (approved vs rejected)
description: Approved patterns (Strategy, Adapter, Facade, Registry, Decorator, Composite, Observer) and rejected anti-patterns (Singleton, Template-Method inheritance, reflection dispatch, Service Locator, FSM state objects) with the rule each breaks.
tags:
    - go
    - architecture
dependencies:
    - dependency-injection
    - types
---

# Design patterns (approved vs rejected)

> The classic patterns this stack uses and the anti-patterns it rejects. Approved: Strategy, Adapter,
> Facade, Registry, Decorator, Composite, Observer. Rejected (with the rule each breaks):
> Singleton/global state, Template-Method inheritance, reflection dispatch, Service Locator, FSM
> state objects.

{{define "when"}}
- Reviewing a PR/design for architectural correctness.
- Adding a new pattern or external dependency.
- Choosing DI vs singleton, inheritance vs composition, reflection vs type-safety.
{{end}}

{{define "must"}}
1. Use an Observer with weak pointers for ephemeral in-process fan-out (live subscribers); the caller
   holds the strong ref.
{{end}}

{{define "approved"}}
| Pattern   | Use it for                                                             | Reference                                                 |
|-----------|------------------------------------------------------------------------|-----------------------------------------------------------|
| Adapter   | Wrap every external dependency behind a consumer-declared port         | [[grpc-adapter]], [[external-provider]], [[http-clients]] |
| Strategy  | Behavior variants (providers, rule types, routing) selected at runtime | [[external-provider]], [[rule-engine]]                    |
| Registry  | Type-keyed constructors; reflection-free permission schemas            | [[rule-engine]], [[permitek-schema]]                      |
| Facade    | Cross-module calls; provider unaware of consumer                       | [[interop]], [[module-boundaries]]                        |
| Decorator | Resilience over a port (retry, tracing)                                | [[retry-backoff]], [[observability]]                      |
| Composite | `And`/`Or` rule selectors; permission set composition                  | [[rule-engine]], [[authorization]]                        |
| Observer  | Ephemeral in-process fan-out (weak refs)                               | [[observer-weakptr]]                                      |

A Facade is more than a call seam: its contract carries only weak, static connascence - a small named
interface plus typed DTOs (Connascence of Name + Type). Keep positional arguments, magic values, shared
algorithms, and every form of dynamic connascence (call order, timing, shared mutable state) BEHIND the
boundary, and collapse the consumer's fan-out to one stable port. See [[module-boundaries]].
{{end}}

{{define "recipe"}}
```go
// Adapter: wrap an external client behind a port declared in the consumer.
type Sender interface { Send(ctx context.Context, m Message) (Receipt, error) }

type httpSender struct{ client httpdatasource.Client }
func (s httpSender) Send(ctx context.Context, m Message) (Receipt, error) { /* ... */ }
var _ Sender = httpSender{}
```

```go
// Strategy + Registry: variants selected by key, no switch in the lib.
registry.Register("foo", func() Strategy { return &fooStrategy{} })
```

```go
// Decorator: resilience over a port (use case unaware).
type retrySender struct{ inner Sender; p Policy }
func (s retrySender) Send(ctx context.Context, m Message) (Receipt, error) {
    return Do(ctx, s.p, func(ctx context.Context) (Receipt, error) { return s.inner.Send(ctx, m) })
}
```

A **repeated type-switch** or if-else cascade duplicated across three or more sites is the smell that
calls for Strategy + a typed Registry, not another `case` (see [[code-smells]], [[rule-engine]]).
Treat frameworks, the database, and the web as **details behind a port** chosen at the composition
root, never as the thing the domain is built on (see [[frameworks-as-plugins]]).
{{end}}

{{define "rejected"}}
| Anti-pattern                                 | Reject because                              | Use instead                      |
|----------------------------------------------|---------------------------------------------|----------------------------------|
| Singleton / global mutable state             | [[startup]]: no package-level mutable state | Constructor-based DI             |
| Template-Method behavioral inheritance       | [[types]]: composition over inheritance     | Composition                      |
| Reflection-based dispatch                    | [[authorization]]: reflection-free          | Typed registries                 |
| Service Locator (injector in a collaborator) | [[dependency-injection]]                    | Concrete deps via constructors   |
| FSM state objects                            | [[types]]: typed enums over FSM objects     | Typed-string enums + transitions |

Constructor DI, concretely: the monolith has ONE composition root (`cmd/<app>`) - it calls infra
`RegisterAndInit` once, then each module's `DoInjections`. Collaborator constructors (use cases,
services, repositories) take CONCRETE deps, never `remy.Injector`/`DependencyRetriever`. The ONE place
that legitimately reads the injector is the composition seam itself (`DoInjections`, an app
`Routers(inj, ...)` builder) - the root owning wiring, not a collaborator reaching in. See
[[bootstrap-and-di]].
{{end}}

{{define "forbidden"}}
- Singleton / global mutable state (use DI).
- Template-Method behavioral inheritance / embedding for behavior (use composition).
- Reflection-based dispatch (use typed registries).
- Service Locator (a collaborator taking/storing `remy.Injector`/`DependencyRetriever`; take concrete
  deps - only the composition seam may read the injector).
- An app shipping its own `main`/DB/server instead of a `DoInjections` seam under the one composition root.
- FSM state objects (use typed-string enums + explicit transitions).
{{end}}

{{define "validation"}}
- [ ] External deps wrapped as Adapters behind ports.
- [ ] Variants use Strategy + typed Registry, not switch/inheritance.
- [ ] Cross-module calls via Facade; resilience via Decorator.
- [ ] No singletons/globals; collaborators take CONCRETE deps via constructors (only the composition
  seam reads the injector); one composition root with `DoInjections` seams.
- [ ] Composition over inheritance; typed enums over FSM objects; no reflection dispatch.
- [ ] Ephemeral fan-out uses a weak-pointer Observer; durable side effects do not.
{{end}}
