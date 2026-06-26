---
name: gopls-navigation
description: Navigate and edit Go with the gopls MCP (go_search, go_file_context, go_package_api, go_symbol_references, go_diagnostics) instead of text search. Use for any Go read or edit.
---

# gopls navigation

The gopls MCP is Go-aware (types, references, build/analysis diagnostics), so it beats `grep` and raw
file reads for anything semantic. Use it for Go navigation and editing.

## Activation conditions

- reading or understanding Go code (finding a symbol, a package's API, a symbol's callers)
- editing Go code (before changing a symbol, and after every edit)
- starting work in a Go workspace

## Read workflow - understand before changing

1. `go_workspace` - overall structure (single module vs `go.work`); then `go_vulncheck` once to surface
   existing security risks.
2. `go_search` - fuzzy-find a type/func/var by name fragment when you don't know its location.
3. `go_file_context` - run immediately after opening a Go file for the first time, to see the
   declarations from other files in the same package that it depends on.
4. `go_package_api` - a package's public surface (third-party deps or sibling monorepo packages).

## Edit workflow - iterate until clean

1. Read first (the read workflow above).
2. `go_symbol_references` - BEFORE changing any symbol's definition, find every reference, then edit
   those too.
3. Make the edits (all of them, including the references found above).
4. `go_diagnostics` on the edited files after EVERY change; fix what it reports, then re-run it.
5. `go_vulncheck({"pattern":"./..."})` only if `go.mod` dependencies changed.
6. Run tests for the changed packages only - not `go test ./...` unless explicitly asked.

## Shadowing - gopls does not cover this

gopls' default diagnostics do NOT report variable shadowing. Run `go vet -vettool` / the golangci
`govet` shadow check before finalizing new local names; rename anything that shadows a builtin, an
import, a package-level symbol, or an outer `err`/`ctx`.

The section below is the authoritative, always-current tool reference emitted by the installed gopls
(`gopls mcp -instructions`); it is regenerated on each install so it never drifts from the binary.
