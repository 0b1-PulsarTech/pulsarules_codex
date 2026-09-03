---
id: dart-style
name: Dart style
---

Dart style governs the baseline shape of every Dart file in a project where Dart is the authorial
language: naming (packages, files, identifiers), three-group imports (`dart:`, external `package:`,
workspace `package:`), `const` constructors on immutable value types, `sealed` hierarchies for a
closed set of variants, and no `dynamic` anywhere - `Object?` plus a single cast at the boundary
instead. Reach for it whenever you write or edit any `.dart` file, name a package or file, write an
import block, or model a closed set of variants (a domain-error hierarchy, a result type). It is the
Dart analogue of this repo's `go-style`: the floor beneath every other Dart skill, deciding how code
is named, imported, and shaped, never architecture or placement.

The rules below are the composed dart-style rule.
