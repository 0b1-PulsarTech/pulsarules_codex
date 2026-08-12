---
id: effective-go
name: Effective Go subset
description: The enforceable Effective Go / Code Review Comments baseline every Go file must meet.
references:
    - effective-go
    - code-review-comments
    - go-stdlib
tags:
    - go
    - style
linters:
    - gofumpt
    - goimports
    - golines
    - govet
    - cyclop
    - funlen
    - nestif
    - nakedret
    - modernize
    - predeclared
    - godoclint
    - forbidigo
analyzers:
    - file-size
    - text-markers
    - typographic-markers
    - top-of-file
    - big-comment
    - shadowing
---

# Effective Go subset

> The enforceable Effective Go / Code Review Comments baseline every Go file must meet.

Applies to: all Go code. Canonical references: [Effective Go](https://go.dev/doc/effective_go),
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Target Go version: track the
newest stable the module pins (source repos use `go 1.26`).

{{define "when"}}
- Writing or editing any `.go` file.
- Choosing control-flow shape, doc comments, or stdlib idioms.
- Deciding whether to hand-roll something the stdlib already provides.
{{end}}

{{define "must"}}
1. Format with the project formatter (gofumpt + goimports + golines) before finishing; never hand-format.
2. Use early returns over deep nesting; keep functions under ~80 lines and cyclomatic complexity under ~15.
   `else` is fine when both branches do genuinely different work; when one branch only holds a default
   that a condition overrides, assign the default value directly and drop the `else`.
3. A comment earns its place by saying WHY. Doc-comment an exported symbol only when its name and
   signature leave something out - a constraint, an invariant, a non-obvious bound or format, a
   reason the simpler-looking implementation is wrong. Delete the ones that restate the signature:
   `// ID returns the analyzer's unique identifier` above `func (a *A) ID() string` costs a reader
   a pause and repays nothing. Any comment kept on an exported symbol still starts with its name
   (staticcheck ST1020-ST1022 enforce that, and only that). No `// Package foo` docstrings, and no
   doc comments on unexported symbols beyond a `// why` when it is non-obvious. Cap any block at 5
   lines - `big-comment` enforces it.
4. Comments explain WHY, not WHAT; the code shows what. Do not narrate the obvious.
5. Reach for the newest fitting stdlib: `slices`/`maps`/`cmp`, `errors.Join`, `min`/`max`/`clear`,
   `for range int`, `iter.Seq` (range-over-func), typed atomics, generic type aliases. Prefer
   `slices.Equal` over hand-rolled comparison. Target the module's pinned Go version.
6. Pick value or pointer receivers and apply consistently per type. Design zero values to be safe.
7. Use `any`, never `interface{}`.
8. Add a compile-time `var _ Iface = (*impl)(nil)` assertion to any file whose type implements an interface.
9. No naked returns in long functions; no shadowing of `err`/`ctx`/predeclared identifiers in inner scopes.
10. Keep one concept per file (~180 lines, ~468 for a `_test.go`); split before a file grows past that.
11. Drop explicit generic type parameters wherever Go INFERS them from an argument: write
    `fuegoport.GET(reg, "/me", h.GetMe)`, not `fuegoport.GET[struct{}, OperatorResponse](reg, "/me", h.GetMe)`.
    Spell type params only where inference cannot (no value of the type appears in the call).
12. No AI text marker in `.go` or `.md`. Two classes, and they are handled differently. Typographic
    punctuation - em dash U+2014, en dash, ellipsis U+2026, curly quotes - is REPORTED and never
    rewritten, because inside a string literal or a fenced block it is data; replace it with its
    ASCII form by hand. Invisible carriers - zero-width characters, bidi controls, soft hyphens,
    exotic spaces - are removed by `pulsarules_cli clean --write`, and only the ones no neighbouring
    context can justify. A marker that a test needs as DATA is written as a Go escape (`\u2014`),
    never as the character, so the fixture never trips the check that validates it.
13. No AI provenance key in a markdown frontmatter block (`generator`, `ai_generated`, `claude`,
    `synthid`, `c2pa`, ...). Prose that merely names a vendor is not provenance and is not flagged.
14. No `regexp` for simple string work (prefix/suffix/contains/split/trim/case): use the `strings`
    stdlib or a small generic `[T ~string]` helper shared in a `pkg/strutils`-style package. Reserve
    `regexp` for genuinely pattern-based matching.
15. Model an optional value with its ZERO VALUE (`0`, `""`, `false`, a nil slice/map), never a
    pointer. Reach for `*T` only when a caller must distinguish "never set" from "set to the zero
    value" AND that distinction changes behaviour - a partial-update payload, a tri-state flag.
    Optionality alone does not earn a pointer: it costs every reader a nil check, every writer an
    allocation, and turns a missing value into a panic instead of a harmless default.
15. The nil-slice zero value is right in general and WRONG on a REST `Output` DTO: `encoding/json`
    serialises a nil slice as `null`, not `[]`, surprising a consumer expecting an array. A DTO
    field crossing a JSON boundary initialises its slice fields explicitly (`[]T{}` or a mapped
    result), even when the general zero-value principle above would otherwise leave it nil.
{{end}}

{{define "forbidden"}}
- Hand-formatting code or bypassing the formatter.
- Deep nesting where an early return flattens it; functions past ~80 lines without justification.
- `interface{}` instead of `any`; mixed value/pointer receivers on the same type.
- `fmt.Println`/`log.Println` in non-test code.
- A comment that restates the signature; a comment block over 5 lines; package docstrings; doc
  comments on unexported symbols beyond a non-obvious `// why`.
- Re-implementing what the stdlib already does (`slices.Equal`, `min`/`max`, `iter.Seq`).
- Shadowing an outer `err`/`ctx` with `:=` in an inner scope (govet `shadow`).
- Spelling out generic type parameters the compiler can infer from an argument.
- An AI text marker in `.go` or `.md`: typographic punctuation, an invisible carrier, an exotic
  space, or an AI provenance key in a markdown frontmatter block.
- `regexp` for simple string work the `strings` stdlib or a generic `[T ~string]` helper covers.
- `*T` for an optional field or parameter whose zero value already means "absent".
- A nil slice field left unset on a REST `Output` DTO, serialising as `null` instead of `[]`.
{{end}}

{{define "validation"}}
- [ ] Formatter run; no manual formatting remains.
- [ ] Functions under ~80 lines, complexity under ~15; early returns used.
- [ ] No comment restates its signature; every kept doc comment begins with its symbol's name and
  fits in 5 lines; no package docstrings; no doc comments on unexported symbols.
- [ ] Newest fitting stdlib used; no hand-rolled equivalents.
- [ ] Receivers consistent; `any` used; zero values safe.
- [ ] Interface-implementing types carry `var _ Iface = (*impl)(nil)`.
- [ ] No `err`/`ctx`/predeclared shadowing.
- [ ] Optional values carry the zero value, not `*T`, unless "unset" must differ from zero.
- [ ] Inferable generic type params omitted; no AI text marker in `.go` or `.md`; no `regexp` for simple
  string work (use `strings`/a generic `[T ~string]` helper).
- [ ] Output DTO slice fields crossing a JSON boundary are initialised explicitly, not left nil.
{{end}}
