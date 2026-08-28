---
id: types
name: Types & interfaces
description: Named structs at boundaries, consumer-declared small interfaces, accept interfaces / return concretes.
tags:
    - go
    - style
linters:
    - ireturn
    - interfacebloat
    - stylecheck
analyzers:
    - golangci-lint
---

# Types & interfaces

> Named structs at boundaries, consumer-declared small interfaces, accept interfaces / return
> concretes.

Applies to: type declarations and interface design.

{{define "when"}}
- Choosing between a named struct and `map[string]any` at a boundary.
- Declaring an interface a service depends on.
- Designing return types from public functions.
- Placing a type parameter: on the type, on a package-level function, or on a method.
{{end}}

{{define "must"}}
1. Use named structs at package boundaries; `map[string]any` only at the raw I/O edge, translated
   to a typed value immediately.
2. Declare interfaces in the **consumer** package with the smallest method set the consumer needs, and
   name the port for its role (`Repository`/`Store`/`Resolver`/`Sender`), never a generic `Port` or a
   transport-leaking name (`metaAPIPort`) even when the implementation is an HTTP-API client.
3. Accept interfaces, return concrete structs.
4. Keep interfaces small (interfacebloat enforces a method cap); do not export a fat interface from
   the implementor.
5. Returning an interface from a public function requires a documented justification (the `ireturn`
   linter allows `error`, `empty`, `stdlib`, `generic`, `anon`).
6. Use typed string enums + explicit transitions over FSM/state objects.
7. Prefer composition over behavioral inheritance/embedding for behavior.
8. Wrap domain scalars in named types instead of passing bare primitives across a boundary: an
   identifier/code/email is a `type UserID string` (with a validating constructor where the value has
   invariants), money is minor-unit `int64` (see [[proposal-window]]), never a stringly-typed or
   bare-`int` domain value. Generate an identifier's underlying value with the standard library's
   `uuid` package (`uuid.NewV7` when the id should sort by creation time), not a third-party
   dependency; the named type still wraps it. Avoid primitive obsession and data clumps - a
   recurring group of fields is a named struct.
9. Put the type parameter on the type, or on a package-level function, in that order; a generic
   method (Go 1.27) is the last resort and item 10 sets its bar. If the parameter is fixed for a
   value's whole life, it belongs on the type (`Set[Item]`) and the methods stay ordinary. If the
   operation relates two independent parameters, it is a package-level generic function
   (`slices.Collect`-shaped) - the form the standard library uses, and the one a caller can pass
   around as a value.
10. A generic method is never required. Go scopes visibility to the package, so a function beside
    the type reaches the same unexported state a method does; anything a generic method computes, a
    same-package generic function computes too. All a method buys is call-site form -
    `set.Map(transform)` over `Map(set, transform)` - and that is the whole of its case. Spend it
    only where the type parameter varies per call on an already-constructed value and the type is
    built to read fluently. The form is paid for dearly: an interface cannot declare a generic
    method, so the operation leaves the port model entirely - no consumer-side interface can
    declare it (#2), no test can fake it behind one, no decorator can wrap it, and every caller is
    pinned to the concrete type. An operation that must be reachable through a port is disqualified
    outright, however well it reads as a method.
{{end}}

{{define "forbidden"}}
- `map[string]any` crossing a package boundary.
- Fat interfaces exported from the implementor.
- Returning interface types from public functions without a documented `ireturn` exception.
- `interface{}` instead of `any`.
- FSM state objects where typed-string enums + transitions suffice.
- A bare `string`/`int` carrying a domain concept across a boundary where a named type belongs
  (primitive obsession); a recurring field group passed loose instead of a named struct (data clump).
- A third-party UUID dependency where the standard library's `uuid` package covers the need.
- A generic method where a type parameter on the type, or a package-level generic function,
  expresses the same operation.
- A generic method whose type parameter is fixed for the value's whole life - it belongs on the
  type.
- Designing a port around an operation that needs a generic method - an interface cannot declare one.
{{end}}

{{define "validation"}}
- [ ] No `map[string]any` crossing a package boundary; named structs used instead.
- [ ] Interfaces declared consumer-side, small; no fat implementor-exported interface.
- [ ] Public functions return concretes (or carry a documented `ireturn` exception).
- [ ] Typed enums over FSM objects; composition over behavioral inheritance.
- [ ] Domain scalars are named types (money as minor-unit `int64`); no primitive-obsession or data
  clumps crossing a boundary.
- [ ] Identifier values come from the stdlib `uuid` package, wrapped in a named type.
- [ ] Generic operations are a parameterised type or a package-level generic function by default;
  a generic method appears only where the type parameter varies per call, and nothing reachable
  through a port uses one.
{{end}}
