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
    - vacuous-doc
    - doc-comment
    - body-header-comment
    - too-many-results
    - ungrouped-interfaces
    - package-level-var
    - shadowing
---

# Effective Go subset

> The enforceable Effective Go / Code Review Comments baseline every Go file must meet.

Applies to: all Go code. Canonical references: [Effective Go](https://go.dev/doc/effective_go),
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). Target Go version: track the
newest stable the module pins (source repos use `go 1.27`).

{{define "when"}}
- Writing or editing any `.go` file.
- Choosing control-flow shape, result and interface shape, or stdlib idioms.
- Deciding whether to hand-roll something the stdlib already provides.
{{end}}

{{define "must"}}
1. Format with the project formatter (gofumpt + goimports + golines) before finishing; never hand-format.
2. Use early returns over deep nesting; keep functions under ~80 lines and cyclomatic complexity under ~15.
   `else` is fine when both branches do genuinely different work; when one branch only holds a default
   that a condition overrides, assign the default value directly and drop the `else`.
3. Go-only comment checks: no doc comment on a func, method, or type, exported or not (`doc-comment`);
   the exceptions are a `Deprecated:` note (3 lines at most), a directive-only group such as
   `//nolint:ireturn // reason`, and a simplification-path marker group; tests are owned by
   `test-comment`. A comment block of 2+ lines opening a function body is flagged
   (`body-header-comment`, `min_lines` 2). Policy: [[comments]].
4. Reach for the newest fitting stdlib: `slices`/`maps`/`cmp`, `errors.Join`, `min`/`max`/`clear`,
   `for range int`, `iter.Seq` (range-over-func), typed atomics, generic type aliases,
   `cmp.Or(flagValue, envValue, defaultValue)` for a first-non-zero fallback chain,
   `strings.CutLast`/`bytes.CutLast` for a last-separator split, and the standard library's `uuid`
   package (`uuid.NewV7` for a time-ordered id). Prefer `slices.Equal` over hand-rolled comparison.
   `cmp.Or` evaluates every argument before the call, so keep an `if` chain where a fallback is
   expensive or has a side effect. A `uuid` value is a raw scalar and crosses a boundary wrapped in
   a named type (see [[types]]). Target the module's pinned Go version.
5. Pick value or pointer receivers and apply consistently per type. Design zero values to be safe.
6. Use `any`, never `interface{}`.
7. Add a compile-time `var _ Iface = (*impl)(nil)` assertion to any file whose type implements an interface.
8. No naked returns in long functions; no shadowing of `err`/`ctx`/predeclared identifiers in inner scopes.
9. Keep one concept per file (~180 lines, ~468 for a `_test.go`); split before a file grows past that.
10. Drop explicit generic type parameters wherever Go infers them from an argument: write
    `route.GET(reg, "/me", h.GetMe)`, not
    `route.GET[struct{}, OperatorResponse](reg, "/me", h.GetMe)`. Spell type params only where
    inference cannot (no value of the type appears in the call).
11. No AI text marker in `.go` or `.md`. Two classes, and they are handled differently. Typographic
    punctuation - em dash U+2014, en dash, ellipsis U+2026, curly quotes - is reported and never
    rewritten, because inside a Go string literal it is data; replace it with its ASCII form by
    hand. In markdown the fence is machine-readable, so a fenced block or an inline code span is
    skipped outright - the character there is content being shown, not prose style. Invisible
    carriers - zero-width characters, bidi controls, soft hyphens, exotic spaces - are removed by
    `pulsarules_cli clean --write`, and only the ones no neighbouring context can justify. A marker
    that a test needs as data is written as a Go escape (`\u2014`), never as the character, so the
    fixture never trips the check that validates it.
12. How hard a marker lands depends on its class. An invisible carrier no neighbouring context can
    justify blocks: the bidi controls among them are the Trojan Source vector (CVE-2021-42574),
    where the diff a reviewer reads is not the code the compiler sees. One that may be
    load-bearing - a joiner between emoji, a byte order mark - only advises, because this check
    cannot judge it. Typographic punctuation blocks by default; a project that treats ASCII
    punctuation as house style rather than a defect lowers it through the analyzer's `severity`
    param (`error`, `warning`, `info`), which `--typographic-severity` bakes into the generated
    hooks at install time.
13. No AI provenance key in a markdown frontmatter block (`generator`, `ai_generated`, `claude`,
    `synthid`, `c2pa`, ...). Prose that merely names a vendor is not provenance and is not flagged.
14. No `regexp` for simple string work (prefix/suffix/contains/split/trim/case): use the `strings`
    stdlib or a small generic `[T ~string]` helper shared in a `pkg/strutils`-style package. Reserve
    `regexp` for genuinely pattern-based matching.
15. Model an optional value with its zero value (`0`, `""`, `false`, a nil slice/map), never a
    pointer. Reach for `*T` only when a caller must distinguish "never set" from "set to the zero
    value" and that distinction changes behaviour - a partial-update payload, a tri-state flag.
    Optionality alone does not earn a pointer: it costs every reader a nil check, every writer an
    allocation, and turns a missing value into a panic instead of a harmless default.
16. The nil-slice zero value is right in general and wrong on a REST `Output` DTO: `encoding/json`
    serialises a nil slice as `null`, not `[]`, surprising a consumer expecting an array. A DTO
    field crossing a JSON boundary initialises its slice fields explicitly (`[]T{}` or a mapped
    result), even when the general zero-value principle above would otherwise leave it nil.
17. Set a promoted field directly in a keyed struct literal - `Outer{ID: 7}`, not
    `Outer{Base: Base{ID: 7}}` (Go 1.27). A literal may not name both a promoted field and the
    embedded field that promotes it, and the form does not reach through a pointer-embedded type,
    which still needs the embedded value spelled out.
18. A function returns at most 3 results (`too-many-results`; it also flags func types, interface
    methods and func literals). Past that, return a named result of an anonymous struct, or a named
    struct type when reused, so call sites read by field name and a new value is not a signature
    break.
19. Two or more interfaces declared in one file go in a single `type ( ... )` block
    (`ungrouped-interfaces`), so the file's contracts read as one list.
20. No package-level var outside the exempt set: sentinel errors and re-exported `pkg.Err*`
    sentinels (`var ErrX = pkg.ErrY`), `//go:embed`, `var _ I = (*T)(nil)` / `T{}` assertions,
    TestMain-assigned test vars (`package-level-var`; policy: [[code-smells]]).
{{end}}

{{define "forbidden"}}
- Hand-formatting code or bypassing the formatter.
- Deep nesting where an early return flattens it; functions past ~80 lines without justification.
- `interface{}` instead of `any`; mixed value/pointer receivers on the same type.
- `fmt.Println`/`log.Println` in non-test code.
- A function with 4 or more results; return a named result of an anonymous struct, or a named struct
  type when reused.
- Two or more interface declarations in one file outside a single `type ( ... )` block.
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
- [ ] No function returns more than 3 results; no file declares 2+ interfaces outside one `type ( )` block.
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
