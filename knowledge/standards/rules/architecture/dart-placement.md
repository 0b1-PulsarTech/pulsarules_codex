---
id: dart-placement
name: Code placement - Dart pubspec workspace
description: A pubspec workspace (root pubspec.yaml workspace field) of Dart packages under packages/<name>/, each exposing one public barrel lib/<name>.dart that exports selected lib/src/** files, everything else in lib/src/ an implementation detail; an executable entry point lives in that package's own bin/, never lib/; a business-capability directory keeps one name across every package/transport it touches. The Dart concrete instantiation of code-placement.
tags:
    - dart
    - layout
---

# Code placement - Dart pubspec workspace

> A pubspec workspace: the root `pubspec.yaml` lists every member under `workspace:`; each member is
> `packages/<name>/`, exposing exactly one public barrel `lib/<name>.dart` that `export`s selected
> `lib/src/**` files, everything else in `lib/src/` staying an implementation detail; an executable
> entry point lives in that package's own `bin/`, never in `lib/`; a business-capability directory
> under `lib/src/<capability>/` carries the same name in every package and transport layer that
> touches it.

Applies to: a Dart project organized as a pubspec workspace (Dart 3.6+ `workspace:` field), one or
more packages under `packages/`. The Dart concrete instantiation of [[code-placement]]'s
language-neutral invariants; see [[code-placement-go]], [[code-placement-monorepo]], and
[[code-placement-inner-modules]] for the Go concrete maps. Test placement follows [[dart-testing]]
(`test/` mirrors `lib/src/`, dropping the `src/` segment); naming and import style follow
[[dart-style]].

{{define "when"}}
- Creating a new package inside a Dart pubspec workspace.
- Deciding whether a new Dart file belongs in `lib/` or `lib/src/`.
- Adding to or reviewing a package's public export barrel.
- Adding an executable (a relay, a codegen tool) to a Dart package.
- Naming a business-capability directory shared across packages or transports.
{{end}}

{{define "must"}}
1. Every workspace member is `packages/<name>/`, listed in the root `pubspec.yaml`'s `workspace:`
   field; each member carries its own `pubspec.yaml` with `resolution: workspace` and depends on a
   sibling package by relative `path:`, never a pub.dev version range.
2. A package exposes exactly one public barrel at `lib/<name>.dart` - a `library;` directive plus
   explicit `export 'src/....dart';` lines for what it publishes. Everything else lives under
   `lib/src/` and is an implementation detail: [[code-placement]]'s non-API-code invariant, enforced
   here by never importing another package's `package:<name>/src/...` path. A second barrel is
   justified only for a genuinely separate boundary the first would force on every consumer (for
   example a `dart:js_interop` surface split into its own `lib/route.dart`), never as a second
   dumping ground.
3. An executable entry point (a relay process, a codegen tool) lives at that package's own
   `bin/<tool>.dart`, wiring the package's `lib/` API together - [[code-placement]]'s entry point,
   kept separate from the library code it wires.
4. A business capability gets one directory name, reused identically under every package's
   `lib/src/<capability>/` it touches and in the transport-layer route/module name for that
   capability - [[code-placement]]'s capability-naming invariant, so one grep finds every layer of a
   feature.
5. Dependency direction across packages is one-way and stated once (for example `web -> transport ->
   domain`); check it with `dart pub deps --json` per package - a domain package importing a
   transport package, or any package importing a UI-framework package it has no business needing, is
   the violation [[dependency-rule]] forbids in the abstract.
{{end}}

{{define "forbidden"}}
- A package with no exported public symbols, forcing every consumer to import `lib/src/...` directly.
- A cross-package dependency pinned as a pub.dev version range instead of a relative `path:`.
- An executable's `main()` under `lib/` instead of `bin/`.
- The same business capability spelled differently across the packages/transports it spans.
- A domain package depending on a transport or UI-framework package.
{{end}}

{{define "validation"}}
- [ ] Every workspace member is under `packages/<name>/`, listed in the root `workspace:` field.
- [ ] Each package exports its public surface from one `lib/<name>.dart` barrel; no cross-package
  `lib/src/...` import.
- [ ] An executable lives under `bin/`, not `lib/`.
- [ ] A shared business capability uses one name across every package/transport it appears in.
- [ ] `dart pub deps --json` (or equivalent) shows no reverse-direction package import.
{{end}}
