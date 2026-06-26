---
id: template-engine
name: Template engine (variable registry)
description: A variable registry built from stored VariableDef rows; lazy Resolver lookups; render via text/template with missingkey=error; hidden variables resolved separately; per-target adapters for positional external formats.
tags:
    - go
    - engine
---

# Template engine (variable registry)

> A variable registry built from stored `VariableDef` rows (names are author-chosen data, never
> hardcoded enums); lazy `Resolver` lookups; render via `text/template` with `missingkey=error`;
> hidden/metadata variables resolved separately and never inlined into the visible body. The engine
> is a library; the targets (system render vs a positional/structured external format) are
> per-target adapters.

Reference tools: `text/template`; a variable-registry engine package.

{{define "when"}}
- Building user-defined template authoring/rendering.
- Rendering text from a variable registry (notifications, drafts, composed bodies).
- Compiling a template to a positional external format (e.g. an HSM with numbered slots).
{{end}}

{{define "recipe"}}
Variables are DATA, not code:

```go
defs := []tplate.VariableDef{
    {Name: "customer_name", Type: tplate.VarString, Source: "customer.name"},
    {Name: "amount", Type: tplate.VarCurrency, Source: "budget.amount"},
}
```

Build the engine from defs + a resolver:

```go
resolver := tplate.MapResolver(map[string]any{
    "customer.name": "Ada",
    "budget.amount": money.FromMinor(1_500_00, money.BRL),
})
engine := tplate.New().FromDefs(defs, resolver)
```

System render (validate at save; render at send; missing var is an error):

```go
vars, err := engine.ApplyTemplateOnMessage(ctx, &body) // missingkey=error
if err != nil {
    return fmt.Errorf("render template: %w", err)
}
```

Positional external target (named `{{"{{"}}vars}}` -> positional `{{"{{"}}1}},{{"{{"}}2}}`; reject conditions/logic
the target does not support):

```go
compiled := tplate.CompileToPositional(body, defs)
```

Hidden variables resolved separately, never inlined:

```go
hidden, err := engine.RetrieveHiddenVariables(ctx, []tplate.HiddenVariable{{"{{"}}Name: "tracking_id", ...}})
```
{{end}}

{{define "forbidden"}}
- Hardcoding a variable enum or message text in Go.
- Resolving variables by reflection (use an explicit `Resolver.Resolve(source)`).
- Conditions/logic on a target that does not support them.
- Inlining hidden variables into the visible body; eager callback evaluation.
- Global/package-level engine caches.
{{end}}

{{define "validation"}}
- [ ] Variables loaded from stored rows; no hardcoded variable enum or message text.
- [ ] Engine built via `New().FromDefs`; resolution lazy; missing var surfaces as a render error.
- [ ] Positional targets reject unsupported conditions/logic.
- [ ] Hidden variables resolved via `RetrieveHiddenVariables`, never inlined.
{{end}}
