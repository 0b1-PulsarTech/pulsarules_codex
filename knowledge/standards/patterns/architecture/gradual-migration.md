---
id: gradual-migration
name: Gradual migration (type alias)
description: Relocate an exported type across a package boundary without a flag-day commit - a type OldName = newpkg.NewName alias at the old path carrying a Deprecated doc-comment marker, callers migrated incrementally, the alias deleted once nothing references it. Optional - reach for it only when the blast radius makes a single-commit move risky.
tags:
    - go
    - architecture
    - refactoring
dependencies:
    - module-boundaries
    - code-placement
---

# Gradual migration (type alias)

> Move an exported type to its new package home, leave a `type OldName = newpkg.NewName` alias with a
> `// Deprecated: use newpkg.NewName` marker at the old path, migrate callers incrementally, then
> delete the alias. This is the HOW for a relocation [[module-boundaries]] or [[code-placement]] already
> calls for; it is OPTIONAL - a small move with few callers is still a single commit.

{{define "when"}}
- [[module-boundaries]] or [[code-placement]] calls for relocating an exported type/function and the
  caller count makes one flag-day commit hard to review.
- NOT for a small move with few callers - move it in one commit instead.
{{end}}

{{define "recipe"}}
1. Implement the type at its new home, fully.
2. At the OLD path, leave an alias, never a second copy:

```go
package oldpkg

import "path/to/newpkg"

// Deprecated: use newpkg.NewName.
type OldName = newpkg.NewName
```

   `// Deprecated: <replacement>` is Go's own doc-comment convention (godoc and the `staticcheck`
   SA1019 check both recognize it); this repo adopts it here to mark the OLD path during a migration,
   not as a general deprecation policy.

   A function has no alias form - forward it instead, with the same marker:

```go
// Deprecated: use newpkg.NewFunc.
func OldFunc(a A) (B, error) { return newpkg.NewFunc(a) }
```

3. Commit the move plus the alias/forward together - both paths compile and behave identically, so
   this stays a pure move (see [[git-history]] on move purity).
4. Migrate callers in small batches, one commit per batch: swap `oldpkg.OldName` for `newpkg.NewName`
   at each call site. Each batch is independent, reviewable, and green on its own.
5. Once no caller references the old path (grep, or `staticcheck` flagging every remaining use via
   SA1019), delete the alias in its own commit.
{{end}}

{{define "forbidden"}}
- Copying the implementation to the new path and leaving the old path as a second, diverging body -
  it must be an alias/forward, not a fork.
- An alias or forward with no `// Deprecated:` marker - callers and tooling lose the migration signal.
- Migrating every caller in the same commit as the move - that defeats the alias; do a flag-day move
  instead when the blast radius is small enough for one commit.
- Leaving the alias in place once every caller has migrated.
- Reaching for this pattern by default on a small move with few callers.
{{end}}

{{define "validation"}}
- [ ] The old path holds an alias (`type OldName = newpkg.NewName`) or a one-line forward, never a
  duplicated implementation.
- [ ] The alias/forward carries a `// Deprecated: use newpkg.NewName` doc comment.
- [ ] The move and its alias landed as one commit; each caller-migration batch is its own commit.
- [ ] The alias was deleted once no caller referenced the old path.
- [ ] The blast radius - not habit - justified using this pattern instead of a single-commit move.
{{end}}
