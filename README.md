# pulsarules_codex

A knowledge-first repository of reusable engineering standards, rules, patterns, and workflows for
Go services - plus a small Go installer that embeds them and renders Claude skills on demand.

> **Knowledge is the primary asset. Skills are an output artifact.**
> Markdown (with YAML frontmatter) is the canonical source of truth. The repo stays useful even if
> Claude Code disappears.

## What this is

Every project re-discovers the same engineering contract: how to name things, wrap errors, wire
dependencies, lay out a monorepo, run migrations, write tests, commit, review. This repo captures
that contract **once**, independent of any product domain, so projects consume it instead of
re-authoring it.

The knowledge lives in `knowledge/standards/` (rules, patterns, workflows as markdown with
frontmatter, plus a single `skills.yaml` defining the 29 skill compositions). The Go binary embeds
it via `//go:embed` and renders skills on demand - it never hand-writes them and nothing derived is
committed.

## Philosophy (order of importance)

1. **Standards** - the contract.
2. **Rules** - mandatory, enforceable.
3. **Patterns** - how to apply the rules (copyable code).
4. **Workflows** - the order to apply them for a task.
5. **Metadata** - frontmatter on each item + `skills.yaml` for composition.
6. **Skills** - generated from the above; one consumer of the model.
7. **Tooling** - the embedded installer that does the generation.

## What lives here vs in application repos

| Here (portable)                                      | Application repos (not moved)        |
|------------------------------------------------------|--------------------------------------|
| Go conventions, naming, imports, types               | Business logic, product requirements |
| Errors, logging, concurrency, security               | Customer-specific rules              |
| Code placement, startup, config, DI                  | Domain-specific concepts             |
| Transport-agnostic use cases, interop, gRPC          | Application-specific workflows       |
| Authorization model, database chain, eventing/outbox | Product specs                        |
| Testing, HTTP clients, build, minimalism             | The product itself                   |

## Quick start

```sh
go run ./cmd/pulsarules_codex-installer validate   # check the embedded knowledge
go run ./cmd/pulsarules_codex-installer list        # print the skill catalog
go run ./cmd/pulsarules_codex-installer generate    # render generated/skills/ (gitignored)
go run ./cmd/pulsarules_codex-installer install --project /path/to/project
go run ./cmd/pulsarules_codex-installer package     # zip all skills into build/
```

With no `--root`, the binary reads the embedded snapshot. Pass `--root DIR` to read
`<DIR>/knowledge` from disk, so you can edit a rule and re-run without rebuilding.

See [INSTALL.md](INSTALL.md) and [ARCHITECTURE.md](ARCHITECTURE.md).

## Repository layout

```
pulsarules_codex/
├── knowledge/                       # Go package: embeds + parses the knowledge base
│   ├── embed.go                     # //go:embed standards templates
│   ├── model.go  frontmatter.go  index.go
│   ├── standards/
│   │   ├── rules/<id>.md            # rule + YAML frontmatter
│   │   ├── patterns/<id>.md         # pattern + frontmatter
│   │   ├── workflows/<id>.md        # workflow + frontmatter
│   │   ├── skills.yaml              # the 29 skill compositions (single file)
│   │   ├── examples/  references/
│   │   └── README.md
│   └── templates/
│       ├── skills/{SKILL.md.tmpl, router.md.tmpl}
│       ├── hooks/{skill-router-reminder.sh, README.md, settings.hooks.json.tmpl}
│       ├── docs/  installers/
├── internal/skill/
│   ├── render/                      # transclude composed bodies into a SKILL.md
│   ├── validate/                    # knowledge-base integrity checks
│   ├── output/                      # install / generate / package the rendered skills
│   └── hookwire/                    # install the hook + idempotently wire settings.local.json
├── cmd/pulsarules_codex-installer/  # CLI (one file per command)
├── README.md  INSTALL.md  ARCHITECTURE.md
├── Taskfile.yml  go.mod  go.sum  .gitignore
```

Go packages: `knowledge` (embed + parse) and the `internal/skill/*` operation packages
(`render`, `validate`, `output`, `hookwire`), driven by `cmd` (CLI). The old `manifests/` tree is
gone - metadata lives in frontmatter + `skills.yaml`. Hook assets are embedded under
`knowledge/templates/hooks/`. `generated/` is produced on demand and gitignored.

## Customization

- **Add a rule:** write `knowledge/standards/rules/<id>.md` (with frontmatter), reference it from a
  skill in `knowledge/standards/skills.yaml` (`compose_rules`). Rebuild.
- **Add a skill:** add an entry to `skills.yaml` declaring its `compose_rules`/`compose_patterns`
  /`compose_workflows`/`triggers`. Rebuild - the skill is rendered by transcluding its composed
  knowledge.
- **Generalize a project convention:** extract the principle into a rule/pattern here; keep the
  product-specific instantiation in the application repo.
- **Add a customization profile:** add an entry to `knowledge/standards/profiles.yaml` with an axis and
  per-skill `compose_rules`/`compose_patterns` overrides (e.g. a `code-layout` variant), and author the
  variant rule files it references. Install with `--layout <profile>`; the override is applied to a
  freshly-loaded index before rendering, so one knowledge base serves every project shape.

## Why hooks?

Claude Code's autonomous skill activation is unreliable: the agent invokes `project-router`, sees the
dispatch table, then skips to implementation without actually LOADING and APPLYING every matched
skill when it writes code. A skill description is a suggestion the model can ignore; a **hook fires
deterministically**. The embedded `knowledge/templates/hooks/skill-router-reminder.sh` injects the
routing contract at `SessionStart` and a pointed reminder at the first `.go`/`.sql` write of a
session (non-blocking, always exit 0, fires once per session via a per-session flag). See
[`knowledge/templates/hooks/README.md`](knowledge/templates/hooks/README.md) for the full rationale
and design constraints - it is installed alongside the hook.

`install --project PATH` (or `--global`) installs the hook into `PATH/.claude/hooks` and wires it
into a settings file with an idempotent merge (preserves existing permissions /
`enabledMcpjsonServers` / unrelated hooks; re-running never duplicates). `--hooks-scope project` (the
default) targets `settings.json`; `--hooks-scope local` targets `settings.local.json`. The wired
command resolves the project root at runtime via `$CLAUDE_PROJECT_DIR`, so it survives moving the
repo. When `gopls` is on PATH, install also wires the gopls MCP into `.mcp.json` and regenerates the
`gopls-navigation` skill (`--no-mcp` to skip). `--target` is repeatable (`claude`/`opencode`): the
opencode target writes `.opencode/skills`, `.opencode/AGENTS.md`, and `opencode.json`. Use
`--no-hooks` to install skills only, or `--print-hooks` to print the resolved hooks block. A no-Go
fallback lives at `knowledge/templates/installers/install.sh.tmpl` (`jq` + `bash`, no node).

## Origin

This repository is the source of truth for these standards. It was initially seeded from the
engineering conventions of several Go services by the same author. The standards here stand on their own - application projects copy
from this repo, never the reverse.
