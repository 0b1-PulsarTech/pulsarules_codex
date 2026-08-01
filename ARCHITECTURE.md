# Architecture

This document describes the knowledge model and skill-rendering pipeline: how a rule becomes a
`SKILL.md`.

## Knowledge model

Every reusable item is one of four kinds. The first three are authored as markdown with YAML
frontmatter; the fourth is generated.

| Kind     | Location                                | Role                                                                  |
|----------|-----------------------------------------|-----------------------------------------------------------------------|
| Rule     | `knowledge/standards/rules/<id>.md`     | Mandatory, enforceable principle (what must hold)                     |
| Pattern  | `knowledge/standards/patterns/<id>.md`  | Implementation recipe with code (how to build it)                     |
| Workflow | `knowledge/standards/workflows/<id>.md` | Ordered sequence for a task (in what order)                           |
| Sidecar  | `knowledge/standards/skills/<id>.md`    | Authored orientation per skill (what it governs, when to reach for it) |
| Skill    | `generated/skills/<id>/SKILL.md`        | Generated capability that composes rules + patterns + workflows       |
| Profile  | `knowledge/standards/profiles.yaml`     | Install-time customization that overrides a skill's composition       |
| Router   | `knowledge/standards/router.yaml`       | The router's baseline, dispatch table, and order as filterable data   |

A **skill** is never authored directly. `knowledge/standards/skills.yaml` declares the 29 skills,
each naming the rules, patterns, and workflows it composes (`compose_rules`, `compose_patterns`,
`compose_workflows`), plus triggers, load policy (`always_load`), composition `order`, and
skill-level dependencies (`compose_skills`). The installer renders a skill by transcluding the
markdown body of its **sidecar** (`knowledge/standards/skills/<id>.md`) as the skill head -
orientation only, saying what the skill governs and when to reach for it - then **transcluding** the
markdown bodies of its composed rules, patterns, and workflows under `###` sub-headings as the
reference detail, wrapped in a frontmatter + activation + composition shell. A sidecar is required
for every non-router skill (validate enforces it); `project-router` renders from its own template
and carries no sidecar.

Edit a rule's or sidecar's markdown, rebuild, and every skill that composes it updates - skills never
drift from the knowledge.

`knowledge/standards/profiles.yaml` declares install-time profiles, each on an axis (e.g.
`code-layout`) with per-skill `compose_rules`/`compose_patterns` overrides. Selecting a profile
(`install --layout <id>`) applies the override to a freshly-loaded index via `Index.ApplyProfiles`
before rendering, so one knowledge base serves several project shapes (`monorepo` vs
`inner-modules`) with no skill duplication. `validate` checks every override resolves to an
existing skill and to rules/patterns with non-empty bodies, so a selected variant is guaranteed to
render rather than be silently dropped.

`knowledge/standards/router.yaml` holds the project-router's baseline, dispatch table, and
composition order as data. When `install --skills <subset>` selects only some skills,
`Selection.Resolve` adds the mandatory baseline (`project-router` plus every `always_load` skill)
and, transitively, every skill a selected skill's `compose_skills` names (cycle-safe; `install`
prints what got pulled in and by which skill); the renderer then filters the router to that
resolved set, dropping baseline entries, dispatch rows, and order steps that name no installed
skill, so the orchestrator never advertises a skill that was not installed. `install --all` or
`--router-only` passes no filter and renders the complete router.

## Frontmatter

```markdown
---
id: errors
name: Error Handling
description: Error handling standards for Go services
tags: [go, errors]
dependencies: [logging]
tool_bindings: [apperr]
linters: [errcheck, wrapcheck]
---
# Error Handling
> ...body...
```

Patterns add `composes`; workflows add `steps`, `composes_rules`, `composes_patterns`. The
frontmatter is the metadata. This repo is the source of truth; application repos copy from it, not
the other way.

`skills.yaml`:

```yaml
skills:
  - id: errors-logging
    name: Errors & Logging
    description: Error wrapping + structured logging for Go services
    triggers:
      - returning/wrapping/mapping an error
      - adding any log statement
    always_load: true
    order: 20
    compose_rules: [errors, logging]
```

## Embed + dev mode

`knowledge/embed.go` does `//go:embed standards templates`, so the binary carries the committed
knowledge snapshot. `knowledge.Load(root)`:

- `root == ""` -> read `embed.FS` (the shipped behavior).
- `root != ""` -> read `<root>/knowledge` from disk (dev edits without rebuild).

Both return the same `*Index` (typed items + by-id maps + bodies) and a `fs.FS` for templates.

## Generation pipeline (skill-rendering)

