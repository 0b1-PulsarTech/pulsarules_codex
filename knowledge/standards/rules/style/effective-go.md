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
    - vacuous-doc
    - shadowing
---

# Effective Go subset

> The enforceable Effective Go / Code Review Comments baseline every Go file must meet.

Applies to: all Go code. Canonical references: [Effective Go](https://go.dev/doc/effective_go),
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Target Go version: track the
newest stable the module pins (source repos use `go 1.27`).

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
3. Default to no comment. Write one only when it clears a bar: it says something the code cannot
   say, and a reader would otherwise get wrong - a constraint, an invariant, a non-obvious bound or
   format, a reason the simpler-looking implementation is wrong. Restating a signature, narrating
   control flow, or naming what a well-named function already names does not clear it: delete
   `// ID returns the analyzer's unique identifier` above `func (a *A) ID() string`, it costs a
   reader a pause and repays nothing. A comment kept on an exported symbol still starts with its
   name (staticcheck ST1020-ST1022 enforce that, and only that). No `// Package foo` docstrings, and
   no doc comments on unexported symbols beyond a `// why` that clears the bar. Cap any block at 5
   lines (`big-comment`) - one sentence of why, not a paragraph; `vacuous-doc` flags a doc comment
   on a function whose whole body is one return of a literal or field, unless it carries a `why:`
   marker or a second sentence.
4. A comment that survives explains why, never what; the code already shows what.
5. Place the surviving why at the line it explains, not in a header above the function. A header
   comment is the last resort, earned only when the reason spans the whole function and no single
   line owns it. Reasoning parked in a header drifts from the code it describes and is read once, at
   most; the same sentence beside the statement it justifies is read by whoever is about to change
   that statement.
6. Reach for the newest fitting stdlib: `slices`/`maps`/`cmp`, `errors.Join`, `min`/`max`/`clear`,
   `for range int`, `iter.Seq` (range-over-func), typed atomics, generic type aliases,
   `cmp.Or(flagValue, envValue, defaultValue)` for a first-non-zero fallback chain,
   `strings.CutLast`/`bytes.CutLast` for a last-separator split, and the standard library's `uuid`
   package (`uuid.NewV7` for a time-ordered id). Prefer `slices.Equal` over hand-rolled comparison.
   `cmp.Or` evaluates every argument before the call, so keep an `if` chain where a fallback is
   expensive or has a side effect. A `uuid` value is a raw scalar and crosses a boundary wrapped in
   a named type (see [[types]]). Target the module's pinned Go version.
7. Pick value or pointer receivers and apply consistently per type. Design zero values to be safe.
8. Use `any`, never `interface{}`.
9. Add a compile-time `var _ Iface = (*impl)(nil)` assertion to any file whose type implements an interface.
10. No naked returns in long functions; no shadowing of `err`/`ctx`/predeclared identifiers in inner scopes.
11. Keep one concept per file (~180 lines, ~468 for a `_test.go`); split before a file grows past that.
12. Drop explicit generic type parameters wherever Go infers them from an argument: write
    `route.GET(reg, "/me", h.GetMe)`, not
    `route.GET[struct{}, OperatorResponse](reg, "/me", h.GetMe)`. Spell type params only where
    inference cannot (no value of the type appears in the call).
13. No AI text marker in `.go` or `.md`. Two classes, and they are handled differently. Typographic
    punctuation - em dash U+2014, en dash, ellipsis U+2026, curly quotes - is reported and never
    rewritten, because inside a Go string literal it is data; replace it with its ASCII form by
    hand. In markdown the fence is machine-readable, so a fenced block or an inline code span is
    skipped outright - the character there is content being shown, not prose style. Invisible
    carriers - zero-width characters, bidi controls, soft hyphens, exotic spaces - are removed by
    `pulsarules_cli clean --write`, and only the ones no neighbouring context can justify. A marker
    that a test needs as data is written as a Go escape (`\u2014`), never as the character, so the
    fixture never trips the check that validates it.
14. How hard a marker lands depends on its class. An invisible carrier no neighbouring context can
    justify blocks: the bidi controls among them are the Trojan Source vector (CVE-2021-42574),
    where the diff a reviewer reads is not the code the compiler sees. One that may be
    load-bearing - a joiner between emoji, a byte order mark - only advises, because this check
    cannot judge it. Typographic punctuation blocks by default; a project that treats ASCII
    punctuation as house style rather than a defect lowers it through the analyzer's `severity`
    param (`error`, `warning`, `info`), which `--typographic-severity` bakes into the generated
    hooks at install time.
15. No AI provenance key in a markdown frontmatter block (`generator`, `ai_generated`, `claude`,
    `synthid`, `c2pa`, ...). Prose that merely names a vendor is not provenance and is not flagged.
16. No `regexp` for simple string work (prefix/suffix/contains/split/trim/case): use the `strings`
    stdlib or a small generic `[T ~string]` helper shared in a `pkg/strutils`-style package. Reserve
    `regexp` for genuinely pattern-based matching.
17. Model an optional value with its zero value (`0`, `""`, `false`, a nil slice/map), never a
    pointer. Reach for `*T` only when a caller must distinguish "never set" from "set to the zero
    value" and that distinction changes behaviour - a partial-update payload, a tri-state flag.
    Optionality alone does not earn a pointer: it costs every reader a nil check, every writer an
    allocation, and turns a missing value into a panic instead of a harmless default.
18. The nil-slice zero value is right in general and wrong on a REST `Output` DTO: `encoding/json`
    serialises a nil slice as `null`, not `[]`, surprising a consumer expecting an array. A DTO
    field crossing a JSON boundary initialises its slice fields explicitly (`[]T{}` or a mapped
    result), even when the general zero-value principle above would otherwise leave it nil.
19. Set a promoted field directly in a keyed struct literal - `Outer{ID: 7}`, not
    `Outer{Base: Base{ID: 7}}` (Go 1.27). A literal may not name both a promoted field and the
    embedded field that promotes it, and the form does not reach through a pointer-embedded type,
    which still needs the embedded value spelled out.
{{end}}

{{define "forbidden"}}
- Hand-formatting code or bypassing the formatter.
- Deep nesting where an early return flattens it; functions past ~80 lines without justification.
- `interface{}` instead of `any`; mixed value/pointer receivers on the same type.
- `fmt.Println`/`log.Println` in non-test code.
- A comment written by default. One that restates a signature, narrates control flow, or names what
  a well-named function already names; a comment block over 5 lines; package docstrings; doc
  comments on unexported symbols beyond a `// why` that clears the bar.
- A why parked in a function header when one line inside owns the reason - put it on that line.
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
- [ ] No comment by default; every kept one clears the bar (says what the code cannot). None
  restates a signature, narrates control flow, or names what the code already names; every kept doc
  comment begins with its symbol's name and fits in 5 lines; no package docstrings; no doc comments
  on unexported symbols beyond a `// why` that clears the bar.
- [ ] Every surviving why sits at the line it explains; a header comment carries only a reason that
  spans the whole function.
- [ ] Newest fitting stdlib used; no hand-rolled equivalents.
- [ ] Receivers consistent; `any` used; zero values safe.
- [ ] Interface-implementing types carry `var _ Iface = (*impl)(nil)`.
- [ ] No `err`/`ctx`/predeclared shadowing.
- [ ] Optional values carry the zero value, not `*T`, unless "unset" must differ from zero.
- [ ] Inferable generic type params omitted; no AI text marker in `.go` or `.md`; no `regexp` for
  simple string work (use `strings`/a generic `[T ~string]` helper).
- [ ] A blocking marker finding was fixed, not silenced; any `severity` the project lowered is a
  deliberate house-style choice, never a way past a bidi control.
- [ ] Output DTO slice fields crossing a JSON boundary are initialised explicitly, not left nil.
{{end}}
