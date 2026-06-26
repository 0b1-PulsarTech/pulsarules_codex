---
id: rule-engine
name: Rule engine
---

## Mandatory workflow

1. Keep the lib domain-agnostic: rules read facts only through the `Evaluator` (`Fact(key) (any, bool)`); they never
   touch domain state directly.
2. Define the `Rule` interface (`Run(ctx, Evaluator) (bool, error)` + `Decode(map[string]any) error`) and `Evaluator` in
   the lib; the consuming app defines concrete rule types and registers them.
3. Compose selectors with `AndSelector`/`OrSelector` (Composite); model branching with a `RuleTree` (`on_success`/
   `on_failure` branches and actions).
4. Register rule types in a typed `Registry` of type-keyed constructors at boot (no `switch` in the lib, no reflection).
   The registry is the allow-list; inject it, never a package-level global.
5. Decode stored JSON config into a tree with `rules.DecodeTree(registry, jsonBytes)` at author/save time, not lazily.
   An unknown type anywhere in the tree returns `ErrUnknownRuleType`; persist only after a successful decode.
6. Evaluate with `tree.Evaluate(ctx, evaluator, rules.WithObserver(auditor))`. Action rules call a use case/facade and
   emit events; they never mutate canonical state outside the owning use case.
7. Inject a `RunObserver` to audit why a rule did/didn't fire, and a `Debouncer` to coalesce rapid inputs before
   evaluation.

## Validation checklist

- [ ] Rules read facts via `Evaluator`, not domain state directly.
- [ ] Rule types defined in the consuming app and registered in a `Registry` (no lib `switch`, no `reflect`).
- [ ] Stored trees decoded at author/save time; unknown type anywhere returns `ErrUnknownRuleType`.
- [ ] Action rules call a use case/facade and emit events; they do not mutate state directly.
- [ ] Rule-run audit via an injected `RunObserver` (no global); bursts coalesced via an injected `Debouncer`.

## Forbidden actions

- A hard-coded `switch` over rule types inside the lib.
- Reading domain state directly in a rule (use the `Evaluator`).
- An action rule mutating canonical state outside the owning use case.
- A package-level/global registry; ad-hoc `if` chains instead of decoded rules.
- Persisting a rule tree without decoding it against the registry first.

## Expected outputs

- A domain-agnostic `Rule`/`Evaluator`/`RuleTree`/`Registry` engine.
- Rule types defined by the consuming app and registered in a typed registry.
- Stored trees validated at author time; runs audited via an injected `RunObserver`; bursts coalesced.
