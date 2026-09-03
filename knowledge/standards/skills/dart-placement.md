---
id: dart-placement
name: Dart placement
---

Governs where code lives in a Dart pubspec workspace: every member under `packages/<name>/`, listed
in the root `pubspec.yaml`'s `workspace:` field; one public export barrel per package at
`lib/<name>.dart`, everything else under `lib/src/` an implementation detail; an executable entry
point in that package's own `bin/`, never `lib/`; a business capability keeping one directory name
across every package and transport it touches. Reach for it when creating a package inside a
workspace, deciding whether a file belongs in `lib/` or `lib/src/`, reviewing a package's export
barrel, or naming a capability shared across packages. It is the Dart analogue of this repo's
`code-placement`: the same one-way dependency, non-public-code, capability-naming, and
entry-point-separation invariants, mapped onto a pubspec workspace's own directory shape instead of
Go's `cmd`/`internal`/`pkg`.

The rules below are the composed dart-placement rule.
