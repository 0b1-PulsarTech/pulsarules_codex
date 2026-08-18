# Architecture

This document has two halves. The first describes the knowledge model and skill-rendering
pipeline: how a rule becomes a `SKILL.md`. The second describes the governance pipeline: how the
built binary analyzes a project's own source tree and commits against that same knowledge, both as
a git hook and as a standalone `governance` command.

## Knowledge model

Every reusable item is one of four kinds. The first three are authored as markdown with YAML
frontmatter; the fourth is generated.

| Kind     | Location                                | Role                                                                  |
|----------|-----------------------------------------|-----------------------------------------------------------------------|
| Rule     | `knowledge/standards/rules/<id>.md`     | Mandatory, enforceable principle (the WHAT floor)                     |
| Pattern  | `knowledge/standards/patterns/<id>.md`  | Implementation recipe with code (the HOW)                             |
| Workflow | `knowledge/standards/workflows/<id>.md` | Ordered sequence for a task (the ORDER)                               |
| Sidecar  | `knowledge/standards/skills/<id>.md`    | Authored orientation per skill (what it governs, when to reach for it) |
| Skill    | `generated/skills/<id>/SKILL.md`        | Generated capability that composes rules + patterns + workflows       |
| Profile  | `knowledge/standards/profiles.yaml`     | Install-time customization that overrides a skill's composition       |
| Router   | `knowledge/standards/router.yaml`       | The router's baseline, dispatch table, and order as filterable data   |

A **skill** is never authored directly. `knowledge/standards/skills.yaml` declares the 36 skills,
each naming the rules, patterns, and workflows it composes (`compose_rules`, `compose_patterns`,
`compose_workflows`), plus triggers, load policy (`always_load`), composition `order`, and
skill-level dependencies (`compose_skills`). The installer renders a skill by transcluding the
markdown body of its **sidecar** (`knowledge/standards/skills/<id>.md`) as the skill head -
orientation only, saying what the skill governs and when to reach for it - then **transcluding**
the markdown bodies of its composed rules,
patterns, and workflows under `###` sub-headings as the reference detail, wrapped in a frontmatter +
activation + composition shell. A sidecar is required for every non-router skill (validate enforces
it); `project-router` renders from its own template and carries no sidecar.

Edit a rule's or sidecar's markdown, rebuild, and every skill that composes it updates - skills never
drift from the knowledge.

**Customization profiles.** `knowledge/standards/profiles.yaml` declares install-time profiles, each
on an axis (e.g. `code-layout`) with per-skill `compose_rules`/`compose_patterns` overrides. Selecting
a profile (`install --layout <id>`) applies the override to a freshly-loaded index via
`Index.ApplyProfiles` before rendering, so one knowledge base serves several project shapes
(`monorepo` vs `inner-modules`) with no skill duplication. `validate` checks every override resolves
to an existing skill and to rules/patterns with non-empty bodies, so a selected variant is guaranteed
to render rather than be silently dropped.

**Installed-aware router.** `knowledge/standards/router.yaml` holds the project-router's baseline,
dispatch table, and composition order as data. When `install --skills <subset>` selects only some
skills, `Selection.Resolve` adds the mandatory baseline (`project-router` plus every `always_load`
skill) and, transitively, every skill a selected skill's `compose_skills` names (cycle-safe; `install`
prints what got pulled in and by which skill); the renderer then filters the router to that resolved
set - dropping baseline entries, dispatch rows, and order steps that name no installed skill - so the
orchestrator never advertises a skill that was not installed. `install --all` or `--router-only` passes
no filter and renders the complete router.

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
frontmatter IS the metadata - there is no separate `manifests/` tree. This repo is the source of
truth; application repos copy from it, not the other way.

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

This is one of the two things `pulsarules_cli` does. The other, described next, analyzes a target
repository (Go source, commit messages, git history) instead of rendering the knowledge base.

## Governance pipeline

### Session and scopes

`internal/analysis.Session` (`analysis.go`) is the single orchestrator: discovery, context loading, and
analysis. `NewSession(repo, commitMsg, index, cfg)` builds a session over a `vcs.Repository` (nilable
when `vcs.Open` returned `vcs.ErrNoRepository` - the session then runs with no git history and no
changed-file discovery instead of failing), an optional commit message, the knowledge `*Index` (for
rule-body injection), and a `*config.GovernanceConfig` (defaulted and preset-applied when nil).

