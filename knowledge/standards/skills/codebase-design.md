---
id: codebase-design
name: Codebase design
---

Shared vocabulary for designing deep modules: a lot of behavior behind a small interface,
placed at a clean seam, testable through that interface. Use this language whenever code is
being designed or restructured. The payoff is leverage for callers, locality for maintainers,
and testability for everyone. This is the design vocabulary that `design-patterns`,
`refactoring`, and `improve-codebase-architecture` build on.

## Glossary (use these terms exactly)

Do not substitute "component", "service", "API", or "boundary" - consistent language is the point.

- Module - anything with an interface and an implementation: a function, a type, a package, or
  a slice spanning several. Scale-agnostic on purpose.
- Interface - everything a caller must know to use the module correctly: the signature, but also
  invariants, ordering constraints, error modes, required config, and performance. Wider than a
  Go `interface` type or a package's exported set.
- Implementation - the code inside a module. Distinct from Adapter, which names the role a
  concrete thing fills at a seam.
- Depth - behavior a caller (or test) exercises per unit of interface it must learn. Deep = a lot
  of behavior behind a small interface; shallow = the interface is nearly as complex as the body.
- Seam - a place where behavior can change without editing in that place; the location where the
  interface lives. Prefer "seam" over "boundary" (overloaded with DDD's bounded context).
- Leverage - what callers gain from depth: more capability per unit of interface. One
  implementation pays back across N call sites and M tests.
- Locality - what maintainers gain from depth: change, bugs, knowledge, and verification
  concentrate in one place. Fix once, fixed everywhere.

## The vocabulary in house terms

Map the abstract terms onto the constructs this codebase actually uses, so a suggestion names real
things:

- A deep MODULE is a `UseCase` (one action per file, business invariants inside), an `Engine` /
  `EngineFactory`, or a `Pipeline` / `Step`.
- The INTERFACE is a consumer-declared port named for its ROLE - `Repository`, `Sender`, `Resolver`,
  `Notifier` - never a generic `Port`; keep it the smallest set the consumer needs.
- The SEAM is usually a `Facade` (a cross-module boundary, a marker interface with a private method so
  it is only satisfiable through its blessed constructor) or a per-package port in `ports.go`.
- An ADAPTER is a driver-named implementation (`mysqlrepo`, `repomongo`) at that seam; a second driver
  is what makes the seam real rather than hypothetical.
- A type-keyed constructor registry (`BindKey[T]`, `ProviderKey[V]`, a `map[Kind]constructor`) is a
  deep selection module - no `switch`, no reflection at the call site.

## Designing for testability

- Accept dependencies, do not construct them: `ProcessOrder(order Order, gateway PaymentGateway)`,
  not a body that news up a concrete gateway.
- Return results, avoid hidden side effects: prefer `CalculateDiscount(cart) Discount` over a
  function that mutates the cart in place.
- Keep the surface small: fewer methods mean fewer tests, fewer parameters mean simpler setup.
