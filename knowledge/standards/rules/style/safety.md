---
id: safety
name: Safety - nil interfaces, loop defers, slice aliasing, integer truncation
description: Four classic Go footguns - a typed nil pointer boxed into an interface reading as non-nil, a defer inside a loop body waiting for the function instead of the iteration, append silently reusing a shared backing array, and a narrowing integer conversion truncating without a bounds check.
tags:
    - go
    - style
linters:
    - gosec
analyzers:
    - golangci-lint
---

# Safety - nil interfaces, loop defers, slice aliasing, integer truncation

> A typed nil pointer boxed into an interface is not `== nil`; a `defer` inside a loop body waits
> for the function to return, not the iteration to end; `append` can silently overwrite a slice's
> shared backing array; and a narrowing integer conversion truncates without complaint. None of
> these are syntax errors - each compiles clean and ships a binary that misbehaves on the right
> input.

Applies to: returning a pointer type through an `error`/interface return, deferring inside a
`for`/`range` loop body, handing a derived slice to code that appends, and narrowing integer
conversions.

{{define "when"}}
- Returning a pointer-receiver type through an `error`- or interface-typed return value.
- Writing a `defer` statement inside a `for`/`range` loop body.
- Slicing a value out of another slice and handing it to code that appends.
- Converting an integer to a narrower type (`int64`->`int32`, `int`->`int8`, `uint`->`int`, etc).
{{end}}

{{define "must"}}
1. Never return a typed nil pointer through an interface-typed return (`error` or otherwise); return
   the untyped `nil` literal explicitly instead of a nil `*T` variable boxed into the interface.
2. Never place a `defer` inside a `for`/`range` loop body; extract the loop body into its own
   function (or an immediately-invoked closure) so the deferred call fires at the end of each
   iteration instead of accumulating until the enclosing function returns.
3. When handing a slice derived from another slice to code that appends, cut it with the full slice
   expression `s[:len(s):len(s)]` so the append is forced to allocate a new backing array instead of
   silently overwriting the tail the original slice still shares.
4. Bounds-check a narrowing integer conversion against the target type's max/min before converting;
   never convert on the assumption the value already fits the narrower type.
{{end}}

{{define "forbidden"}}
- Returning a typed nil pointer (`*T`) through an `error`- or interface-typed return value.
- A `defer` statement inside a `for`/`range` loop body.
- Appending to a slice taken from another slice without a full slice expression bounding its
  capacity, where the original slice's tail must not be overwritten.
- A narrowing integer conversion with no bounds check against the target type's range.
{{end}}

{{define "validation"}}
- [ ] No function returns a typed nil pointer through an interface-typed return; nil interface
  returns use the untyped `nil` literal, not a nil `*T` variable.
- [ ] No `defer` sits inside a `for`/`range` loop body; per-iteration cleanup is extracted into a
  function.
- [ ] A slice derived from another slice that is handed to appending code uses the full slice
  expression `s[:len(s):len(s)]` wherever the original's tail must survive.
- [ ] Every narrowing integer conversion is preceded by a bounds check against the target type's
  max/min (gosec `G115`).
{{end}}
