---
id: fitness-functions
name: Architecture fitness functions
description: Every architectural invariant is encoded as an automated, objective check that runs in CI - package-cycle detection, layer/import-direction rules, and complexity budgets via go-arch-lint/depguard/go-test - so structure fails the build instead of relying on reviewer vigilance.
tags:
    - go
    - architecture
    - build
dependencies:
    - build
linters:
    - depguard
    - go-arch-lint
---

# Architecture fitness functions

> An architecture fitness function is an automated, objective test of an architectural characteristic.
> Every structural invariant the handbook states - import direction, layer boundaries, no package
> cycles, complexity budgets - must be encoded as a machine check wired into the Taskfile lint target
> and the CI gate, so a violation fails the build rather than depending on a reviewer remembering it.

Applies to: governing architecture over time. Turns the principles in [[code-placement]],
[[dependency-rule]], [[module-boundaries]], and [[frameworks-as-plugins]] from prose into gates. The
checks live in the build tooling described by [[build]].

{{define "when"}}
- Adding or changing an architectural invariant (a boundary, a layer, an allowed import set).
- Standing up or extending the lint/test gate in CI.
- Reviewing whether a stated rule is actually enforced or only documented.
{{end}}

{{define "must"}}
1. Every architectural invariant has a corresponding automated check; if a rule cannot be machine
   checked, record WHY (review-only) instead of leaving it silently ungoverned.
2. Encode the checks with Go-native tooling: `depguard` for allowed/denied import sets (layer and
   framework boundaries), `go-arch-lint` (or an equivalent `go test` that walks the import graph) for
   package-cycle detection and component dependency rules, `cyclop`/`funlen` for complexity budgets.
3. Wire every fitness function into the ONE Taskfile lint target and run it in CI as a build-failing
   gate - not a warning, not a nightly job a human reads (see [[build]]).
4. A fitness function is committed code reviewed like any other: new boundaries land WITH the check
   that enforces them, in the same change.
5. Keep each check objective and fast (the gate runs on every push); a flaky or subjective "fitness
   function" is not one - delete it or make it deterministic.
{{end}}

{{define "forbidden"}}
- Stating an architectural rule with no automated check and no recorded review-only justification.
- Boundary rules enforced only by code review or a wiki page.
- Architecture checks that only warn, or run outside the build-failing CI gate.
- A new layer/boundary merged without the fitness function that guards it.
{{end}}

{{define "validation"}}
- [ ] Each architectural invariant maps to a depguard/go-arch-lint/go-test/complexity check.
- [ ] Checks run in the Taskfile lint target and the CI gate as build-failing.
- [ ] Package-cycle detection and import-direction rules are present.
- [ ] A new boundary lands with its enforcing fitness function in the same change.
- [ ] Review-only invariants are explicitly recorded as such, not silently ungoverned.
{{end}}
