---
id: atomic-design
name: Atomic design for Svelte
description: Every component under web/src/lib/presentation/ belongs to exactly one level (atoms, molecules, organisms, templates) plus the route as the pages level; the route is the only level that reads data from load() and navigates; templates are layout-only with no state; files stay under ~150 lines.
tags:
    - svelte
    - architecture
---

# Atomic design for Svelte

> Every component lives in `web/src/lib/presentation/` and belongs to exactly one level: atoms,
> molecules, organisms, or templates. The route (`src/routes/<slice>/`) is the fifth level, pages, and
> the only level that reads `data` from `load()` and decides navigation. Files stay under ~150 lines;
> split along the levels above that bound, not inside one file.

Applies to: `.svelte` components in a project where SvelteKit is the UI/SSR shell over an authorial
domain language (Dart via `+page.server.dart`, or another server-side language). Reach for this
whenever creating, moving, or classifying a component.

{{define "when"}}
- Creating a new `.svelte` component.
- Deciding which directory under `web/src/lib/presentation/` a component belongs in.
- An organism reaches for state, data, or navigation it should not own.
- A file under `presentation/` grows toward or past ~150 lines.
{{end}}

{{define "must"}}
1. Classify a component before writing its first line, into exactly one of five levels:

   | Level | Path | Rule |
   |---|---|---|
   | Atoms | `atoms/` | Single responsibility, no business logic. |
   | Molecules | `molecules/` | A combination of atoms serving one UI function. |
   | Organisms | `organisms/` | A complete screen section; receives callbacks, does not own navigation. |
   | Templates | `templates/` | Layout only: arranges organisms; no state, no data access. |
   | Pages | `src/routes/<slice>/` | The route; reads `data` from `load()`; the only level that navigates. |
2. An organism does not depend on another organism when a molecule already solves the composition -
   reach for the smaller level first.
3. A template is the only level that defines screen structure (the arrangement of organisms), and it
   holds no state of its own - a template that needs a value takes it as a prop from the page that
   uses it.
4. The route is the only level that reads `data` returned by its `load()` and the only level that
   decides where to navigate next; an atom, molecule, organism, or template never calls a
   navigation API and never reads the route's `data` directly - it receives what it needs as props or
   callbacks from the page above it.
5. Keep a component file at or under ~150 lines. When a component grows past that, split it along the
   atomic levels above (extract an organism's sections into molecules, a molecule's pieces into
   atoms) rather than growing one file.
6. One component per file, named for its role, never its screen position (`LeadStatusBadge`, not
   `TopRightBadge`).
{{end}}

{{define "forbidden"}}
- A component placed outside `web/src/lib/presentation/`, or inside it but not classified into one of
  the five levels.
- An organism importing another organism where a molecule would do.
- A template that holds `$state` or reads data itself instead of receiving it as a prop.
- Any level other than the route reading `load()`'s `data` or calling a navigation API directly.
- A `presentation/` component file that grows past ~150 lines without being split along atomic levels.
- A component file holding more than one component, or a component named by screen position instead
  of role.
{{end}}

{{define "validation"}}
- [ ] Every component sits under `web/src/lib/presentation/` in exactly one level's directory (or is
  the route itself).
- [ ] No organism depends on another organism where a molecule would resolve it.
- [ ] Templates hold no state and read no data; they only arrange organisms via props.
- [ ] Only the route reads `load()`'s `data` and calls navigation.
- [ ] No `presentation/` file exceeds ~150 lines; a growing file was split along atomic levels.
- [ ] One component per file, named for its role.
{{end}}
