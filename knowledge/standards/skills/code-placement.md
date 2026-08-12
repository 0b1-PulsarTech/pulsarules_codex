---
id: code-placement
name: Code placement
---

Code placement governs which top-level directory a new file or module belongs in, and the
one-way dependency direction between them - apps/libs/tools/build in a monorepo, cmd/internal/pkg
in a single module. Reach for it when creating a new module, app, or lib, deciding where a file
belongs, or adding a cross-module dependency. It does not prescribe the internal layout of a
single use case or feature (see usecase-layout) - only the boundary a file crosses and which
direction imports are allowed to flow.

The rules below are the composed code-placement rule.
