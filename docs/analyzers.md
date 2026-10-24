# Analyzer reference

Every analyzer `pulsarules_cli governance` runs, in run order. A finding names its analyzer id;
that id is the key under `analyzers:` and `overrides:` in `.pulsarules.yaml`.

- Scope is the narrowest `--scope` that runs the analyzer: `commit` analyzers run in every scope,
  `changed` ones in `changed` and `full`, `full` ones only in a full run (the default, and the
  `pre-push` hook).
- Rule is the knowledge rule the finding cites, from `knowledge/standards/rules/`.
- Severity is the default. `analyzers.<id>.severity` or an override changes it.
- Params are set under `analyzers.<id>.params`. A param of the wrong type, or a number of zero or
  less, falls back to its default. A list param takes a YAML list or a comma separated string; a
  YAML list item that is not a string, such as an unquoted number, is a config error.
- Params are project-wide. Only `file-size` accepts per-path params, inside `overrides`.
- A sub-id such as `import-groups/path` is a finding id, not an analyzer. It works only under
  `overrides`, and an override naming a sub-id falls back to the entry for its base id.
- `extensions` limits an analyzer to some file extensions; its default is given per analyzer.

Both tables are checked against the analyzer registry by `TestAnalyzerDocs_MatchRegistry` in
`internal/governance/registry`: a new analyzer, param, rule or scope fails the test until the
tables list it.

| ID | Scope | Rule | Severity |
| --- | --- | --- | --- |
| `file-size` | commit | `source-files` | warning, error past the hard cap |
| `text-markers` | commit | `source-files` | error, warning when context matters |
| `typographic-markers` | commit | `source-files` | error |
| `import-groups` | commit | `imports` | error, warning for `/path` |
| `naming` | commit | `naming` | warning |
| `top-of-file` | commit | `comments` | error for Go, warning elsewhere |
| `big-comment` | commit | `comments` | warning |
| `simplification-path` | commit | `minimalism` | warning |
| `commit-lint` | commit | `commits` | error, info for soft repeats |
| `commit-move-purity` | commit | `commits` | warning |
| `history-narration` | commit | `comments` | warning |
| `branch-name` | full | `commits` | error |
| `golangci-lint` | full | `code-smells` | from the linter |
| `go-syntax` | commit | - | warning, info |
| `dart-syntax` | commit | - | warning, info |
| `typescript-syntax` | commit | - | warning, info |
| `javascript-syntax` | commit | - | warning, info |
| `svelte-syntax` | commit | - | warning, info |
| `control-flow` | commit | `code-smells` | warning |
| `shadowing` | commit | `effective-go` | warning |
| `complexity` | commit | `code-smells` | warning |
| `magic-number` | commit | `code-smells` | info |
| `named-returns` | commit | `named-returns` | warning |
| `test-comment` | commit | `comments` | warning |
| `single-use-declaration` | commit | `minimalism` | warning |
| `doc-comment` | commit | `effective-go` | warning |
| `body-header-comment` | commit | `effective-go` | warning |
| `single-letter-name` | commit | `naming` | warning |
| `forbidden-token` | commit | `naming` | warning |
| `variadic-index` | commit | `flag-arguments` | warning |
| `too-many-results` | commit | `effective-go` | warning |
| `ungrouped-interfaces` | commit | `effective-go` | warning |
| `package-level-var` | commit | `code-smells` | warning |
| `package-file-count` | commit | `code-placement-go` | warning |
| `time-discipline` | commit | `concurrency` | warning |
| `context-discipline` | commit | `concurrency` | warning |
| `pprof-labels` | commit | `security` | error |
| `arch-boundary` | changed | `dependency-rule` | error |
| `import-cycle` | changed | `module-boundaries` | error |

Params, in the order each analyzer declares them:

| ID | Param | Default |
| --- | --- | --- |
| `file-size` | `max_lines` | 180 |
| `file-size` | `error_lines` | 0 (off); 290 for `.svelte` |
| `file-size` | `extensions` | every source extension |
| `text-markers` | `extensions` | every source extension and `.md` |
| `typographic-markers` | `extensions` | every source extension and `.md` |
| `top-of-file` | `extensions` | `.go .proto .sql .nix .svelte .ts .tsx .dart .js .mjs .cjs .jsx` |
| `big-comment` | `max_lines` | 3 |
| `big-comment` | `extensions` | every source extension |
| `simplification-path` | `extensions` | every source extension |
| `commit-lint` | `emoji_hard_window` | 5 |
| `commit-lint` | `emoji_soft_window` | 20 |
| `commit-lint` | `emoji_suggestions` | 7 |
| `commit-move-purity` | `min_similarity` | 90 |
| `history-narration` | `extensions` | every source extension |
| `branch-name` | `extra_types` | none |
| `golangci-lint` | `config_path` | golangci-lint's own lookup |
| `complexity` | `max_complexity` | 15 |
| `complexity` | `max_func_lines` | 80 |
| `complexity` | `max_params` | 5 |
| `control-flow` | `forbid_else` | false |
| `magic-number` | `allowed_numbers` | the built-in list, below |
| `body-header-comment` | `min_lines` | 2 |
| `single-letter-name` | `exported_only` | false |
| `single-letter-name` | `allowed_names` | none |
| `forbidden-token` | `tokens` | none (the analyzer is idle) |
| `forbidden-token` | `allowed_names` | none |
| `too-many-results` | `max_results` | 3 |
| `package-file-count` | `max_files` | 10 |
| `package-file-count` | `exempt` | none |
| `package-file-count` | `extensions` | `.go` |

