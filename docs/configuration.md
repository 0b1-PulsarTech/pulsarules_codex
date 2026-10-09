# Project configuration

A project tunes the governance analyzers with a `.pulsarules.yaml` at its root. The file is
optional: without it the built-in defaults and the `recommended` preset apply.

## Where the configuration comes from

- `--config PATH` (on `governance`) loads only that file. Nothing else is read.
- Otherwise two files are layered, key by key:
  1. `<project>/.pulsarules.yaml`, committed with the project (the team layer).
  2. `<git-common-dir>/pulsarules.yaml`, a private layer that is never committed or published. In a
     linked worktree the git common dir is the main repository's `.git`, so every worktree of one
     repository shares one private file.
- Precedence, lowest to highest: built-in defaults, the preset (`--preset`, else the file's
  `preset`, else `recommended`), the config file, then command line flags. Per-path overrides apply
  after the global severity, so they also beat a hook-baked `--typographic-severity`.

### How the private layer merges

| Key | Rule |
| --- | --- |
| `preset`, `include_generated` | the private value wins |
| `skip_dirs`, `tests` | union of both |
| `analyzers` | per field (`enabled`, `severity`, `params`); the private value wins |
| `overrides` | the team's entries, then the private ones; later matches win key by key |
| `delegated` | by `id`; a private entry replaces the team's |

If the private layer disables an analyzer (directly or through its preset), the team's `overrides`
for that analyzer are dropped: private wins over everything for that analyzer. `--preset` on the
command line alone does not trigger this.

## A broken configuration is loud

An unknown key, unknown analyzer, unknown param, duplicate key, unsupported `version` or unknown
`preset` is an error, never a silent default. Effects:

- `pulsarules_cli governance` prints the error and exits 1.
- The git `pre-commit` hook (`governance --scope commit`) prints
  `pulsarules_cli: .pulsarules.yaml: line N: ...` and the commit is refused. The `commit-msg` hook
  (`commitlint`) fails with `[ERROR] config: ...`, so the commit is refused too.
- The Stop hook reports an `[error] config:` finding as a notice (it never blocks).
- The error lists at most 20 problems and about 3 KB of text, then ends with `... and N more
  problems`; fix those and run again to see the rest.

Schema errors are reported first. Cross-checks, such as an override that enables an analyzer the
project disabled ("an override can only narrow it"), appear only after those are fixed, so expect
two rounds.

## File reference

```yaml
version: 1                  # required; the only supported value
preset: recommended         # recommended | strict | minimal
include_generated: false    # also check generated files
skip_dirs: [vendor]         # directory basenames, not paths
tests:                      # extra globs that count as test files
  - "**/fixtures/**"
analyzers:                  # keyed by analyzer id (see docs/analyzers.md)
  file-size:
    enabled: true
    severity: warning       # error | warning | info
    params: {max_lines: 220}
  single-letter-name:
    params: {allowed_names: [i, j]}
overrides:                  # per-path narrowing
  - files: ["internal/gen/**"]
    analyzers:
      file-size: {params: {max_lines: 600, error_lines: 800}}
      import-groups/path: {enabled: false}
delegated:                  # see docs/delegated-commands.md
  - id: dart-analyze
    format: dart-analyze-json
    run: [dart, analyze, --format=json]
```

- `analyzers` keys are registry ids or delegated ids. Internal ids (`output`, `rule-injection`)
  are rejected.
- Sub-finding ids such as `ai-frontmatter`, `short-decl-reuse` and `commit-*` are governed only
  through their owner analyzer's key.
- Sub-ids that name one finding of an analyzer (`import-groups/path`, `golangci-lint/errcheck`,
  `dart-analyze/unused_import`) are accepted only inside `overrides`.
- An unknown param is an error. A param of the wrong type or a value of zero or less falls back
  to the analyzer's default silently, except for the `file-size` limits below.
- `params` of list type accept a YAML list or a comma separated string. An empty list means the
  default.

## Overrides and path globs

An override applies its analyzer settings to files matching any of its `files` globs. Several
matching overrides apply in order; later ones win key by key. An override can turn an analyzer off
or change its severity for a path (lower or raise it), but cannot enable what the project
disabled.

Glob syntax: slash separated, relative to the project root (a leading `./` is trimmed). `*`, `?`,
`[a-z]` and `[!x]` match inside one segment; `**` matches zero or more segments. The glob is
anchored to the whole path, so write `**/*.test.ts`, not `*.test.ts`. Brace alternatives are
rejected: list each glob.

Per-path params are supported for `file-size` only, and only `max_lines` and `error_lines`
(`extensions` cannot vary per path). `max_lines` must be at least 1; `error_lines` at least 0, and
above `max_lines` when both are set in one block. See the `file-size` section of
`docs/analyzers.md`.

## Limits

Each limit makes the file invalid (exit 1) rather than being trimmed:

| Limit | Value |
| --- | --- |
| File size | 256 KiB, UTF-8, one document, a regular file, no YAML directives |
| Scalar | 1024 bytes |
| Nesting depth | 64 |
| Expanded nodes | 100000 |
| Keys per mapping | 256 |
| Entries per sequence | 1024 |
| One glob | 256 bytes, at most 4 `*` (`**` and a lone `*` segment are free) |
| All globs (`tests` + `overrides`) | 256 globs, 4096 bytes in total |
| Wildcard classes and `?`, all globs | 256 |
| Text after the first `*` of a segment, all globs | 2048 bytes |

## Test files

`tests:` globs only add test files. The built-in rules, by language:

- Go: `_test.go`.
- Dart: `_test.dart`, or any file under `test/` or `integration_test/`.
- TypeScript and JavaScript: `.test.` or `.spec.` in the file name, or any file under `test/`,
  `tests/` or `__tests__/` at any depth, except SvelteKit `+` route files (`+page.ts`, ...), which
  ignore test directories.
- Svelte: only `.test.` or `.spec.` in the file name; a `.svelte` file ignores test directories.
- Other languages: `_test.`, `.test.`, `_spec.` or `.spec.` in the name.

Test files get a looser `file-size` limit (x2.6), a looser function length (x1.5), skip the magic
number check, are not counted by `package-file-count`, and are the subject of `test-comment`.
A plain `.ts` or `.js` module under a directory named `test/`, `tests/` or `__tests__/`, for
example `src/routes/test/helpers.ts`, is classed as a test even inside a SvelteKit route tree,
while `+page.ts` and every `.svelte` file there are not. There is no exclude list.

## Hook-baked flags

The installed hooks pass `--typographic-severity` and `--branch-extra-types`; these set the
matching analyzer params and sit above the file but below per-path overrides.
