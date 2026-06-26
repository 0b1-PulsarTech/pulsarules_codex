---
id: refactoring
name: Refactoring
---

## Mandatory workflow

1. Wear ONE hat per edit: refactoring (behavior fixed, structure changes) XOR adding behavior (tests
   change). Never both in the same change.
2. Identify the smell from the code-smells catalog, or take a failing structure linter
   (`funlen`/`cyclop`/`nestif`/`dupl`/`goconst`) as the trigger.
3. Apply the catalog's remedy, not a local patch: extract a function, invert to guard clauses,
   introduce a named parameter struct or a domain type, replace a repeated switch with a typed
   Strategy + Registry, delete dead/commented-out code, encapsulate global mutable state.
4. Make ONE small behavior-preserving step at a time; run `go test ./... -race` green before the next.
5. Refactor BEFORE the feature it enables (preparatory refactoring); squash micro-steps and land the
   refactor as its own commit, separate from any behavior change.

## Validation checklist

- [ ] One hat per edit; no refactor mixed with a behavior change.
- [ ] Smell named from the catalog (or a structure linter); the listed remedy applied.
- [ ] Structure-linter failures cleared, not `//nolint`-suppressed.
- [ ] Tests green (`-race`) between every step.
- [ ] Refactor landed as its own commit, separate from features.

## Forbidden actions

- Mixing a behavior change into a refactor commit.
- Suppressing a smell's linter with `//nolint` instead of clearing it.
- Layering new behavior on a known smell instead of refactoring first.
- A "refactor" with no tests green between steps.

## Expected outputs

- Behavior-preserving cleanups in small verified steps, tests green throughout.
- Smells cleared via their catalog remedy (extraction, guard clauses, named struct/domain type,
  Strategy+Registry, dead-code deletion).
- Refactors committed separately from feature changes.
