---
id: named-returns
name: Named results - when to name them, and how to assign them
description: Name a function's results when a defer reads or writes them, when two results share a type, or when the signature alone would not say what comes back - and never re-bind a parameter or a named result with := when var plus = says it plainly.
references:
    - effective-go
    - code-review-comments
tags:
    - go
    - style
linters:
    - nakedret
analyzers:
    - named-returns
    - shadowing
---

# Named results - when to name them, and how to assign them

> A named result is documentation the compiler keeps honest, and the only handle a `defer` has on
> what the function returns. Name results when they carry that weight - not to save a `var` in the
> body, and never so a `return` can go naked in a long function.

Applies to: designing a function or method signature, and to any `:=` that names a parameter or a
result. Pairs with [[effective-go]] (no naked returns in long functions) and [[transactions]] (the
commit/rollback defer keyed on the named error).

{{define "when"}}
- Choosing a function or method signature that returns more than one value.
- Adding a `defer` that must inspect or replace what the function returns.
- Writing a `:=` whose left side names a parameter, a receiver, or a named result.
{{end}}

{{define "must"}}
1. Name the results when a `defer` reads or writes them: the commit-on-success / rollback-on-error
   finisher, a `recover` that turns a panic into an error, a span that records the returned error.
   Without a name the `defer` cannot reach the value at all.
2. Name the results when two or more share a type - `(min, max int)`, `(host, port string)`,
   `(width, height int)`. The type no longer tells them apart, so only the name does, and a caller
   reading the signature must not have to guess the order.
3. Name the result when the signature alone would not say what comes back at an API boundary -
   `(claims *JaneToken, err error)` over `(*JaneToken, error)`. Godoc shows the signature, not
   the body.
4. Assign a result, never re-declare it: a `:=` that names a parameter, a receiver, or a named
   result reassigns it silently, because Go puts all of them in the function's own block. Declare
   the genuinely new variable with `var` above and assign with `=`, so the reassignment is visible:
   `var span trace.Span` then `ctx, span = tracer.Start(ctx, "...")`.
5. Return values explicitly. A named result documents; it does not license a naked `return` outside
   a short function (`nakedret`).
6. Do not name a result merely to pre-declare a working variable for the body. A name assigned in
   ten places and returned once is an accumulator wearing a result's clothes - declare it in the
   body instead.
{{end}}

{{define "examples"}}
A `defer` needs the name to do its job:

```go
func (r Repo) Insert(ctx context.Context, in Lead) (out Lead, err error) {
    tx, err := r.base.BeginTx(ctx)
    if err != nil {
        return Lead{}, fmt.Errorf("begin: %w", err)
    }
    defer func() { err = tx.Finish(err) }() // only reachable because err is named

    // ...
    return out, nil
}
```

Two results of one type: the names are the documentation.

```go
func (w Window) Bounds() (low, high int64) // not (int64, int64)
```

Assign, do not re-declare. Both forms compile; only the second says what happens:

```go
ctx, span := tracer.Start(ctx, "repo.Insert") // ctx is reassigned, and nothing says so

var span trace.Span
ctx, span = tracer.Start(ctx, "repo.Insert")  // the reassignment is on the page
```
{{end}}

{{define "forbidden"}}
- A `defer` that must touch the result of a function whose results are unnamed.
- Two or more unnamed results of the same type, where only order tells them apart.
- A `:=` that re-binds a parameter, a receiver, or a named result - use `var` plus `=`.
- A naked `return` outside a short function, named results or not (`nakedret`).
- Results named only to pre-declare body variables, or to shorten a `return`.
{{end}}

{{define "validation"}}
- [ ] Results are named wherever a `defer` reads or writes them.
- [ ] No two unnamed results share a type; the ambiguous ones carry names.
- [ ] API-boundary results whose meaning the type does not carry are named.
- [ ] No `:=` re-binds a parameter, a receiver, or a named result; `var` plus `=` is used instead.
- [ ] Every `return` is explicit; no naked returns outside short functions.
- [ ] No result named purely to pre-declare a body variable.
{{end}}
