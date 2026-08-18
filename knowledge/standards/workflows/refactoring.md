---
id: refactoring
name: Refactoring
description: Behavior-preserving cleanup as a disciplined loop - one hat at a time (refactor XOR add behavior), small steps with green tests between each, refactors landed as their own commits separate from features.
tags:
    - go
    - workflow
    - refactoring
steps:
    - put on one hat - refactoring XOR adding behavior, never both at once
    - identify the smell from the code-smells catalog (or a failing structure linter)
    - gate on blast-radius coverage - high refactor freely, medium test the disturbed paths first, low write characterization tests first
    - make one small behavior-preserving change
    - run go test ./... -race; it must stay green before the next step
    - repeat until the smell is gone; squash the micro-steps into one reviewable refactor commit
    - zoom out and sweep every touched file/function for the same naming, error-wrapping, and guard-clause shape
    - land the refactor as its own commit, separate from any feature change
composes_rules:
    - code-smells
    - effective-go
    - testing
    - commits
---

# Refactoring workflow

> Improve the design of existing code without changing its behavior, in small verified steps. Wear
> one hat at a time: either you are refactoring (structure changes, behavior fixed) or adding behavior
> (tests change) - never both in the same edit. Tests stay green between every step, and the refactor
> lands as its own commit so review can read structure and behavior separately.

## When to use

- Clearing a [[code-smells]] entry before layering a feature on top (preparatory refactoring).
- A structure linter (`funlen`/`cyclop`/`nestif`/`dupl`) fails on code you are about to extend.
- Comprehension refactoring: renaming/extracting to understand code you are reading.

## Steps

1. **One hat.** Decide: this edit refactors (behavior fixed) or adds behavior (tests change). Never
   mix them - a refactor that also changes behavior is no longer behavior-preserving.
2. **Name the smell.** Identify it from the [[code-smells]] catalog, or take the failing structure
   linter as the trigger. Pick the catalog's remedy, not a local patch.
3. **Gate on blast-radius coverage.** Measure coverage of the blast radius - the files and functions
   this change will touch - never the whole repo. Run it scoped to the touched package(s):
   `go test ./<touched-pkg>/... -coverprofile=cover.out && go tool cover -func=cover.out`.
   Gate the next step on that number:
   - **High (>= 80% statement coverage on the touched paths):** refactor directly - the small steps
     and existing tests are the safety net.
   - **Medium (40-79%):** add tests for the specific paths you are about to disturb first, then
     refactor.
   - **Low (< 40%, or the touched code has no test at all):** write characterization tests that pin
     current behavior - including current bugs - before touching the code; only then refactor.

   80%/40% mark well-tested / partially-tested / effectively-untested, so the new-test effort tracks
   the actual risk of the change instead of a blanket "write more tests" instinct.
4. **Small step.** Make one behavior-preserving change (extract a function, invert to guard clauses,
   introduce a named struct/domain type, replace a repeated switch with Strategy+Registry).
5. **Stay green.** Run `go test ./... -race`; it must pass before the next step. A red bar means the
   step changed behavior - back it out (see [[testing]]).
6. **Repeat**, then squash the local micro-steps into one reviewable commit.
7. **Zoom out.** Sweep every file/function the change touched for the same shape - naming, error
   wrapping, guard-clause style - so the diff reads as one hand (see [[code-smells]]).
8. **Commit separately.** Land the refactor as its own emoji-prefixed commit, never folded into a
   feature change (see [[commits]]).

## Conflict resolution

If a refactor is needed to make a feature clean, do the refactor first as a separate commit, then add
the feature on the cleaned structure - preparatory refactoring, not a mixed commit.

## References

- rules: [[code-smells]], [[effective-go]], [[testing]], [[commits]]
- skill: `refactoring`
