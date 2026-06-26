# Installation

The installer is a single Go binary, `cmd/pulsarules_codex-installer`. It embeds the knowledge base
(standards + templates) via `//go:embed`, so it is self-contained: `generate`, `list`, `validate`,
`install`, and `package` all read the embedded snapshot by default.

## Prerequisites

- Go 1.26+ (the module pins `go 1.26`).
- Optional: [Taskfile](https://taskfile.dev/) for the convenience targets.
- Optional but recommended: `gopls` on PATH (`go install golang.org/x/tools/gopls@latest`). When
  present, `install` wires the gopls MCP into the target's `.mcp.json` and (re)generates the
  `gopls-navigation` skill from `gopls mcp -instructions`. If absent, install prints a warning and
  skips only that step.

## Build

```sh
go build -o build/bin/pulsarules_codex-installer ./cmd/pulsarules_codex-installer
# or: task build
```

## Commands

```
pulsarules_codex-installer generate [--root DIR] [--out DIR]
pulsarules_codex-installer validate [--root DIR]
pulsarules_codex-installer list [skills|rules|patterns|workflows] [--root DIR]
pulsarules_codex-installer install [--root DIR] (--global | --project PATH) [--target claude|opencode ...] (--all | --skills a,b,c | --router-only) [--no-hooks] [--no-mcp] [--print-hooks] [--hooks-scope project|local] [--layout PROFILE] [--interactive]
pulsarules_codex-installer package [--root DIR] [--out FILE]
pulsarules_codex-installer version
```

With no `--root`, the binary reads the embedded snapshot. `--root DIR` reads `<DIR>/knowledge` from
disk for dev edits (no rebuild needed).

### Generate

Render every skill into `generated/skills/<id>/SKILL.md` (gitignored; for inspection).

```sh
pulsarules_codex-installer generate
```

### Validate

Check the knowledge base: frontmatter parses, ids unique, composition references resolve, the
mandatory `project-router` exists, composed bodies are non-empty.

```sh
pulsarules_codex-installer validate
```

### List

Print a catalog table to stdout. Kinds: `skills` (default), `rules`, `patterns`, `workflows`.

```sh
pulsarules_codex-installer list            # skills
pulsarules_codex-installer list rules
```

### Install

Render each selected skill in-memory and write it to a Claude skills directory, then install the
skill-router reminder hook and wire it into the target's `settings.local.json`. No pre-generated dir
needed.

```sh
# all skills + hook into a project:
pulsarules_codex-installer install --project /path/to/project --all

# all skills + hook globally:
pulsarules_codex-installer install --global --all

# only the router:
pulsarules_codex-installer install --project . --router-only

# a selected subset (plus the mandatory baseline and any compose_skills deps it pulls in):
pulsarules_codex-installer install --project . --skills go-style,errors-logging,commits

# skills only, no hook:
pulsarules_codex-installer install --project . --all --no-hooks

# inspect the resolved hooks block without writing anything:
pulsarules_codex-installer install --project . --print-hooks
```

`--global` -> `~/.claude/`; `--project PATH` -> `PATH/.claude/`. Mutually exclusive. Installing skills
requires exactly one of `--all`, `--skills a,b,c`, or `--router-only`; with none of them and an
interactive terminal, `install` prompts with a numbered list of every skill (mandatory skills shown
pre-selected and locked) accepting `1,4,7-9` ranges, `a` for all, or Enter for the mandatory-only
baseline. With no flag and no terminal (e.g. CI), it fails rather than guessing. `project-router` plus
every `always_load` skill is always installed and cannot be deselected; a selected skill whose
`compose_skills` names another skill pulls that skill in too (transitively), and `install` prints each
one it added and which skill required it. The skill render is idempotent (overwrites). The hook wiring
is idempotent too: it copies the hook to `.claude/hooks/skill-router-reminder.sh` (executable) and
merges its `SessionStart` + `PreToolUse` entries into the chosen settings file - preserving existing
permissions / `enabledMcpjsonServers` / unrelated hooks, never duplicating on re-run. The wired command
resolves the project root at runtime via `$CLAUDE_PROJECT_DIR`, so it survives moving the repo. It also
ensures `.claude/.gitignore` contains `*` so the installed assets stay local.

Additional install flags:

- `--target` (repeatable, default `claude`): pass `--target claude --target opencode` to install both
  layouts in one run. The **opencode** target renders skills into `<target>/.opencode/skills`, writes
  `<target>/.opencode/AGENTS.md` (which carries the routing contract, since opencode has no SessionStart
  hook), and merges `<target>/opencode.json` with the skills as `instructions` plus the gopls `mcp`
  server.
- `--hooks-scope project|local` (default `project`): wire the hook into `settings.json` (shared) or
  `settings.local.json` (per-machine).
- `--no-mcp`: skip the `.mcp.json` + gopls-navigation generation.
- `--layout PROFILE`: apply a customization profile from `profiles.yaml` (`monorepo` |
  `inner-modules`) that overrides which rules a skill composes (e.g. the `code-placement` layout
  variant). `--interactive` prompts for the layout when it is unset.

### Package

Render all skills in-memory and zip them into `build/standards-skills.zip`.

```sh
pulsarules_codex-installer package
```

## Dev mode

Edit a rule/pattern/workflow `.md` or `skills.yaml` under `knowledge/standards/`, then:

```sh
go run ./cmd/pulsarules_codex-installer validate --root .
go run ./cmd/pulsarules_codex-installer generate --root .
```

`--root .` reads the on-disk `knowledge/` so changes apply without rebuilding. Omit `--root` to use
the embedded snapshot (the shipped behavior).

## No-Go fallback

`knowledge/templates/installers/install.sh.tmpl` is a `jq` + `bash` script (no node) that copies
already-rendered skills (run `generate` first) into a Claude skills directory, wires the hook the same
way the Go installer does (with `--hooks-scope project|local`), and - when `gopls` is on PATH - wires
the gopls MCP into `.mcp.json` plus the generated `gopls-navigation` skill (`--no-mcp` to skip). It
covers the Claude target only; use the Go installer for `--target opencode` and `--layout`. Strip the
`.tmpl` suffix to use it.

## Taskfile targets

```sh
task            # vet + test
task build      # build the binary
task generate   # render skills (gitignored)
task validate   # validate the embedded knowledge
task list       # list skills
task package    # zip skills
task fmt        # gofmt knowledge internal cmd
task lint       # golangci-lint (if installed)
```
