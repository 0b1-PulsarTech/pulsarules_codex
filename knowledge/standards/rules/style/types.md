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
{{end}}

{{define "must"}}
1. Use named structs at package boundaries; `map[string]any` only at the raw I/O edge, translated
   to a typed value immediately.
2. Declare interfaces in the **consumer** package with the smallest method set the consumer needs, and
   name the port for its ROLE (`Repository`/`Store`/`Resolver`/`Sender`), never a generic `Port` or a
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
   bare-`int` domain value. Avoid primitive obsession and data clumps - a recurring group of fields is
   a named struct.
{{end}}

{{define "forbidden"}}
- `map[string]any` crossing a package boundary.
- Fat interfaces exported from the implementor.
- Returning interface types from public functions without a documented `ireturn` exception.
- `interface{}` instead of `any`.
- FSM state objects where typed-string enums + transitions suffice.
- A bare `string`/`int` carrying a domain concept across a boundary where a named type belongs
  (primitive obsession); a recurring field group passed loose instead of a named struct (data clump).
{{end}}

{{define "validation"}}
- [ ] No `map[string]any` crossing a package boundary; named structs used instead.
- [ ] Interfaces declared consumer-side, small; no fat implementor-exported interface.
- [ ] Public functions return concretes (or carry a documented `ireturn` exception).
- [ ] Typed enums over FSM objects; composition over behavioral inheritance.
- [ ] Domain scalars are named types (money as minor-unit `int64`); no primitive-obsession or data
  clumps crossing a boundary.
{{end}}
