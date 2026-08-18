---
id: code-smells
name: Code smells and their remedies
description: A catalog of code smells that must trigger a refactor before new behavior is layered on, each tied to its Go remedy and the linter that catches it - long function, deep nesting, high complexity, duplication, repeated switch, long parameter list, primitive obsession, global mutable state, dead code.
tags:
    - go
    - refactoring
linters:
    - funlen
    - cyclop
    - nestif
    - unparam
    - dupl
    - goconst
analyzers:
    - control-flow
    - complexity
    - golangci-lint
---

# Code smells and their remedies

> A code smell is a surface symptom of a deeper design problem. When one appears, refactor it away
> BEFORE adding behavior on top - a smell layered under a feature compounds. Each smell below maps to
> a concrete Go remedy and the linter that flags it, so "this needs a refactor" is an objective call,
> not a taste argument.

Applies to: spotting and clearing structural debt. This rule is the trigger list for the
[[refactoring]] workflow; remedies route into [[effective-go]], [[types]], [[module-boundaries]], and
[[design-patterns]].

{{define "when"}}
- A function/file trips `funlen`/`cyclop`/`nestif`/`dupl`.
- Touching code that is hard to read or change before adding to it.
- Reviewing a diff for structural debt.
{{end}}

{{define "catalog"}}
| Smell                | Symptom                                | Remedy                                                       | Catch               |
|----------------------|----------------------------------------|--------------------------------------------------------------|---------------------|
| Long function        | a function past ~80 lines              | extract a named helper per sub-step                          | `funlen`            |
| Deep nesting         | arrow-shaped `if/for`                  | invert to guard clauses / early return                       | `nestif`            |
| High complexity      | too many branches in one func          | decompose the conditional; split the func                    | `cyclop`            |
| Duplicated code      | the same body in 3+ sites              | extract one shared function                                  | `dupl`              |
| Magic value          | a repeated literal                     | a named constant                                             | `goconst`           |
| Repeated switch      | the same type-switch in many sites     | Strategy + typed Registry (see [[design-patterns]])          | review/`dupl`       |
| Long parameter list  | many positional args                   | a named parameter struct (see [[flag-arguments]], [[types]]) | review              |
| Dead code            | unused symbol / commented-out block    | delete it; VCS remembers (see [[minimalism]])                | `unused`, `unparam` |
| Primitive obsession  | bare `string`/`int` for a domain value | a domain type with a validating constructor (see [[types]])  | review              |
| Global mutable state | a mutable package-level `var`          | encapsulate behind a constructor/accessor (see [[startup]])  | `gochecknoglobals`  |
{{end}}

{{define "must"}}
1. When a smell with a linter is present, the linter failure is the trigger - clear it rather than
   suppressing it with `//nolint`.
2. Apply the listed remedy, not a local patch: a repeated switch becomes a Strategy+Registry; a data
   clump becomes a named struct; primitive obsession becomes a domain type.
3. Clear the smell as a behavior-preserving step BEFORE the feature change that exposed it, in its own
   commit (see [[refactoring]]).
4. Review-only smells (feature envy, repeated switch, primitive obsession) still count - name them in
   review even though no linter fires.
5. After the local fixes land, zoom out one level: sweep every file and function the change touched
   for the same shape - naming, error wrapping, guard-clause style - so the diff reads as one hand.
{{end}}

{{define "forbidden"}}
- Suppressing a smell's linter with `//nolint` instead of refactoring.
- Layering new behavior on top of a known smell instead of clearing it first.
- "Fixing" a duplicated/repeated-switch smell by copying the patch into every site.
- Mixing a behavior change into a refactor commit.
- A "refactor" with no tests green between steps.
- Stopping at the first fixed spot while sibling files/functions the change touched keep the old
  naming, error-wrapping, or guard-clause shape.
{{end}}

{{define "validation"}}
- [ ] Smell-linter failures (`funlen`/`cyclop`/`nestif`/`dupl`/`goconst`) cleared, not suppressed.
- [ ] The catalog's remedy applied (Strategy/Registry, named struct, domain type, extraction).
- [ ] The refactor landed before the feature, in its own commit.
- [ ] One hat per edit; no refactor mixed with a behavior change.
- [ ] Tests green (`-race`) between every step.
- [ ] Review-only smells called out even without a linter hit.
- [ ] Every file/function the change touched was swept for the same naming, error-wrapping, and
  guard-clause shape.
{{end}}
