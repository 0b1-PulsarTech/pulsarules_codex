# Installation

The CLI is a single Go binary, `cmd/pulsarules_cli`. It embeds the knowledge base (standards +
templates) via `//go:embed`, so it is self-contained: `generate`, `list`, `validate`, and `package`
all read the embedded snapshot by default.

## Prerequisites

- Go 1.26+ (the module pins `go 1.26`).
- Optional: [Taskfile](https://taskfile.dev/) for the convenience targets.

## Build

```sh
go build -o build/bin/pulsarules_cli ./cmd/pulsarules_cli
# or: task build
```

## Commands

```
pulsarules_cli generate [--root DIR] [--out DIR]
pulsarules_cli validate [--root DIR]
pulsarules_cli list [--root DIR] [skills|rules|patterns|workflows]
pulsarules_cli package [--root DIR] [--out FILE]
pulsarules_cli commitlint [--msg MSG | --file FILE] [--project DIR]
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

### Package

Render all skills in-memory and zip them into `build/standards-skills.zip`.

```sh
pulsarules_cli package
```

### Commitlint

Validate a commit message against the emoji, format, type, scope, and body rules. `--msg` checks a
literal string; `--file PATH` reads a `COMMIT_EDITMSG` file (the git `commit-msg` hook passes `$1`);
`--project DIR` enables emoji-variance checks against recent git history. Exits non-zero on
error-severity findings.

```sh
pulsarules_cli commitlint --file .git/COMMIT_EDITMSG --project .
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
task            # vet + test
task build      # build the binary
task generate   # render skills (gitignored)
task validate   # validate the embedded knowledge
task list       # list skills
task package    # zip skills
task fmt        # gofmt knowledge internal cmd
task lint       # golangci-lint (if installed)
```
