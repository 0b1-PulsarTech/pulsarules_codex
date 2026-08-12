# Installation

The installer is a single Go binary, `cmd/pulsarules_cli`. It embeds the knowledge base
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
go build -o build/bin/pulsarules_cli ./cmd/pulsarules_cli
# or: task build:bin
```

## Commands

```
pulsarules_cli generate [--root DIR] [--out DIR]
pulsarules_cli validate [--root DIR]
pulsarules_cli list [skills|rules|patterns|workflows] [--root DIR]
pulsarules_cli install [--root DIR] (--global | --project PATH) [--target claude|opencode|agents|cursor ...] (--all | --skills a,b,c | --router-only) [--no-hooks] [--no-mcp] [--print-hooks] [--hooks-scope project|local] [--layout PROFILE] [--interactive] [--git-hooks commit-msg,pre-commit] [--no-git-hooks]
pulsarules_cli uninstall (--global | --project PATH) [--target claude|opencode|agents|cursor ...] [--hooks-scope project|local] [--keep-skills]
pulsarules_cli package [--root DIR] [--out FILE]
pulsarules_cli commitlint [--msg MSG | --file FILE] [--project DIR]
pulsarules_cli governance [--project DIR] [--root DIR] [--preset recommended|strict|minimal] [--scope full|commit] [--golangci-config PATH] [--all-files] [--include-generated]
pulsarules_cli version
```

With no `--root`, the binary reads the embedded snapshot. `--root DIR` reads `<DIR>/knowledge` from
disk for dev edits (no rebuild needed).

### Generate

Render every skill into `generated/skills/<id>/SKILL.md` (gitignored; for inspection).

```sh
pulsarules_cli generate
```

### Validate

Check the knowledge base: frontmatter parses, ids unique, composition references resolve, the
mandatory `project-router` exists, composed bodies are non-empty.

```sh
pulsarules_cli validate
```

### List

Print a catalog table to stdout. Kinds: `skills` (default), `rules`, `patterns`, `workflows`.

```sh
pulsarules_cli list            # skills
pulsarules_cli list rules
```

### Install

Render each selected skill in-memory and write it to a Claude skills directory, then install the
skill-router reminder hook and wire it into the target's `settings.json` (the `--hooks-scope`
default; pass `--hooks-scope local` to wire `settings.local.json` instead). No pre-generated dir
needed.

```sh
# all skills + hook into a project:
pulsarules_cli install --project /path/to/project --all

# all skills + hook globally:
pulsarules_cli install --global --all

# only the router:
pulsarules_cli install --project . --router-only

# a selected subset (plus the mandatory baseline and any compose_skills deps it pulls in):
pulsarules_cli install --project . --skills go-style,errors-logging,commits

# skills only, no Claude hook (git hooks still install; add --no-git-hooks):
pulsarules_cli install --project . --all --no-hooks

# inspect the resolved hooks block without writing anything:
pulsarules_cli install --project . --print-hooks
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
merges its `SessionStart`, `PreToolUse`, `PostToolUse`, `UserPromptSubmit`, `SubagentStart`, `Stop`
and `SessionEnd` entries into the chosen settings file - preserving existing
permissions / `enabledMcpjsonServers` / unrelated hooks, never duplicating on re-run. The wired command
locates the installed hook script via `$CLAUDE_PROJECT_DIR` (a Claude Code variable, legitimately
named there); the script then exports `PULSARULES_PROJECT_DIR` and `PULSARULES_SKILLS_DIR` - the
only two variables the binary itself reads - so it survives moving the repo without hardcoding a
host's own variable name or a host's own skills layout. It also stamps `.claude/.gitignore` with `/bin/` and `/hooks/` (plus the ownership marker
`Remove` needs), so the copied binary and hook script stay local while `settings.json` and the
rendered skills - which carry their own `.gitignore` - remain yours to commit or not.

**Upgrading from an install that predates `PULSARULES_PROJECT_DIR`/`PULSARULES_SKILLS_DIR`:** an
older `skill-router-reminder.sh` exports neither, so the binary resolves an empty project dir and
the `stop`, `pre-search`, and `post-edit` checks all go silently quiet. The binary now prints a
one-time stderr warning naming the fix - re-run `install` (this command) to regenerate the hook
script.

Additional install flags:

