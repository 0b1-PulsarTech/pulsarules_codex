---
id: design-patterns
name: Design patterns
---

## Mandatory workflow

1. Wrap every external dependency behind a consumer-declared port (Adapter). The adapter lives in `internal/infra/...`;
   the use case depends only on the port.
2. Model behavior variants (providers, rule types, routing) as a Strategy selected at runtime, registered in a typed
   Registry by key - no `switch` in the lib, no reflection.
3. Route cross-module calls through a Facade: a consumer-declared port implemented in `interop/facades/`; the provider
   is unaware of the consumer.
4. Add resilience (retry, tracing) as a Decorator over a port, so the use case stays unaware.
5. Compose selectors with Composite (`And`/`Or`) for rule trees and permission sets.
6. Use an Observer with weak pointers for ephemeral in-process fan-out (live subscribers); the caller holds the strong
   ref.
7. Choose DI over Singleton/global state, composition over behavioral inheritance, typed-string enums over FSM state
   objects, typed registries over reflection dispatch.
8. Reject the anti-patterns and apply the named replacement: Singleton/global state -> Constructor-based DI;
   Template-Method inheritance -> Composition; reflection dispatch -> typed Registry; Service Locator (injector on a
   struct) -> thread deps through constructors; FSM state objects -> typed enums + transitions.

## Validation checklist

- [ ] External deps wrapped as Adapters behind consumer-declared ports.
- [ ] Variants use Strategy + typed Registry, not `switch`/inheritance.
- [ ] Cross-module calls via Facade; resilience via Decorator.
- [ ] No singletons/globals; collaborators arrive via DI constructors.
- [ ] Composition over inheritance; typed enums over FSM objects; no reflection dispatch.
- [ ] Ephemeral fan-out uses a weak-pointer Observer; durable side effects do not.

## Forbidden actions

- Singleton / global mutable state (use DI).
- Template-Method behavioral inheritance / embedding for behavior (use composition).
- Reflection-based dispatch (use typed registries).
- Service Locator (storing the injector on a struct; thread deps through constructors).
- FSM state objects (use typed-string enums + explicit transitions).

## Expected outputs

- External dependencies behind ports; variants via Strategy + Registry.
- Cross-module calls via Facade; resilience via Decorator; ephemeral fan-out via weak-pointer Observer.
- No globals, no inheritance-for-behavior, no reflection dispatch, no FSM objects.