Delegated commands add their own ids (see the README). A `config` finding (error) reports an
invalid `.pulsarules.yaml`; it is not an analyzer and cannot be configured.

## Text and layout

- `file-size`: a source file longer than `max_lines` (default 180). A test file gets 2.6 times
  the limit (468). `error_lines` (default 0, off; 290 for `.svelte`) is a hard cap reported as an
  error, also times 2.6 for tests; when both are set it must exceed `max_lines`. Only these two
  params can vary per path. `extensions`: every source extension.
- `text-markers`: invisible and no-break characters (zero-width, bidi controls, soft hyphens,
  exotic spaces). One that may be load-bearing, such as an emoji joiner, is a warning. It also
  reports AI provenance keys in Markdown frontmatter.
  `extensions`: every source extension and `.md`.
- `typographic-markers`: AI typographic punctuation such as an em dash or curly quotes. The
  installed hooks can bake another severity with `--typographic-severity`. `extensions`: as above.
- `import-groups`: import blocks out of order. Go: standard library, third party, project. Dart:
  `dart:`, `package:`, relative. TypeScript, JavaScript and Svelte: builtins, external packages,
  local paths and aliases. The `import-groups/path` warning flags a `../` import that an alias
  covers, an own-directory alias, and an import that climbs out of its directory or package.
- `package-file-count`: a Go package directory with more than `max_files` (default 10) non-test
  files. `exempt` takes path patterns (`dir/**` exempts a tree). `extensions`: `.go`.

## Comments

- `top-of-file`: a file whose first content is a comment. Generated markers and tool directives
  are skipped. `extensions`: `.go .proto .sql .nix .svelte .ts .tsx .dart .js .mjs .cjs .jsx`.
- `big-comment`: a hand-written comment block longer than `max_lines` (default 3); directive
  lines do not count. `extensions`: every source extension.
- `simplification-path`: a `simplification:` comment that names no upgrade path (`upgrade path:`
  or `revisit when`). `extensions`: every source extension.