`Session.Analyze(scope, status)` runs three steps: `Discover` gathers changed files, git history, and
staged renames with no AST parsing and returns a `Discovery`; `Load` turns that `Discovery` into a full
`core.AnalysisContext`, pre-parsing changed Go files into an `astcache.Cache`; then a `StageRunner` is
built, registered for `scope`, and run. Separating `Discover` from `Load` lets a caller inspect or edit
the discovered state first; `status` is the already-computed `vcs.Status`, so a caller that read it
already (a hook) does not pay for a second read.

Three scopes (`scope.go`) control which analyzers run and what context is built:

| Scope          | Runs                                                                    | Used by                                  |
|----------------|-------------------------------------------------------------------------|------------------------------------------|
| `ScopeFull`    | every registered analyzer, full project context                         | `governance` command (default)           |
| `ScopeCommit`  | static + AST + commit analyzers, for pre-commit hooks                   | `commitlint` (the git `commit-msg` hook) |
| `ScopeChanged` | static + AST + arch analyzers over changed files, skipping delegation   | the Claude Code `Stop` hook              |

`ScopeCommit` still gathers staged renames and per-file staged status (`commit-move-purity` needs
exactly that), even though it skips the wider `SourceProvider`/AST-cache work that `ScopeFull` and
`ScopeChanged` do.

### Stages and the analyzerSpecs table

`StageRunner` (`pipeline.go`) holds analyzers grouped by `core.StageID` and runs them in stage order
(`StageContext` through `StageOutput`), skipping an analyzer that is disabled in `AnalysisConfig` or
whose `Needs()` (`NeedsAST`, `NeedsGitHistory`) are not met by the current context. `boot.go`'s
`analyzerSpecs` is the single ordered list of every analyzer the binary registers, each entry pairing an
`analyzerBuilder` with the scopes it runs under (`staticScopes` = Full+Commit+Changed for static/AST/
commit/rule-injection/output work that stays useful with only a commit message or only changed files;
`archScopes` = Full+Changed for the architecture analyzers, which need a source set; `delegationScopes`
= Full only, since only a full run pays the cost of spawning golangci-lint or gopls). Adding an analyzer
is one entry in this table. `registerForScope` builds the `core.LanguageRegistry` (Go registered via
`internal/analyzer/golang`) once and registers only the specs whose scopes contain the requested
scope.

### internal/analyzer/** families

Every analyzer implements the small `core.Analyzer` interface (`ID`, `Name`, `Description`, `Stage`,
`Category`, `Needs`, `Analyze`) declared in `internal/analyzer/core`, which also carries the shared
`AnalysisContext`, `Finding`, `Requirements`, and the `Language`/`LanguageRegistry` Strategy for
file-extension-based dispatch. The families:

- `static/*` - pure text/structure checks needing no parser: `filesize` (line-count ceiling),
  `textmarkers` (invisible carriers, exotic spaces and AI typographic punctuation), `topoffile` and
  `bigcomment` (comment placement and length, both via the `LanguageRegistry`).
- `ast/*` - `go/ast`-based checks: `complexity` (cyclomatic complexity, function size, parameter count,
  flag arguments, magic numbers), `controlflow` (redundant else, deep nesting), `shadowing` (variable/
  builtin shadowing via a scope stack), `imports` (three-group import order), `naming` (identifier
  conventions).
- `arch` - multi-file architecture checks: `PackageBoundaryAnalyzer` (inner-layer packages must not
  import outer-layer ones) and `ImportCycleAnalyzer` (package import-cycle detection).
- `commit` - commit-message validation (`CommitAnalyzer`): format, emoji catalog and repetition
  window, tool-attribution trailers, delegating emoji lookups to `internal/emoji`.
- `movepurity` - flags a staged rename below the configured git rename-similarity score, or one
  staged alongside unrelated edits, as not a pure move.
- `delegation/{golangcilint,gopls}` - shell out to an external tool and translate its output into
  `core.Finding`s; both are gated to `delegationScopes` (`ScopeFull`) since spawning a process is only
  worth the cost on a full run.
- `core/astcache` - a per-invocation cache of parsed Go ASTs, populated once in `Session.Load` so
  analyzers call `Get(path)` instead of each re-parsing the same file.
- `golang` - the one `Language` handler registered today, implementing comment/blank/package-
  declaration detection for `.go` files.

### internal/vcs

