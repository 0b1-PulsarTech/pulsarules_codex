---
id: refactoring
name: Refactoring
---

Governs behavior-preserving cleanup: wear one hat at a time - refactor XOR add behavior, never both
in the same edit - take small steps with tests green between each, and land the refactor as its own
commit separate from any feature. Reach for this when a function or file trips
`funlen`/`cyclop`/`nestif`/`dupl`, or code needs cleanup before new behavior lands on top of it. Not
a design skill: codebase-design and design-patterns say what "better" looks like, this skill governs
the safe path to get there, one small verified step at a time.

The rules below are the composed refactoring rule.
