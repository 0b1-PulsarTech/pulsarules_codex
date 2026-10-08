---
id: dart-style
name: Dart style
description: Dart naming, three-group imports (dart:/package:/relative), const constructors on immutable value types, sealed hierarchies for closed variant sets, and no dynamic - Object? with one cast at the boundary instead.
tags:
    - dart
    - style
references:
    - effective-dart
---

# Dart style

> Dart naming (packages, files, identifiers), three-group imports (`dart:`, `package:`, relative),
> `const` constructors on immutable value types, `sealed` hierarchies for a closed set of variants,
> and no `dynamic` anywhere - `Object?` plus one cast at the boundary instead.

Applies to: every `.dart` file in a project where Dart is the authorial language and a non-Dart shell
(for example SvelteKit) only hosts it. The Dart analogue of [[effective-go]] and [[naming]].

{{define "when"}}
- Writing or editing any `.dart` file.
- Naming a package, directory, file, class, or member.
- Writing or reviewing an import block.
- Choosing between `Object?` and `dynamic` at a JSON, JS-interop, or FFI boundary.
- Modeling a closed set of variants (a domain-error hierarchy, a result type, a panel state).
{{end}}

{{define "must"}}
1. Never write `dynamic`. A value whose type is not known until runtime is `Object?`; the code that
   receives it performs exactly one cast (or a `switch`/`is` check) at the boundary where the value
   enters typed Dart, and every line past that boundary is fully typed.
2. Group imports into three blocks, separated by one blank line, each block alphabetized: `dart:`
   (standard library) first, every `package:` import (pub.dev dependencies and this workspace's own
   packages alike) second, relative imports last. This is the order the `directives_ordering` lint
   checks. Multiple imports from the same package stay together with no blank line between them; a
   different package starts a new line in the same block, not a new block.
3. Give an immutable value type a `const` constructor and mark the class `@immutable`. A type with no
   mutable field and no reason to allocate more than one identical instance is a `final class` with a
   `const` constructor, not a mutable one constructed fresh every time.
4. Model a closed set of variants as a `sealed class` with each variant a `final class` extending it,
   never a payload-carrying enum or an open (non-`sealed`) base class standing in for one. Every
   `switch` over a `sealed` hierarchy is exhaustive: no `default` case, so the compiler forces every
   call site to handle a variant added later.
5. Package and directory names are lowercase, a single evocative word, no underscore, and carry no
   brand prefix (a retired prefix such as `tkt_` does not return). A value that crosses a JSON or
   JS-interop boundary with an implicit unit (money, a scaled quantity) carries that unit in its field
   name (`averageTicketMinor`), because a bare `number` on the other side of the boundary carries no
   scale of its own.
6. File names are `snake_case.dart`, matching the primary symbol the file defines; a
   `<name>_test.dart` colocates with `<name>.dart` (see [[dart-testing]]).
7. A file name the framework dictates is exempt from rule 6 and is never "corrected". SvelteKit
   discovers routes by file name, so `+page.server.dart`, `+layout.server.dart`, `+server.dart` and
   a `[id]/` dynamic-parameter directory are required spellings - renaming `+page.server.dart` to
   `page_server.dart` does not rename a route, it stops the route from loading at all. The same
   class covers `_app.tsx`, `[slug].tsx`, `__init__.py` and `Package.swift`.
{{end}}

{{define "forbidden"}}
- `dynamic` anywhere in application code, including a generic type argument. Where an external
  package's own public API leaves no other spelling for a type argument, the call site is isolated in
  its own file with a comment naming the package constraint - it is never a default choice, and never
  spreads past that one call site.
- An import block that mixes `dart:`, `package:` and relative origins in one paragraph, skips the
  blank-line separation between them, or leaves a block unsorted.
- A mutable class standing in for a value type that has no mutable field.
- A payload-carrying enum, or an open (non-`sealed`) class hierarchy, used where a closed set of
  variants is meant; a `default` case on a `switch` over a `sealed` hierarchy.
- A `tkt_`-prefixed or underscored package name; a scaled numeric value crossing a JSON/JS boundary
  with no unit named in its field.
{{end}}

{{define "validation"}}
- [ ] No `dynamic` in the diff; every runtime-typed value is `Object?` cast once at its boundary.
- [ ] Imports are three blank-line-separated, alphabetized groups: `dart:`, then all `package:`,
  then relative.
- [ ] Immutable value types are `@immutable` with a `const` constructor.
- [ ] A closed variant set is `sealed`, every variant `final`, every `switch` over it exhaustive with
  no `default`.
- [ ] Package/directory names are lowercase, single-word, unprefixed; a boundary-crossing scaled value
  names its unit in the field.
- [ ] Files are `snake_case.dart`; a `_test.dart` colocates with the source file it tests.
- [ ] No framework-dictated file name (`+page.server.dart`, `[id]/`) was renamed to satisfy the
  `snake_case.dart` rule.
{{end}}
