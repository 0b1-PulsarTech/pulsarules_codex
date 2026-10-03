---
id: flag-arguments
name: Function arguments - no flags, no output args, small arity
description: No boolean/flag parameters (split the function or take a typed enum), no output/mutating-pointer arguments (mutate the receiver instead), and a small argument count - beyond ~3 collapse the parameters into a named struct.
tags:
    - go
    - style
linters:
    - revive
analyzers:
    - complexity
    - variadic-index
---

# Function arguments - no flags, no output args, small arity

> A boolean parameter that selects behavior is two functions hiding in one - split it or pass a typed
> enum. An output argument the function writes through is a mutation in disguise - mutate the receiver
> or return the value. Keep the argument count small; past ~3, a named parameter struct reads better
> than a positional list and survives reordering.

Applies to: designing a function or method signature. Pairs with [[types]] (named structs at
boundaries) and [[module-boundaries]] (a parameter struct removes positional connascence).

{{define "when"}}
- Adding a parameter to a function or method.
- A signature reaching three-plus positional arguments.
- Tempted to pass a `bool` that picks one of two behaviors.
{{end}}

{{define "must"}}
1. No flag arguments: a `bool` (or other parameter) that selects between two behaviors becomes two
   named functions, or - if the variants share a body - a typed-string enum parameter that names each
   case (never a bare `true`/`false` at the call site).
2. No output arguments: a function mutates its own receiver or returns the new value; it does not take
   a pointer purely to write a result back through it. Constructing-and-returning beats filling-in.
3. Keep arity small (revive `argument-limit`, target ~3). Beyond that, collapse related parameters
   into a named parameter struct so call sites read by field name, not position (see [[types]]).
4. Never index a variadic parameter positionally (`opts[0]`, `names[1]`; `variadic-index`): a position
   inside `...T` is an unnamed flag in disguise. Either take a named options struct, or range over
   the values and treat each alike.
5. A command function returns only `error`; do not add a parameter to also smuggle a status out (see
   [[command-query-separation]]).
{{end}}

{{define "forbidden"}}
- A boolean/flag parameter that selects behavior (split the function or take a typed enum).
- An output/mutating-pointer argument used to return a result.
- A long positional parameter list where a named struct belongs.
- Positional indexing of a variadic parameter (`opts[0]`).
- Bare `true`/`false` literals at a call site standing in for a mode.
{{end}}

{{define "validation"}}
- [ ] No behavior-selecting boolean/flag parameters; variants are split functions or a typed enum.
- [ ] No output arguments; functions mutate the receiver or return the value.
- [ ] Argument count small (revive `argument-limit`); larger sets use a named parameter struct.
- [ ] No variadic parameter is indexed positionally; it is ranged over or replaced by an options struct.
- [ ] No bare `true`/`false` mode literals at call sites.
{{end}}
