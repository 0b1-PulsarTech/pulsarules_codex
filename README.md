# pulsarules_codex

Engineering standards kept as Markdown, and one Go binary, `pulsarules_cli`, that puts them to
work. The binary embeds the standards and:

- renders them into AI agent skills and installs them into a project or your home directory, for
  Claude Code, Codex, opencode, Cursor, or any agent that reads a root `AGENTS.md`;
- installs hooks that remind the agent to load the skills a file needs, and git hooks that check
  each commit;
- runs the governance analyzers, which check a Go, Dart, TypeScript, JavaScript or Svelte project
  against the standards.

The standards live in `knowledge/standards/`: rules, patterns and workflows as Markdown with YAML
frontmatter, and one `skills.yaml` that composes them into skills. The binary embeds that tree with
`//go:embed` and renders skills on demand; nothing it derives is committed. With `--root DIR` the
commands read `DIR/knowledge` from disk instead, so an edited rule shows up without a rebuild.

## Install

Go 1.27 or newer is the only requirement. The binary needs no cgo and never runs Node.

```sh
go install github.com/0b1-PulsarTech/pulsarules_codex/cmd/pulsarules_cli@latest
```

From a checkout, `task build:bin` writes `build/bin/pulsarules_cli`.

Then install the skills and hooks into a project, or globally:

```sh
pulsarules_cli install --project /path/to/project --all
pulsarules_cli install --global --all
pulsarules_cli install --project . --skills go-style,errors-logging,commits
```

- `--project PATH` writes into the project, under `PATH/.claude/` for the default target;
  `--global` writes for the user, under `~/.claude/` (or `$CLAUDE_CONFIG_DIR`). Give one of them,
  not both. A global install needs no project and no `.claude` directory to exist.
- Pick the skills with `--all`, `--skills a,b,c` or `--router-only`. With none of them on a
  terminal, `install` shows a numbered list; without a terminal it fails. `project-router` and the
  `always_load` skills are always installed, and a skill pulls in the skills it composes.
- `--target claude|codex|opencode|agents|cursor` is repeatable (default `claude`). The table
  below lists what each writes. `opencode`, `agents` and `cursor` are project only and refuse
  `--global`.
- `--languages go,dart` overrides the detected languages, recorded in
  `.pulsarules-languages.json`. A global install belongs to no project, so it scans nothing and
  assumes every language unless `--languages` says otherwise. For each language whose command is
  on PATH, a project install wires its MCP server (`gopls mcp` for Go, `dart mcp-server` for
  Dart) and, for Go, renders the `gopls-navigation` skill from `gopls mcp -instructions`.
  `--no-mcp` skips both.
- `--layout PROFILE` applies a profile from `knowledge/standards/profiles.yaml` (`monorepo`,
  `inner-modules`), which changes the rules some skills compose. `--interactive` asks for it.

| Target | Skills | Also writes |
| --- | --- | --- |
| `claude` | `.claude/skills` | hooks in `.claude/settings.json`; MCP in `.mcp.json` |
| `codex` | `.agents/skills` | `AGENTS.md`; `.codex/hooks.json`; MCP in `.codex/config.toml` |
| `opencode` | `.opencode/skills` | `AGENTS.md`; `opencode.json` (skills, MCP); a plugin |
| `agents` | none | `AGENTS.md` with the routing contract |
| `cursor` | `.cursor/rules/<id>.mdc`, one per skill | one small always-applied pointer rule |

- Under `--global` the `claude` target never writes the MCP servers: Claude rewrites its own user
  file, so `install` prints the `claude mcp add --scope user` command to run instead.
- The `codex` target, project or global, needs two steps in Codex after install: trust the
  project, and approve the hooks with `/hooks`. Its hooks use Codex's own dialect, and a
  `hooks.json` that is a symlink is left alone with a warning. The MCP servers go into one marked
  block of `config.toml`, which `uninstall` removes; a `[mcp_servers.<name>]` entry of your own,
  or a `config.toml` that is a symlink or has an unterminated block, is never replaced.
- A global `codex` install writes skills to `$HOME/.agents/skills` and the rest under
  `~/.codex` (or `$CODEX_HOME`), with `AGENTS.md` there too.
- `AGENTS.md` and the Cursor rules are owned files: each carries a hash of what `install` wrote.
  A file you edited is moved to `<name>.pulsarules-backup` before it is rewritten or removed,
  never lost, and a file `install` did not write is left alone.

Every step is idempotent: re-running `install` overwrites what it wrote and never duplicates a
settings entry. `uninstall` reverses it; with no `--target` it removes every layout it finds
(`--global`: `claude` and `codex`), and `--keep-skills` keeps the rendered skills.

### Hooks

