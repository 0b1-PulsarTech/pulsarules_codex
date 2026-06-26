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
    - make one small behavior-preserving change
    - run go test ./... -race; it must stay green before the next step
    - repeat until the smell is gone; squash the micro-steps into one reviewable refactor commit
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

1. **One hat.** Decide: this edit refactors (behavior fixed) OR adds behavior (tests change). Never
   mix them - a refactor that also changes behavior is no longer behavior-preserving.
2. **Name the smell.** Identify it from the [[code-smells]] catalog, or take the failing structure
   linter as the trigger. Pick the catalog's remedy, not a local patch.
3. **Small step.** Make one behavior-preserving change (extract a function, invert to guard clauses,
   introduce a named struct/domain type, replace a repeated switch with Strategy+Registry).
4. **Stay green.** Run `go test ./... -race`; it must pass before the next step. A red bar means the
   step changed behavior - back it out (see [[testing]]).
5. **Repeat**, then squash the local micro-steps into one reviewable commit.
6. **Commit separately.** Land the refactor as its own emoji-prefixed commit, never folded into a
   feature change (see [[commits]]).

## Conflict resolution

If a refactor is needed to make a feature clean, do the refactor FIRST as a separate commit, then add
the feature on the cleaned structure - preparatory refactoring, not a mixed commit.

## References

- rules: [[code-smells]], [[effective-go]], [[testing]], [[commits]]
- skill: `refactoring`
