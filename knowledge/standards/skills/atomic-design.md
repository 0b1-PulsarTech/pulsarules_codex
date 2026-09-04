---
id: atomic-design
name: Atomic design for Svelte
---

Governs where a `.svelte` component lives and what it may do: every component under
`web/src/lib/presentation/` belongs to exactly one of atoms, molecules, organisms, or templates, and
the route (`src/routes/<slice>/`) is the fifth level, pages - the only level that reads `data` from
`load()` and decides navigation. Templates are layout-only and hold no state. Files stay under ~150
lines, split along the levels rather than left to grow. Reach for it whenever creating, moving, or
classifying a Svelte component, or when a file under `presentation/` approaches its line-count bound.
It is the atomic-design skill rewritten for Svelte components instead of Flutter widgets.

The rules below are the composed atomic-design rule.