`install` copies `skill-router-reminder.sh` to `.claude/hooks/` and merges its entries
(`SessionStart`, `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `SubagentStart`, `Stop`,
`SessionEnd`) into `settings.json`, or `settings.local.json` with `--hooks-scope local`. Existing
permissions and unrelated hooks are kept. The script runs `pulsarules_cli hook <mode>`, which
injects the routing contract at session start and reminds the agent once per routed file. The
`Stop` hook runs `governance --scope changed` and reports findings as context; it never blocks,
and stops waiting after 60 seconds. `knowledge/templates/hooks/README.md` explains the design.
`--no-hooks` skips the Claude and Codex hooks and `--print-hooks` prints the block without
writing it.

The `codex` target installs the same script under `.codex/hooks/` and wires the same events into
`.codex/hooks.json`, in Codex's hook dialect.
A global install writes the script, binary and log under the user directory (`~/.claude` or
`~/.codex`) and names the script by absolute path in the user settings. The hook finds skills in
the project's skills directory first, then the user's. When the project has its own install, the
global hook exits and lets the project hook run alone, so the agent is never reminded twice. A
global install skips the git hooks, which belong to a repository.

The git hooks are the gate. They go into the repository's shared hooks directory
(`git rev-parse --git-common-dir`), so every worktree gets them:

| Hook | Runs | Installed |
| --- | --- | --- |
| `commit-msg` | `commitlint` | by default |
| `pre-commit` | `governance --scope commit` | by default |
| `pre-push` | `governance` (full scope) | with `--git-hooks commit-msg,pre-commit,pre-push` |

`--no-git-hooks` skips them. A hook gets no flags from the person committing, so two governance
flags are baked in at install time: `--typographic-severity error|warning|info` and
`--branch-extra-types a,b` (pre-push only).

## Commands

| Command | Does |
| --- | --- |
| `install`, `uninstall` | install or remove skills, hooks and MCP wiring, as above |
| `generate [--out DIR]` | render every skill into `generated/skills/` (gitignored) |
| `validate` | check the knowledge base: frontmatter, unique ids, references, the router |
| `list [skills\|rules\|patterns\|workflows]` | print a catalog table |
| `package [--out FILE]` | zip every skill into `build/standards-skills.zip` |
| `commitlint [--msg MSG \| --file FILE] [--project DIR]` | check a commit message |
| `governance` | run the analyzers; exit 1 on any error finding |
| `clean [--project DIR] [--write]` | report AI watermark characters; `--write` removes some |
| `evals --artifact FILE [--scenario ID]` | grade an artifact against the skill scenarios |
| `hook <mode>` | the entry point the installed hook scripts call |
| `version` | print the version |

`pulsarules_cli --help` lists every flag. Flags go before a positional argument: parsing stops at
the first one.

`commitlint` checks the emoji, format, type, scope and body of a message; with `--project` it also
checks the emoji against recent history.

`governance` analyzes the files `git status` reports as changed, or every file git lists with
`--all-files`. `--scope commit` runs the fast static checks over staged files, `--scope changed`
adds the architecture analyzers, and `--scope full` (the default) adds golangci-lint, the
delegated commands and `branch-name`. Findings in generated files are dropped unless
`--include-generated` is given. The analyzers, their default severities and their params are
listed in [docs/analyzers.md](docs/analyzers.md).

`clean` handles `.go` and `.md` files. It never rewrites inside a string literal or a fenced
block, where a character may be data; those are reported for a person to judge.

## Configuration

A project tunes governance with an optional `.pulsarules.yaml` at its root:

```yaml
version: 1                  # required; 1 is the only value
preset: recommended         # recommended | strict | minimal
include_generated: false
skip_dirs: [vendor]         # directory basenames
tests: ["**/fixtures/**"]   # extra globs that count as test files
analyzers:                  # keyed by analyzer id
  file-size:
    severity: warning       # error | warning | info
    params: {max_lines: 220}
  doc-comment: {enabled: false}
overrides:                  # per-path settings
  - files: ["internal/gen/**"]
    analyzers:
      file-size: {params: {max_lines: 600}}
      import-groups/path: {enabled: false}
delegated:                  # see Delegated commands
  - id: dart-analyze
    format: dart-analyze-json
    run: [dart, analyze, --format=json]
```

- Precedence, lowest first: built-in defaults, the preset (`--preset`, else the highest layer
  that names one, else `recommended`), the layers below, the command line flags, then
  `overrides`.
- `recommended` changes nothing. `strict` lowers the `complexity` limits to 10, 50 function lines
  and 4 parameters. `minimal` turns off the architecture analyzers, golangci-lint and most AST and
  comment checks, and limits `single-letter-name` to upper-case declaration names.
- Without `--config`, three files are layered, each merged key by key over the one before:
  a user file, `<user-config-dir>/pulsarules/pulsarules.yaml` (`$XDG_CONFIG_HOME` or
  `~/.config` on Linux), then the project's `.pulsarules.yaml`, then a private file,
  `<git-common-dir>/pulsarules.yaml`. All three are optional and take the same keys. The user
  file holds your own defaults for every project; the private file is never committed and is
  shared by every worktree.
- A layer above wins: scalars and `analyzers` fields take the higher value, `tests` and
  `skip_dirs` are joined, `overrides` are appended, and a `delegated` entry with the same `id`
  replaces the lower one. Disabling an analyzer in a layer also drops the `overrides` of the
  layers below it for that analyzer. What the user file switches off never invalidates an
  override that enables the analyzer for some paths. `--config PATH` reads that one file and
  nothing else.
- Override globs are relative to the project root; `*` stays inside one segment and `**` spans
  segments. An override can turn an analyzer off or change its severity for a path, but cannot
  enable one the project disabled. Per-path params exist only for `file-size`.
- Sub-ids such as `import-groups/path` or `golangci-lint/errcheck` work only inside `overrides`.
- A broken file is an error, never a silent default: an unknown key, analyzer, param or preset
  fails `governance` and both commit hooks, and the `Stop` hook reports it. So does a list param
  with an item that is not a string: quote numbers, as in `allowed_numbers: ["0", "24"]`.

## Languages

| Language | Extensions | Parser | Needs |
| --- | --- | --- | --- |
| Go | `.go` | `go/parser` | nothing |
| TypeScript, JavaScript | `.ts .tsx .mts .cts .js .jsx .mjs .cjs` | built in | nothing |
| Svelte | `.svelte` | built in | nothing |
| Dart | `.dart` | a helper compiled with `dart compile exe` | `dart` on PATH |

The syntax based analyzers (names, imports, function length, nesting, ...) need a parser; the text
analyzers run on every file. A file that does not parse gets one `<lang>-syntax/unparsed` warning
and no syntax checks. A language whose parser cannot run gets one `<lang>-syntax/unavailable`
info: without `dart` on PATH, only the Dart syntax checks are skipped.

- TypeScript, JavaScript and Svelte are parsed by a typescript-go fork linked into the binary,
  in a child process with per-file limits (256 MiB, 10 seconds, 1 MiB of source).
- The Dart helper is built once, which needs a populated pub cache or network, and is cached
  under the user cache directory. A failed build is not retried for 10 minutes.
- Dart is checked by `control-flow` (`forbid_else` included), `magic-number` (Go and Dart only),
  `forbidden-token` and `single-letter-name`, among others. `commit-move-purity` follows Dart
  import rewrites, so moving a file and fixing its imports in one commit is not an edit.
- Test files are recognised per language (`_test.go`, `_test.dart` or `test/`, `.test.` or
  `.spec.`, ...) and get looser limits; the `tests` key adds globs.

## Delegated commands

A project can hand a check to its own tool and get the output back as findings. The one format
is `dart-analyze-json`:

```yaml
delegated:
  - id: dart-analyze            # unique; not a built-in analyzer id
    dir: apps/mobile            # optional; must stay inside the project
    format: dart-analyze-json
    rule: dart-style            # optional rule attached to the findings
    run: [dart, analyze, --format=json]
```

- `run` is executed without a shell, so a committed `.pulsarules.yaml` deserves the same review
  as a committed Taskfile.
- Delegated commands run only at full scope: a plain `governance` run and the `pre-push` hook.
- Each run has a 5 minute timeout and a 32 MiB cap per output stream.
- Findings are `<id>/<code>` with the tool's severity, plus `<id>/unavailable` (the tool is
  missing, or `dart pub get` has not run), `<id>/config`, `<id>/timeout`, `<id>/failed` and
  `<id>/dropped` (diagnostics on paths outside the project). Tune them under `analyzers.<id>` or,
  for one code or path, in `overrides` with a sub-id such as `dart-analyze/unused_import`.

## Credits and licences

This project is licensed under the MIT licence in `LICENSE`. The binary links these modules:

- `github.com/Zzzen/typescript-go`, a fork of `microsoft/typescript-go` (Apache-2.0): the
  TypeScript, JavaScript and Svelte parser.
- `github.com/go-json-experiment/json` (BSD-3-Clause).
- `github.com/wrapped-owls/goremy-di` (MIT): dependency injection.
- `golang.org/x/mod`, `golang.org/x/sync`, `golang.org/x/sys`, `golang.org/x/term` and
  `golang.org/x/text` (BSD-3-Clause).
- `gopkg.in/yaml.v3` (MIT and Apache-2.0).

`go version -m <binary>` prints the exact versions.
