---
id: imports
name: Imports
description: Three-group import layout, std lib unaliased, proto aliased <service>v<n>.
tags:
    - go
    - style
linters:
    - goimports
    - gci
    - importas
    - depguard
---

# Imports

> Three-group import layout, std lib unaliased, proto aliased `<service>v<n>`.

Applies to: every Go file's import block.

{{define "when"}}
- Writing or editing an import block.
- Aliasing a proto/external package.
- Reviewing import hygiene.
{{end}}

{{define "must"}}
1. Three blocks separated by blank lines, in order: standard library, external (third-party),
   this module's own packages.
2. Alias proto packages `<service>v<n>` (e.g. `foov1`, `foov1grpc`) so generated stubs are
   visually distinct; enforce with `importas`.
3. Otherwise avoid aliases; never alias the standard library.
4. No unused imports; no dot imports in production code.
5. Keep the import list deterministic (goimports orders within blocks).
{{end}}

{{define "forbidden"}}
- Aliasing the standard library.
- Unaliased proto imports that collide or read as generic names.
- Dot imports in non-test code; unused imports.
- Mixing the three groups or ordering them inconsistently.
{{end}}

{{define "validation"}}
- [ ] Three grouped blocks (std, external, this module).
- [ ] Proto aliased `<service>v<n>`; std lib unaliased.
- [ ] No unused imports; no dot imports in production code.
{{end}}
