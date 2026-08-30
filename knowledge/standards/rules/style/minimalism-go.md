---
id: minimalism-go
name: Code minimalism - Go tooling
description: The Go instantiation of the neutral function-scope minimalism rule - use go_symbol_references (the gopls MCP) to find every caller before patching a shared symbol, and let cyclop/funlen/nestif/unused/unparam machine-check the ladder and dead-code cleanup.
tags:
    - go
    - minimalism
dependencies:
    - effective-go
linters:
    - cyclop
    - funlen
    - nestif
    - unused
    - unparam
---

# Code minimalism - Go tooling

> In a Go project, `go_symbol_references` (the gopls MCP) is the references search that finds every
> caller of a shared symbol before you patch it; `cyclop`/`funlen`/`nestif` machine-check the
> decision ladder's "smallest fix" outcome, and `unused`/`unparam` machine-check dead-code deletion.
> This is the Go instantiation of [[minimalism]]'s neutral contract; run both together.

Applies to: function and method bodies, helper/abstraction/dependency decisions, in a Go project.

{{define "when"}}
- Writing or editing any Go function/method body.
- About to patch a shared symbol that has more than one caller.
- Reviewing a Go diff for bloat / over-engineering / speculative generality.
{{end}}

{{define "must"}}
1. Before patching a shared symbol, run `go_symbol_references` (the gopls MCP) to enumerate every
   caller, then land the fix once at the shared definition instead of at each call site (the Go
   instantiation of [[minimalism]]'s root-cause clause).
2. Keep functions inside the thresholds `cyclop`, `funlen`, and `nestif` enforce (cyclomatic
   complexity, function length, nesting depth); a function that trips one of them has skipped a rung
   of the decision ladder, not found a case the ladder does not cover.
3. Delete dead code: `unused` and `unparam` catch an uncalled symbol or parameter in CI, not just in
   review. Never `//nolint:unused`/`//nolint:unparam` to keep a symbol nothing calls.
{{end}}

{{define "forbidden"}}
- Patching a shared symbol's call sites one by one instead of running `go_symbol_references` and
  fixing the shared definition once.
- `//nolint:unused` or `//nolint:unparam` to retain a dead symbol or parameter.
- A function shape only `cyclop`/`funlen`/`nestif` would have caught, shipped with the linter
  silenced instead of simplified.
{{end}}

{{define "validation"}}
- [ ] A shared-symbol bug fix ran `go_symbol_references` and landed once at the root.
- [ ] `cyclop`, `funlen`, and `nestif` are clean (or a `simplification:` marker names the ceiling).
- [ ] `unused` and `unparam` are clean; no `//nolint:unused`/`//nolint:unparam` retaining dead code.
{{end}}