```
knowledge/embed.go  ──//go:embed──>  embed.FS (standards + templates)
        │
        ▼
knowledge.Load(root)  ──>  *Index (rules, patterns, workflows, skills, bodies)  +  templates fs.FS
        │
        ▼
render.NewRenderer(templates)  ──text/template──>  SKILL.md (transcluded bodies)
        │
        ├── output.Generate   -> generated/skills/<id>/SKILL.md   (gitignored)
        ├── output.Install    -> <dest>/<id>/SKILL.md             (rendered in-memory)
        ├── output.Package    -> build/standards-skills.zip       (rendered in-memory)
        ├── validate.Validate -> integrity check
        └── hookwire.{InstallHook,WireSettings} + gitignore.Ensure  -> .claude/hooks + settings.json (settings.local.json with --hooks-scope local)
```

## Packages

| Package                          | Responsibility                                                                                                                |
|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| `knowledge`                      | `embed.go` (embed), `model.go` (Rule/Pattern/Workflow/Skill), `frontmatter.go` (fence parser), `load.go` + `read*.go` (loading), `index.go` (Index) |
| `internal/skill/render`          | `render.go` (Renderer + transclusion), `compose.go`, `router*.go`, `sidecar.go`, `namespace.go` (template funcs)              |
| `internal/skill/validate`        | `validate.go` (pipeline), `checks.go` (reference + body integrity checks)                                                     |
| `internal/skill/output`          | `install.go`, `selection.go` (Selection), `generate.go`, `package.go`, `write.go`                                             |
| `internal/skill/hookwire`        | `hooks.go` (install hook), `settings.go` (idempotent settings.json/.local merge, `$CLAUDE_PROJECT_DIR`)             |
| `internal/skill/mcpwire`         | `mcp.go` (merge gopls into `.mcp.json`), `gopls.go` (PATH check + generate gopls-navigation from `gopls mcp -instructions`)   |
| `internal/skill/opencodewire`    | `agents.go` (render `.opencode/AGENTS.md`), `config.go` (merge `opencode.json`: instructions + gopls mcp)                     |
| `internal/skill/target`          | `target.go` (`Target`/`Context`/`Report`), `registry.go` (`Registry`), `claude.go`, `opencode.go` - one Strategy per install layout |
| `cmd/pulsarules_cli`             | `main.go` only: parse args, run `bootstrap.DoInjections`, dispatch to `internal/cli`, the one place that calls `os.Exit` |

`knowledge` is the asset + embed anchor; the `internal/skill/*` packages are cohesive operation
units (most source files carry a name-matched `_test.go`); commands live in `internal/cli`, and
`cmd/pulsarules_cli` is `main.go` only. Installing skills, the hook, and the settings wiring is one
`install` command - hook assets are embedded under `knowledge/templates/hooks/`.

## The router skill

`project-router` is mandatory and always-loaded first. Its dispatch table maps a task signal to the
skills to load, in composition order; the rows live in `knowledge/standards/router.yaml` and
`knowledge/templates/skills/router.md.tmpl` renders them. The rows include engineering tasks such as
new app, bootstrap/DI, use case, REST/gRPC endpoint, entity/migration, sqlc query, multi-write tx,
outbox event, worker, permission, external provider, test, module boundary, refactoring and
architecture review. The vertical Dependency Rule (`dependency-rule`) and architecture fitness
functions (`fitness-functions`) govern layer direction and CI enforcement; an
`architecture-decision-records` workflow records significant decisions.

## Portability

- Markdown + frontmatter + `skills.yaml` are the source of truth; the Go installer is a thin,
  std-lib + `yaml.v3` renderer that embeds them.
- Named tools (`remy`, `ent`, `sqlc`, `atlas`, `goverter`, `Fuego`, `slog`, `mockgen`, `weak`,
  `text/template`) appear as canonical reference implementations. Helper package names in code
  examples are role-based and tool-agnostic (`tx`, `backoff`, `tracing`, `window`, `tplate`,
  `rules`, `permits`, `money`, `dberr`, `mixins`, `configload`, `dbtest`, `apptest`, ...) so the
  recipes port to any project.
- No `.proto`, no generated stubs, no business logic.
- Nothing derived is committed: `generated/`, `build/`, and `*.zip` are gitignored. The catalogs are
  printed by `list`, not stored as files.
- If Claude Code disappears, `knowledge/standards/` remains a usable engineering handbook; the skills
  are just one rendered view of it.

There is one known deviation: `knowledge/` is a top-level exported Go package, outside the allowed
set in `code-placement` (`cmd/`, `internal/`, `pkg/`, `build/` for a single module). It is not
compliant with that rule, and the files that import it are not a reason to call it compliant. It
stays top-level for two reasons: `knowledge/embed.go` does `//go:embed standards templates`, and
`go:embed` can only reach files at or below the package that declares the directive, so the embed
forces `knowledge/` to sit above `standards/`; and burying `standards/` under `pkg/` would defeat
the portability guarantee above: it must stay usable as an engineering handbook on its own,
independent of `pkg/`'s Go-API framing. Revisit this placement if a second module or binary ever
appears in this repo.
