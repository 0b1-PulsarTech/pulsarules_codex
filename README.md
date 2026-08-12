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
frontmatter, plus a single `skills.yaml` defining every skill composition). The Go binary embeds
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
go run ./cmd/pulsarules_cli validate   # check the embedded knowledge
go run ./cmd/pulsarules_cli list        # print the skill catalog
go run ./cmd/pulsarules_cli generate    # render generated/skills/ (gitignored)
go run ./cmd/pulsarules_cli install --project /path/to/project
go run ./cmd/pulsarules_cli package     # zip all skills into build/
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
│   │   ├── skills.yaml              # every skill composition (single file)
│   │   ├── examples/  references/
│   │   └── README.md
│   └── templates/                   # skills/  hooks/  docs/  installers/  mcp/ (embedded)
├── internal/                        # governance pipeline, hook dispatcher, skill render/install
├── cmd/pulsarules_cli/              # CLI entrypoint (one file per command)
├── README.md  INSTALL.md  ARCHITECTURE.md
├── Taskfile.yml  go.mod  go.sum  .gitignore
```

This is an orientation sketch, not an exhaustive tree - it rots the moment a package moves. See
[ARCHITECTURE.md](ARCHITECTURE.md#packages) for the current, package-by-package breakdown of
`internal/*` (including every `internal/skill/*` operation package) and `cmd/pulsarules_cli`. The old
`manifests/` tree is gone - metadata lives in frontmatter + `skills.yaml`. Hook assets are embedded
under `knowledge/templates/hooks/`. `generated/` is produced on demand and gitignored.

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
routing contract at `SessionStart` and a pointed reminder at the first write to each file the router
matches to a skill via `Router.SkillsForFile` (non-blocking, always exit 0; gated per file path, not
once per session, so a session touching several routed files gets reminded once per file). See
[`knowledge/templates/hooks/README.md`](knowledge/templates/hooks/README.md) for the full rationale
and design constraints - it is installed alongside the hook.

`install --project PATH` (or `--global`) installs the hook into `PATH/.claude/hooks` and wires it
into a settings file with an idempotent merge (preserves existing permissions /
`enabledMcpjsonServers` / unrelated hooks; re-running never duplicates). `--hooks-scope project` (the
default) targets `settings.json`; `--hooks-scope local` targets `settings.local.json`. The wired
command locates the installed hook script via `$CLAUDE_PROJECT_DIR` (a Claude Code variable,
legitimately named there); the script then exports `PULSARULES_PROJECT_DIR` and
`PULSARULES_SKILLS_DIR` - the only two variables the binary itself reads - so it survives moving the
repo without hardcoding a host's own variable name or a host's own skills layout. An install
predating this rename has a hook script with no such exports, so the binary silently resolves an
empty project dir and the `stop`/`pre-search`/`post-edit` checks go quiet; it now warns once on
stderr naming the fix (`pulsarules_cli install`) - re-run `install` to pick up the new script. When
`gopls` is on PATH, install also wires the gopls MCP into `.mcp.json` and regenerates the
`gopls-navigation` skill (`--no-mcp` to skip). `--target` is repeatable
(`claude`/`opencode`/`agents`/`cursor`): the opencode target writes `.opencode/skills` and
`opencode.json`; both opencode and the thin `agents` target write a single `AGENTS.md` at the project
root (carrying the routing contract) from the same builder, so the file can never diverge between
them - `agents` writes nothing else, covering the AI coding agents that read only a repo-root
`AGENTS.md`. The `cursor` target writes `.cursor/rules/<id>.mdc`, one file per skill, each
`alwaysApply: false` so Cursor pulls it in on demand from its `description`; only a small pointer
rule carrying the routing contract is `alwaysApply: true`, since Cursor injects every
`alwaysApply: true` rule into every request rather than firing once per session like the Claude hook.
Use `--no-hooks` to skip the Claude hook script and settings wiring (git hooks: `--no-git-hooks`), or `--print-hooks` to print the resolved hooks block. A
no-Go fallback lives at `knowledge/templates/installers/install.sh.tmpl` (`jq` + `bash`, no node).

## Origin

This repository is the source of truth for these standards. It was initially seeded from the
engineering conventions of several Go services by the same author. The standards here stand on their own - application projects copy
from this repo, never the reverse.