- `history-narration`: a comment that narrates history ("used to", "was replaced by", "no longer
  needed") instead of stating what holds. `extensions`: every source extension.
- `doc-comment`: a doc comment on a Go func, method, type or interface method. Directive-only
  groups and a short `Deprecated:` note are exempt; tests belong to `test-comment`.
- `body-header-comment`: a comment block of `min_lines` (default 2) or more prose lines opening a
  Go function body.
- `test-comment`: a doc comment on a Go test function or helper, or a comment inside one, except
  directives and an Example's `// Output:`.

## Names and declarations

- `naming`: numbered names, Hungarian notation and noise words.
- `single-letter-name`: any declared identifier that is one letter, in Go, Dart, TypeScript,
  JavaScript and Svelte. `allowed_names` (default empty) exempts exact names. With
  `exported_only` (default false) only an upper-case letter naming a function, method, type,
  constant, variable, field or interface method is reported; `--preset minimal` sets it. A
  `*testing.T`-style parameter is exempt.
- `forbidden-token`: a name that holds a word from `tokens` (default none, which turns the
  analyzer off; set it in the project file, with generic words such as `draft`, `legacy` or
  `expired`). A name is split into lower-cased words at case changes and at any character that is
  not a letter or digit, so `legacyKept` and `legacy_kept` both hold `legacy` but `legacies` and
  `legacy2` do not: matching is by whole word, not by stem. A multi-word token such as
  `without reload` matches only those words next to each other, in order. Declared names of every
  kind are checked, test files included, but not generated files or names the parser marks
  exempt. A file name is checked without its extension, and each directory on a path is reported
  once, as `forbidden-token/dir`. `allowed_names` (default none) exempts exact names, file base
  names and directory names.
- `shadowing`: a Go identifier that shadows an outer one, or a `:=` that redeclares a name in the
  block that declared it.
- `single-use-declaration`: a Go declaration referenced only inside one function; move it there.
- `package-level-var`: a package-level Go `var`. Sentinel errors, `//go:embed` variables,
  interface assertions and constant enum groups are exempt.
- `ungrouped-interfaces`: a Go file with two or more interfaces outside one `type ( ... )` block.

## Signatures and control flow

- `complexity`: a Go function over `max_complexity` (default 15) cyclomatic complexity,
  `max_func_lines` (default 80) lines or `max_params` (default 5) parameters, and flag arguments.
  Test files get 1.5 times the line limit.
- `magic-number`: a numeric literal that is not in `allowed_numbers`, in Go and Dart. The list
  replaces the built-in one (`0 1 -1 0.0 1.0 2 10 60 100 1000 3600 86400 255 256 1024 1025`)
  rather than extending it, and its items are strings, so quote them
  (`["0", "1", "24"]`). Test files are exempt. Go checks function bodies and skips `const`
  blocks, octal file modes and bit-size arguments of the standard library. Dart skips only the
  literals of a `const` declaration (a `const` expression elsewhere still counts), and a literal
  inside a string interpolation counts. `--preset minimal` turns it off.
- `control-flow`: nesting deeper than 4, and an `else` after a branch that already returns,
  throws, breaks or continues (Go: an `else` block holding a return, a panic, a loop or a
  branch statement). The `else` findings carry the sub-id `control-flow/else`. With
  `forbid_else` (default false) every `else` is reported instead; the nesting check stays.
- `named-returns`: a Go function returning more than one unnamed result of the same type.
- `too-many-results`: a Go function returning more than `max_results` (default 3) values.
- `variadic-index`: positional indexing of a variadic parameter, such as `opts[0]`.

## Concurrency and security

- `time-discipline`: a `time.Sleep` outside a `testing/synctest` bubble, and a `func() time.Time`
  struct field used as a clock seam.
- `context-discipline`: a ticker-paced loop with no `<-ctx.Done()` arm and no
  `//pulsarules:process-lifetime <reason>` marker, and a goroutine that only waits on
  `<-ctx.Done()` (use `context.AfterFunc`).
- `pprof-labels`: a `pprof.Labels` key that names personal data or a secret (`user`, `email`,
  `token`, `password`, ...), since label values reach panic tracebacks.

## Commits and branches

- `commit-lint`: the commit message format, type, scope, emoji (from the catalog), length limits
  (subject 72, body line 100, body 150) and tool attribution trailers. The same emoji within the
  last `emoji_hard_window` (default 5) commits is an error; within `emoji_soft_window` (default 20)
  it is info. `emoji_suggestions` (default 7) caps the suggested replacements.
- `commit-move-purity`: a staged rename below `min_similarity` (default 90) percent, a rename that
  carries an edit beyond its package clause and imports, renames mixed with unrelated edits, and
  moves that change build inclusion or what a `//go:embed` pattern includes. A Dart move is pure
  when its directives (`import`, `export`, `part`, `library`) are the old ones, with the same
  shape apart from the URIs, and the code outside them is byte-identical. Each URI must keep its
  text and still resolve to a target that moved with it, or change text so that it reaches the
  moved target: a relative import rewritten for the new location, or `package:` and relative
  swapped inside one package's `lib/`. A `dart:` or `file:` URI, or a package neither side
  knows, must be unchanged. Re-sorting the directives is an edit. A move of identical bytes whose
  relative imports no longer resolve gets the relocation finding, as in Go. It is always an edit
  when two `pubspec.yaml` files claim one package name, a URI holds `%`, `?`, `#`, `\` or `$`, a
  `package:` path has an empty or dot segment, or the source does not parse.
- `branch-name`: a checked-out branch without a type prefix (`<type>/<description>`, a
  Conventional Commit type or a gitflow line). `main`, `master` and `develop` are exempt.
  `extra_types` adds prefixes; the `pre-push` hook can bake them with `--branch-extra-types`.

## Architecture and delegation

- `arch-boundary`: an inner-layer package importing an outer-layer one.
- `import-cycle`: a cycle among the project's packages.
- `golangci-lint`: runs `golangci-lint` and reports each issue as `golangci-lint/<linter>`, an
  error when the linter says so and a warning otherwise. `config_path` (default: golangci-lint's
  own lookup) names the config file; `governance --golangci-config` sets it for one run.

## Syntax

`go-syntax`, `dart-syntax`, `typescript-syntax`, `javascript-syntax` and `svelte-syntax` report
`<lang>-syntax/unparsed` (warning) for a file that does not parse or exceeds a parser limit, and
`<lang>-syntax/unavailable` (info) once when the language's parser cannot run.
