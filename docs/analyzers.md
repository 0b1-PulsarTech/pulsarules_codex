# Comment and declaration analyzers

Each analyzer below enforces a rule from `knowledge/standards/rules/`. Params are defaults in code;
there is no project-config surface for them yet. `extensions` is a common param name, read under
each analyzer's own id. Findings in generated files are dropped unless `--include-generated` is
passed; the Go AST analyzers documented here (`test-comment`, `doc-comment`, `body-header-comment`,
`variadic-index`, `too-many-results`, `ungrouped-interfaces`, `package-level-var`,
`package-file-count` and `single-letter-name`) also skip generated files, even under
`--include-generated`. `--preset minimal` disables `test-comment`, `single-use-declaration`,
`doc-comment`, `body-header-comment`, `history-narration`, `variadic-index`, `too-many-results`,
`ungrouped-interfaces`, `package-level-var` and `package-file-count`; it keeps `single-letter-name`
with `exported_only` set, so only upper case single letters on a function, method, type, constant,
variable, field or interface method are reported.

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

- Rule: `comments`.
- Flags: comment text such as "was replaced by", "used to be", "the old code", "previously we",
  "no longer needed", "it/this/we/they/never used to ...".
- Params: `extensions` (default: the source extensions known to the language registry).
- Exemptions: text inside string literals is not scanned; the passive "is used to" ("the helper is
  used to build x") is not history and is not flagged.
- Known limits: multi-line strings, Lua `[[ ]]`, Postgres `E''`, Dart nested interpolation quotes,
  nested template-literal quotes, scss `//` inside `<style>`, Swift `#""`, and C++ `R"()"` can be
  misread. "the only tool ever used to call x" is flagged; it cannot be told apart from
  "nobody ever used to call this".

## single-letter-name

- Rule: `naming`.
- Flags: any declared identifier whose name is exactly one letter (`_` and `.` are not letters).
  Sites: the package clause, function and method names, receivers, receiver type parameters
  (`func (pair Pair[K, V])`), parameters and results of every func type (declaration, literal,
  interface method, func-typed field), type names and type parameters, `const`/`var` names
  (package level and local), struct fields, interface method names, `:=` left-hand sides
  (including `if`, `switch`, `for` and `select` forms), `range` key and value, type switch
  bindings, labels and import aliases. Only declarations are reported; each is reported once
  (a package clause once per file).
- Params: `exported_only` (bool, default false); when true a single letter is reported only if it
  is upper case and names a function, method, type, constant, variable, field or interface method;
  type parameters, the package clause, labels, receivers, loop variables, parameters, results, type
  switch bindings and import aliases are skipped. `--preset minimal` sets it.
- Exemptions: `_`; `import . "x"` and `import _ "x"` are not aliases; a parameter typed
  `*testing.T`, `*testing.B`, `*testing.F`, `*testing.M` or `testing.TB`, where `testing` is the
  name the file imports "testing" under (every import of it counts, aliases included; a file
  without the import gets no exemption). Under `import . "testing"` the bare `*T`, `*B`, `*F`,
  `*M` and `TB` are exempt.
- Syntactic: no type information is used, so a `:=` that reassigns an existing single-letter
  variable alongside a new one is reported again (a go/types count found 0 such sites in this
  repo and in terectek_comms), and a type named `testing.T` from another package is told apart
  only by the import name.

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

## test-comment

- Rule: `comments`.
- Flags: a doc comment on a test function or helper in a `_test.go` file, and a comment inside its
  body, except directive-only groups and an Example's `// Output:`.
- Params: none.
- Exemptions: directive-only groups (lint directives); an Example function's `// Output:` comment.

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

## top-of-file

- Rule: `comments`.
- Flags: a file whose first content is a comment. Severity is error for `.go`, warning elsewhere.
- Params: `extensions` (default `.go`, `.proto`, `.sql`, `.nix`, `.svelte`, `.ts`, `.dart`).
- Exemptions: required header lines (generated markers, tool directives) are skipped, and a file
  recognised as generated is exempt in full.
- Known limits: `/* a */ /* global b */ /* hand` is not counted; a directive line followed by code is
  skipped without a flush; nolint and global edge forms; a bare `/* eslint` prefix.

## file-size

- Rule: `effective-go`.
- Flags: a changed source file with more lines than the limit, 180 by default; a test file gets 2.6
  times that.
- Params: `max_lines` (default 180), plus `extensions`.
- Known limits: a `max_lines` at or below 0 (only a Go caller building its own `GovernanceConfig`
  can set it; no flag or config file does) falls back to the default 180 rather than disabling the
  check.

## big-comment

- Rule: `comments`.
- Flags: a hand-written comment block longer than `max_lines`.
- Params: `max_lines` (default 3), plus `extensions`.
- Exemptions: machine-read lines (directives, goverter, pulsarules markers) are not counted; a block
  led by a generated or simplification marker is exempt. Directive spans are gated on the language's
  own block syntax, and a second directive block is kept out of the hand count.
- Known limits: the directive edge forms listed under `top-of-file` apply here too. A `max_lines` at
  or below 0 (only a Go caller building its own `GovernanceConfig` can set it; no flag or config
  file does) falls back to the default 3 rather than disabling the check.

## commit-move-purity

- Rule: `commits`.
- Flags: a staged rename below `min_similarity`, a rename that carries an edit beyond its package
  clause and import fixups, renames mixed with unrelated edits in one commit, a byte-identical move
  that changes build inclusion (build-ignored path, root `vendor/`, `_test` or GOOS/GOARCH suffix,
  extension), a cross-directory move where a name resolves differently or load-time work changes
  package, an edit that drops the last import of a directory that keeps load-time work, and a move
  that strands or picks up a compiled non-Go file.
- Params: `min_similarity` (default 90).
- Exemptions: a pure package split (moved files with only package clause and import fixups) passes;
  any file may change a comment's `core.X` to `sources.X` for an exported name the move carried,
  when the comment follows the import declarations; excepted are directives, comments inside
  `Example` functions, and `// want` or `ERROR` assertion comments, as test tools check them; a file
  staying in the old directory may qualify `X` as `sources.X` in code; `errors.New` initialisers are
  inert; `_test.go` files may carry load-time work when their whole directory moves in one commit.
- Known limits: the analyzer reads syntax only, with no type checking.
  - Accepted as pure: renaming a `//go:embed`-matched asset; a same-directory rename that changes
    `init` order; renaming `go.mod`; moving a referenced exported var whose initialiser calls a
    function; a moved file's plain import used only in a function body whose init stops running;
    a dependant unlinking a directory whose staying files only import a package with init; an
    initialiser that panics only for some dynamic types; a nested module's `vendor/`.
  - Over-flagged: a `//go:build ignore` owner still counts for a name takeover; a moved `_test.go`
    with any load-time initialiser outside a whole-directory move; `var _ = g.F[int]`;
    `var _ = g.MyInt(3)`; a constant identifier as a divisor or shift count; any compiled non-Go
    file (`.s`, `.c`, `.h`, `.syso`) in either directory.
