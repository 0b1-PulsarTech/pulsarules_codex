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
  nested template-literal quotes, Swift `#""`, and C++ `R"()"` can be misread. "the only tool ever
  used to call x" is flagged; it cannot be told apart from "nobody ever used to call this".