- `--target` (repeatable, default `claude`): pass `--target claude --target opencode` to install more
  than one layout in one run. The **opencode** target renders skills into `<target>/.opencode/skills`
  and merges `<target>/opencode.json` with the skills as `instructions` plus the gopls `mcp` server.
  Both **opencode** and the thin **agents** target write a single `<target>/AGENTS.md` at the project
  root (carrying the mandatory routing contract, since a host AI agent may read nothing else) from the
  same builder, so the file can never diverge between them. `agents` writes nothing else - it exists
  for the AI coding agents (Codex, Gemini CLI, Zed, Amp, JetBrains Junie, Jules, and others) that read
  only a repo-root `AGENTS.md`. The **cursor** target renders skills into `<target>/.cursor/rules` as
  one `.mdc` file per skill (Cursor's own rule format), each carrying `alwaysApply: false` so Cursor
  pulls it in on demand by matching its `description`, plus one small `alwaysApply: true` pointer rule
  carrying the routing contract. Cursor injects every `alwaysApply: true` rule into EVERY request
  (unlike Claude's session/file-scoped hooks), and Cursor's own guidance keeps that content near 2000
  tokens; the five `always_load` skills alone render to ~52KB (~13k tokens), so none of them are
  `alwaysApply` here - only the compact pointer rule (under 900 bytes) is.
- `--hooks-scope project|local` (default `project`): wire the hook into `settings.json` (shared) or
  `settings.local.json` (per-machine).
- `--no-mcp`: skip the `.mcp.json` + gopls-navigation generation.
- `--layout PROFILE`: apply a customization profile from `profiles.yaml` (`monorepo` |
  `inner-modules`) that overrides which rules a skill composes (e.g. the `code-placement` layout
  variant). `--interactive` prompts for the layout when it is unset.
- `--git-hooks commit-msg,pre-commit` (default `commit-msg,pre-commit`): comma-separated git hook
  scripts to install into `<target>/.git/hooks/` that delegate to the installed binary. Valid names
  are `commit-msg`, `pre-commit`, and `pre-push`; an unrecognized name fails the whole `install`
  command before anything is written, naming the valid set. `--no-git-hooks` skips installing git
  hooks entirely.

### Uninstall

Reverse `install`. Unlike `install`, it does not default to the `claude` target: with no `--target`
it probes the project for every layout it can detect on disk (a `.claude` dir, a `.opencode` dir /
`opencode.json`, a root `AGENTS.md`, or a `.cursor/rules` dir) and reverses all of them; pass
`--target` to narrow it to one layout. It unwires the hook from both `settings.json` and
`settings.local.json` by default (it cannot recover which `--hooks-scope` install used); pass
`--hooks-scope` to narrow to one file. It also removes the gopls entry from `.mcp.json`, the
opencode plugin plus its `opencode.json` wiring, and the git hooks `install` wrote into
`.git/hooks/`. A root `AGENTS.md` is removed only when its content still carries the marker
`install` wrote it with, so a hand-authored one is never touched. Unless `--keep-skills` is given,
it also removes every rendered skill and workflow doc `install` wrote (a user's own files placed
inside a rendered skill's directory are never touched). Every step is idempotent, so running it
against an install that was never made, or running it twice, is not an error.

```sh
pulsarules_cli uninstall --project /path/to/project --target claude
pulsarules_cli uninstall --global --keep-skills
```

### Package

Render all skills in-memory and zip them into `build/standards-skills.zip`.

```sh
pulsarules_cli package
```

### Commitlint

Validate a commit message against the emoji, format, type, scope, and body rules. `--msg` checks a
literal string; `--file PATH` reads a `COMMIT_EDITMSG` file (the git `commit-msg` hook passes `$1`);
`--project DIR` enables emoji-variance checks against recent git history. Exits non-zero on
error-severity findings; this is what the installed `commit-msg` git hook runs.

```sh
pulsarules_cli commitlint --file .git/COMMIT_EDITMSG --project .
```

### Governance

Run the analyzer pipeline (static, AST, architecture, delegation, rule injection) against the
project and print findings to stderr, exiting non-zero on any error-severity finding. By default it
analyzes only files `git status` reports as changed; `--all-files` walks the whole source tree
instead (skipping `.git/`, `.claude/`, `.opencode/`, `generated/`, `build/`, and `vendor/`).
`--scope commit` runs the lightweight static + AST + commit checks only (what the pre-commit hook
uses); `--scope full` (default) adds architecture and delegation. `--preset
recommended|strict|minimal` selects analyzer thresholds. Findings in generated files (detected by
the `// Code generated ... DO NOT EDIT.` marker) are suppressed by default; `--include-generated`
turns that off.

```sh
# the exact governance gate task tools:governance / CI run:
pulsarules_cli governance --project . --all-files

# full sweep, explicit scope:
pulsarules_cli governance --scope full --all-files
```

## Dev mode

Edit a rule/pattern/workflow `.md` or `skills.yaml` under `knowledge/standards/`, then:

```sh
go run ./cmd/pulsarules_cli validate --root .
go run ./cmd/pulsarules_cli generate --root .
```

`--root .` reads the on-disk `knowledge/` so changes apply without rebuilding. Omit `--root` to use
the embedded snapshot (the shipped behavior).

## Taskfile targets

```sh
task                   # vet + test
task vet                # go vet (CGO-free)
task test                # go test (CGO-free, no -race)
task test:race            # tests under the race detector (needs CGO)
task tools:fmt            # format with the project formatter (gofumpt + golines + goimports)
task tools:lint            # golangci-lint
task tools:vuln            # govulncheck (pinned version)
task tools:governance      # this repo's own analyzers against the full tree - the CI governance gate
task tools:mocks           # regenerate colocated mocks
task build:bin             # build the binary
task build:package         # zip skills
task gen:skills            # render skills (gitignored)
task gen:validate          # validate the embedded knowledge
task gen:emoji             # regenerate the commit emoji catalog
task list                  # list skills
```
