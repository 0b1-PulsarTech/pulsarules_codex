# Comment and declaration analyzers

Each analyzer below enforces a rule from `knowledge/standards/rules/`. Params are defaults in code;
there is no project-config surface for them yet.

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