`vcs.Repository` is the typed, read-only git port the whole governance pipeline depends on (`Root`,
`HeadSubject`, `HeadAuthorEpoch`, `RecentSubjects`, `WorktreeStatus`, `StagedRenames`). `vcs.Open`
walks up from a directory to the repository root (resolving a linked worktree) and returns
`ErrNoRepository` rather than a bare error when the directory is not a git repository, so callers can
degrade quietly. EVERY git read in the product goes through `gitcmd.go`'s one `runGit` helper, which
runs `git -C <dir> <args>` under a 2-second timeout with `LC_ALL=C` pinned (so `isEmptyRepoError`'s
English-message match survives a localized git install). The one deliberate exception is
`tools/emojigen`, documented in its own source: it shells out directly because it scans an entire
reference repository's history (`git log --all --max-count=1000000`), which a per-object walk through
this package would run orders of magnitude slower, and `tools/` is not a runtime dependency the vcs
port needs to cover.

### internal/hook

`Dispatcher.Dispatch(mode, payload)` is the one decode point for a Claude Code hook payload
(`decodeHookPayload`, which degrades a malformed payload to its zero value rather than failing, since a
hook must never block the agent's turn) and routes it to a per-mode handler (`session-start`,
`pre-edit`, `post-edit`, `pre-search`, `user-prompt`, `stop`, `subagent-start`, `session-end`;
plus a deliberately silent `subagent-stop` nothing dispatches today). It always
returns nil; failures surface only through the optional `*slog.Logger`. `Router.SkillsForFile` resolves
which skills a `post-edit`/`pre-search` event should announce from `knowledge.Index`'s own
`skills.yaml`/`router.yaml` data (the always-load baseline for `.go` files, any skill whose trigger
names the file's extension, the "Any test work" dispatch row for `_test.go` files) instead of a
hardcoded per-extension map, so a new skill trigger routes with no Go change. `SessionTracker`
(`session.go`) writes per-session marker files under the OS temp dir, keyed by event and session ID, so
`OncePerSession` and content-hashed `FirstEmission` deduplicate repeat emissions within one session;
`Cleanup` scans and removes them at `session-end`. `checklist.go`'s `TypedChecklist` and `checks.go`'s
`RunGovernanceCheck` are what the `stop` handler calls into the governance pipeline through: the latter
opens a `Session` at `analysis.ScopeChanged` and formats findings with `analysis.StyleHook`.

### internal/config

`GovernanceConfig` is the top-level runtime configuration: per-analyzer enable/params
(`AnalyzerConfig`), emoji rules (`EmojiConfig`), and move-purity thresholds
(`MovePurityConfig`). `Defaults()` returns the embedded baseline; `presets.go`
applies a named preset (strict/recommended/minimal) in-memory. `internal/analysis/runner.go`'s
`toAnalysisConfig` converts a `GovernanceConfig` into the `core.AnalysisConfig` the `StageRunner`'s
`enabled()` check reads, projecting `EmojiConfig`/`MovePurityConfig` onto the `commit-lint` and
`commit-move-purity` analyzers' own `Params` maps.

### internal/obs

Hook telemetry is OFF by default: `obs.New` with an empty `Config.Level` returns a
`slog.DiscardHandler` logger and a no-op `Closer`, so nothing is created, opened, or read - `--log-level`
or `PULSARULES_LOG_LEVEL` must set a level (`debug`/`info`/`warn`/`error`) before any file touches disk.
When enabled, the log is size-capped by keeping its tail: `truncateToTail` rewrites the file to its last
`MaxBytes/2` (default 256 KiB total) worth of content, cut at the first line boundary, before opening it
for append, so a long-lived hook log never grows unbounded.

### internal/emoji

The commit emoji vocabulary is data, not code: `catalog.txt`, `types.txt`, and `themes.txt` under
`internal/emoji/data` back `Catalog` (valid/prohibited/non-rendering shortcodes), the commit-type
suggestion ranking, and `themes.go`'s keyword-to-emoji-family table. The commits skill's emoji-selection
guidance renders its area anchors from `themes.txt` via `emoji.Anchors()` and the `emojiAnchors`
template func (`internal/skill/render/namespace.go`), so that prose cannot drift from the data the
`commit` analyzer actually matches against.

### internal/gitignore and internal/selfbin

Two small leaf helpers, each factored out because two or three install paths needed the identical body:
`gitignore.Ensure(dir, entries...)` appends missing lines to `<dir>/.gitignore` (used by the Claude and
opencode installers, plus `mcpwire` for `.mcp.json`); `selfbin.Copy(dst)` copies the currently running executable to `dst` at `0o755`,
so `pulsarules_cli` installs itself as the hook/plugin binary a project calls back into. `hookwire`,
`githook.InstallBinary`, and `opencodehook` all call it rather than each shelling out to copy the
binary themselves.

## Packages

| Package                          | Responsibility                                                                                                                |
|----------------------------------|-------------------------------------------------------------------------------------------------------------------------------|
| `knowledge`                      | `embed.go` (embed), `model.go` (Rule/Pattern/Workflow/Skill), `frontmatter.go` (fence parser), `index.go` (Load + Index)      |
| `internal/skill/render`          | `render.go` (Renderer + transclusion), `compose.go`, `text.go`                                                                |
| `internal/skill/validate`        | `validate.go` (pipeline), `checks.go` (reference + body integrity checks)                                                     |
| `internal/skill/output`          | `install.go` (Selection + Install), `generate.go`, `package.go`, `write.go`                                                   |
| `internal/skill/hookwire`        | `hooks.go` (install hook + gitignore), `settings.go` (idempotent settings.json/.local merge, `$CLAUDE_PROJECT_DIR`)           |
| `internal/skill/mcpwire`         | `mcp.go` (merge gopls into `.mcp.json`), `gopls.go` (PATH check + generate gopls-navigation from `gopls mcp -instructions`)   |
| `internal/skill/opencodewire`    | `config.go` (merge `opencode.json`: instructions + gopls mcp, SkillsSubdir), `unwire.go`                                      |
| `internal/skill/agentswire`      | `agents.go` (render the root `AGENTS.md`, scoped to the selected skills), `uninstall.go` (marker-gated removal)               |
| `internal/skill/cursorwire`      | `rules.go` (marker-gated write of one `.mdc` per rule under `.cursor/rules`), `remove.go`                                     |
| `internal/skill/target`          | `target.go` (`Target`/`Context`/`Report`), `registry.go` (`Registry`), `claude.go`, `opencode.go`, `agents.go`, `cursor.go` - one Strategy per install layout |
| `cmd/pulsarules_cli`             | one file per command: main, generate, install (dispatches to `internal/skill/target`), list, validate, package, version, hook, commitlint, governance |

`knowledge` is the asset + embed anchor; the `internal/skill/*` packages are cohesive operation
units (each source file carries a name-matched `_test.go`); `cmd` is the CLI. The old `manifests/`
tree and the prior `internal/{model,manifest,render,generate,install,pkgzip,validate,catalog}`
packages are gone. Installing skills, the hook, and the settings wiring is one `install` command -
hook assets are embedded under `knowledge/templates/hooks/`.

The governance pipeline (see above) adds its own package set:

| Package                          | Responsibility                                                                                      |
|----------------------------------|-----------------------------------------------------------------------------------------------------|
| `internal/analysis`               | `session.go` (`Session`: Discover/Load/Analyze), `scope.go` (the 3 scopes), `pipeline.go` (`StageRunner`), `boot.go` (`analyzerSpecs`), `runner.go` (`toAnalysisConfig`), `rule_injection.go`, `output.go`, `format.go`, `history.go` |
| `internal/analyzer/core`          | `Analyzer`/`Language` interfaces, `AnalysisContext`, `Finding`, `Requirements`, `LanguageRegistry`, `core/astcache` (parsed-AST cache) |
| `internal/analyzer/static/*`      | `filesize`, `textmarkers`, `topoffile`, `bigcomment` - text/structure checks                        |
| `internal/analyzer/ast/*`         | `complexity`, `controlflow`, `shadowing`, `imports`, `naming` - `go/ast`-based checks                |
| `internal/analyzer/arch`          | `PackageBoundaryAnalyzer`, `ImportCycleAnalyzer` - multi-file architecture checks                   |
| `internal/analyzer/commit`        | `CommitAnalyzer` - commit-message format, emoji, and trailer rules                                  |
| `internal/analyzer/movepurity`    | flags a staged rename that is not a pure move                                                       |
| `internal/analyzer/delegation/*`  | `golangcilint`, `gopls` - shell out to an external tool, gated to `ScopeFull`                       |
| `internal/analyzer/golang`        | the `Language` handler for `.go` files                                                              |
| `internal/vcs`                    | `Repository` port, `Open`, `Status`/`Rename` types, `gitcmd.go`'s `runGit` (the one git-exec point)  |
| `internal/hook`                   | `dispatcher.go` (`Dispatcher`), `router.go` (`Router`), `session.go` (`SessionTracker`), `checklist.go`, `checks.go` (`RunGovernanceCheck`), `install/*` (per-target hook installers) |
| `internal/config`                 | `GovernanceConfig` and its sub-configs, `presets.go`                                                 |
| `internal/obs`                    | `New` (off-by-default `slog.Logger`), tail-truncating log rotation                                   |
| `internal/emoji`                  | `Catalog`, `Anchors`/`themes.go`, `data/{catalog,types,themes}.txt`                                  |
| `internal/gitignore`              | `Ensure` - idempotent `.gitignore` entries                                                           |
| `internal/selfbin`                | `Copy` - copies the running executable to install it as a hook/plugin binary                         |
| `internal/evals`                  | `model.go`/`load.go` (embedded eval scenarios, `data/<skill>.json`), `validate.go` (`ValidateCheck`), `grade.go` (`Grade`) |

## The router skill

`project-router` is mandatory and always-loaded first. Its dispatch table (in
`knowledge/templates/skills/router.md.tmpl`) maps a task signal to the skills to load, in
composition order. It is **generalized** - product-domain rows were removed; only engineering rows
remain (new app, bootstrap/DI, use case, REST/gRPC endpoint, entity/migration, sqlc query,
multi-write tx, outbox event, worker, permission, external provider, template/rule/proposal engine,
test, module boundary, refactoring, architecture review). The vertical Dependency Rule
(`dependency-rule`) and architecture fitness functions (`fitness-functions`) govern layer direction
and CI enforcement; an `architecture-decision-records` workflow records significant decisions.

## Skill-effectiveness evals

Rendering a skill and having an analyzer catch violations both prove the pipeline is stable; neither
proves a skill's *text* changes what an agent produces. `internal/evals` measures that, for a starting
set of four skills (`code-minimalism`, `integration-tests`, `commits`, and the `safety` rule composed
into `go-style`), following the eval-scenario method `samber/cc-skills-golang` documents: per skill,
a handful of scenarios, each a realistic `prompt`, the specific `trap` the skill exists to prevent, and
plain-English `assertions` a produced artifact is graded against.

**Format and placement.** Scenarios live under `internal/evals/data/<skill>.json`, embedded via
`//go:embed` like `internal/emoji/data/*.txt` - not under `knowledge/standards/`, because they are not
rendered into a `SKILL.md`; they only drive this package's own validation and grading, so they stay
colocated with the code that reads them. Every assertion in `model.go`'s `Assertion` carries a `Kind`:
`"machine"` assertions carry a `Check` (`contains`/`not_contains`/`regex_match`/`regex_absent`) `Grade`
runs directly against an artifact's text; `"judge"` assertions carry none and are left for a human or
LLM reader.

**Validation.** `evals.ValidateCheck` matches `validate.Check`'s `func(*knowledge.Index) []string`
shape and is injected via the same `extra` seam `analysis.RuleAnalyzersCheck` uses
(`internal/cli/validate.go`): every scenario's `skill` must resolve against `skills.yaml`
(`idx.SkillExists`), every scenario must declare a non-empty `trap` and at least one assertion, and
every assertion's `Kind` must be `machine` (with a `Check`) or `judge` (without one).

**Grading.** `evals.Grade(scenario, artifact)` scores every `machine` assertion's `Check` against the
artifact string and reports `pass`/`fail`; `judge` assertions report `needs_judge` untouched -
`Grade` has no model to read the artifact with.

**What this harness does NOT do.** It does not invoke a model. Producing the with-skill and
without-skill artifacts a scenario is graded against is an operator procedure, not something this
binary runs: (1) start two sessions on the same scenario's `prompt`, one with the target skill loaded
(or its knowledge composed into context) and one without any of this repo's guidance; (2) save each
session's produced code/transcript as a plain-text artifact; (3) run
`pulsarules_cli evals --artifact <file>` on both and diff the `machine` tallies (the command is the
one call site of `evals.Grade`, so this step is executable rather than a Go snippet the reader has to
write); (4) read every `needs_judge`
assertion against both artifacts and score it by hand (or via a separate LLM-judge call this repo does
not provide); (5) aggregate pass/fail across the scenarios run so far into a with-score, a without-score,
and a delta - and flag the same two anomalies `cc-skills-golang` does: low delta with a high
without-score (the skill may not be earning its bytes) and a low with-score (the skill's own text needs
rewriting). A harness that faked step (1)-(2) would be exactly the kind of guard that looks like it
proves something and does not; this one stops at the boundary it can actually verify.

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

**Known deviation.** `knowledge/` is a top-level exported Go package, outside the allowed set in
`code-placement` (`cmd/`, `internal/`, `pkg/`, `build/` for a single module). It is not compliant
with that rule, and the ~48 files that import it are not a reason to call it compliant. It stays
top-level for two reasons: `knowledge/embed.go` does `//go:embed standards templates`, and `go:embed`
can only reach files at or below the package that declares the directive, so the embed forces
`knowledge/` to sit above `standards/`; and burying `standards/` under `pkg/` would defeat the
portability guarantee above - it must stay usable as an engineering handbook on its own, independent
of `pkg/`'s Go-API framing. Revisit this placement if a second module or binary ever appears in this
repo.
