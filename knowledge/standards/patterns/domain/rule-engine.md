---
id: rule-engine
name: Rule engine (data-driven selection)
description: A domain-agnostic Rule/Evaluator interface; And/Or composites; a RuleTree with branches; a Registry of type-keyed constructors (no switch, no reflection); write-time validation against the registry; injected RunObserver and Debouncer.
tags:
    - go
    - engine
dependencies:
    - dependency-injection
---

# Rule engine (data-driven selection)

> A domain-agnostic `Rule` (`Run`+`Decode`) and `Evaluator` interface; `AndSelector`/`OrSelector`
> composites; a `RuleTree` with `on_success`/`on_failure` branches and actions; a `Registry` of
> type-keyed constructors the consuming app registers (no `switch` in the lib, no `reflection`).
> Write-time validation: the registry is the allow-list, rejecting unknown types across the whole
> tree before persist. An injected `RunObserver` audits rule runs; an injected `Debouncer`
> coalesces rapid inputs.

Reference tools: a rule-engine library.

{{define "when"}}
- Implementing dynamic/stored JSON rule configs (automation, conditions, branching, segments).
- Registering domain-specific rule types for a consuming app.
- Auditing why a rule did/didn't fire.
- Coalescing bursts of inputs before evaluation.
{{end}}

{{define "recipe"}}
The lib is domain-agnostic; rules read facts only through the `Evaluator`:

```go
type Rule interface {
    Run(ctx context.Context, ev Evaluator) (bool, error)
    Decode(map[string]any) error
}

type Evaluator interface {
    Fact(key string) (any, bool)
}
```

The consuming app defines concrete rule types:

```go
type TemperatureAtLeast struct{ Min int }

func (r *TemperatureAtLeast) Decode(m map[string]any) error {
    v, ok := m["min"]
    if !ok { return errors.New("missing min") }
    f, _ := v.(float64)
    r.Min = int(f)
    return nil
}

func (r *TemperatureAtLeast) Run(_ context.Context, ev Evaluator) (bool, error) {
    cur, ok := ev.Fact("temperature")
    if !ok { return false, nil }
    return cur.(int) >= r.Min, nil
}
```

Register at boot (the registry is the allow-list; never a global):

```go
registry := rules.NewRegistry()
registry.Register("temperatureAtLeast", func() rules.Rule { return &TemperatureAtLeast{} })
remy.RegisterInstance(inj, registry)
```

Decode stored config and evaluate (validate at author/save time, not lazily):

```go
tree, err := rules.DecodeTree(registry, jsonBytes) // unknown type anywhere -> ErrUnknownRuleType
if err != nil {
    return fmt.Errorf("decode rule tree: %w", err)
}
matched, err := tree.Evaluate(ctx, evaluator, rules.WithObserver(auditor))
```
{{end}}

{{define "forbidden"}}
- A hard-coded `switch` over rule types inside the lib.
- Reading domain state directly in a rule (use the `Evaluator`).
- An action rule mutating canonical state outside the owning use case.
- A package-level/global registry; ad-hoc `if` chains instead of decoded rules.
- Persisting a rule tree without decoding it against the registry first.
{{end}}

{{define "validation"}}
- [ ] Rules read facts via `Evaluator`, not domain state directly.
- [ ] Rule types defined in the consuming app and registered in a `Registry` (no lib `switch`,
  no `reflect`).
- [ ] Stored trees decoded at author/save time; unknown type anywhere returns `ErrUnknownRuleType`.
- [ ] Action rules call a use case/facade and emit events; they do not mutate state directly.
- [ ] Rule-run audit via an injected `RunObserver` (no global).
{{end}}
