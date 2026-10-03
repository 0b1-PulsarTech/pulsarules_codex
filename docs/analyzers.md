# Comment and declaration analyzers

Each analyzer below enforces a rule from `knowledge/standards/rules/`. Params are defaults in code;
there is no project-config surface for them yet. `extensions` is a common param name, read under
each analyzer's own id.

## doc-comment

- Rule: `effective-go` (no doc comments on funcs, methods, types).
- Flags: a doc comment on a func, method, type declaration, type group, or interface method.
- Params: none.
- Exemptions: directive-only groups (`//nolint`, `//go:generate`), simplification-path marker groups, a
  `Deprecated:` note of at most 3 lines with no blank comment line, and test functions or helpers in
  `_test.go` files (owned by `test-comment`).
- Known limits: only func, method, type and interface-method declarations are checked; a doc
  comment on a `const` or `var` declaration and a comment on a struct field are not flagged.
- Behaviour: when a directive (for example `// goverter:`) and prose share a group the hint says to
  keep the directive lines.

## body-header-comment

- Rule: `effective-go`.
- Flags: a comment block of `min_lines` or more prose lines directly opening a function or function
  literal body; the comment lines before the first statement, summed.
- Params: `min_lines` (default 2).
- Exemptions: directive lines are not counted; a group led by a simplification-path marker is exempt;
  test functions and helpers.

## history-narration

- Rule: `effective-go`.
- Flags: comment text such as "was replaced by", "used to be", "the old code", "previously we",
  "no longer needed", "it/this/we/they/never used to ...".
- Params: `extensions` (default: the source extensions known to the language registry).
- Exemptions: text inside string literals is not scanned; the passive "is used to" ("the helper is
  used to build x") is not history and is not flagged.
- Known limits: multi-line strings, Lua `[[ ]]`, Postgres `E''`, Dart nested interpolation quotes,
  nested template-literal quotes, scss `//` inside `<style>`, Swift `#""`, and C++ `R"()"` can be
  misread. "the only tool ever used to call x" is flagged; it cannot be told apart from
  "nobody ever used to call this".

## loop-var-i

- Rule: `naming`.
- Flags: a `for` or `range` loop variable named `i` used for more than indexing or bounding.
- Params: none.
- Exemptions: an inner redeclaration of `i` (`:=`, `var`, `if`/`for`/`switch` init, closure param)
  ends tracking; a user-defined `len`/`cap` is not a bound.
- Known limits: only `len` and `cap` are bounds; a range value named `i` is not checked; pointer and
  closure aliasing is not followed; a define in an `if`/`for`/`switch`/`range` init skips its
  right-hand side; a root loop-init define right-hand side counts as a move (under-report,
  contrived); labelled defines; named and generic composite types.

## variadic-index

- Rule: `flag-arguments`.
- Flags: positional indexing of a variadic parameter (`opts[0]`).
- Params: none.
- Remedies, both valid: take a named options struct, or range over the values
  (`for _, opt := range opts`).
- Known limits: pairwise comparison with two range indexes over the same variadic (for example
  `a[i] vs a[j]`) is flagged, and the hint does not fit it.

## too-many-results

- Rule: `effective-go`.
- Flags: a function, method, func type, interface method or func literal returning more than
  `max_results` values.
- Params: `max_results` (default 3).
- Remedy: return a named result of an anonymous struct, or a named struct type when reused.

## ungrouped-interfaces

- Rule: `effective-go`.
- Flags: a file declaring 2 or more interfaces where some are outside the main `type ( ... )` block.
- Params: none.
- Remedy: move all interfaces in the file into one `type ( ... )` block.

## package-level-var

- Rule: `code-smells` (global state).
- Flags: every package-level `var` that is not exempt, mutable or not.
- Params: none.
- Exemptions: sentinel errors (`errors.New` and `fmt.Errorf` values, typed `Err*` sentinels (a
  composite literal of an `*Err`/`*Error` type, or an `Err`-named constructor call), re-exported
  `pkg.Err*` sentinels `var ErrX = pkg.ErrY`), `//go:embed` variables, `var _ I = T{}` assertions,
  and test variables assigned in `TestMain`. A name already reported by `single-use-declaration`
  is not reported twice.
- Skipped: directories Go ignores (leading `.` or `_`, `testdata`).

## package-file-count

- Rule: `code-placement-go`.
- Flags: a package directory with more than `max_files` non-test `.go` files, anchored on a changed
  file of that package.
- Params: `max_files` (default 10), `exempt` (path patterns; `dir/**` exempts a whole tree), plus
  `extensions`.
- Exemptions: generated files, ent schema directories (a strict majority of files embedding
  `ent.Schema`), files excluded by a `//go:build ignore` or `// +build ignore` constraint, files whose
  name starts with `_` or `.`, files of the minority package in a directory, and directories Go
  ignores.
- Known limits: `/* */` and `//go:build` on one line; `+build` followed by `/* */`; the OR
  form (space-separated terms) of ignore expressions; two `//go:build` lines.
