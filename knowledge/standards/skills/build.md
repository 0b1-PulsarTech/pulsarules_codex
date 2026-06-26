---
id: build
name: Build & tooling
---

Build and tooling governs how the project compiles, lints, generates code, and ships: the
Taskfile as the single entrypoint, CGO-free defaults, reproducible codegen, and digest-pinned
images. Reach for it when adding or changing a Taskfile target, linter config, or build tag,
running code generation, or building a binary or container image. It is not concerned with what
the generated code contains - an ent schema, a sqlc query, see database-persistence - only with
the pipeline that produces and ships it.

The rules below are the composed build rule.
