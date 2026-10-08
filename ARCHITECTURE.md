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
| Rule     | `knowledge/standards/rules/<id>.md`     | Mandatory, enforceable principle (what must hold)                     |
| Pattern  | `knowledge/standards/patterns/<id>.md`  | Implementation recipe with code (how to build it)                     |
| Workflow | `knowledge/standards/workflows/<id>.md` | Ordered sequence for a task (in what order)                           |
| Sidecar  | `knowledge/standards/skills/<id>.md`    | Authored orientation per skill (what it governs, when to reach for it) |
| Skill    | `generated/skills/<id>/SKILL.md`        | Generated capability that composes rules + patterns + workflows       |
| Profile  | `knowledge/standards/profiles.yaml`     | Install-time customization that overrides a skill's composition       |
| Router   | `knowledge/standards/router.yaml`       | The router's baseline, dispatch table, and order as filterable data   |

A **skill** is never authored directly. `knowledge/standards/skills.yaml` declares the 45 skills,
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

This is one of the two things `pulsarules_cli` does. The other, described next, analyzes a target
repository (Go source, commit messages, git history) instead of rendering the knowledge base.

## Governance pipeline

### Session and scopes

`internal/analysis.Session` (`analysis.go`) is the single orchestrator: discovery, context loading, and
analysis. `NewSession(repo, commitMsg, index, cfg)` builds a session over a `vcs.Repository` (nilable
when `vcs.Open` returned `vcs.ErrNoRepository` - the session then runs with no git history and no
changed-file discovery instead of failing), an optional commit message, the knowledge `*Index` (for
rule-body injection), and a `*config.GovernanceConfig` (defaulted and preset-applied when nil).

`Session.Analyze(scope, status, files)` runs three steps: `Discover` gathers changed files, git
history, and staged renames with no AST parsing and returns a `Discovery`; `Load` turns that
`Discovery` into a full `core.AnalysisContext`, pre-parsing changed Go files into an
`astcache.Cache`; then a `StageRunner` is built, registered for `scope`, and run. Separating
`Discover` from `Load` lets a caller inspect or edit the discovered state first; `status` is the
already-computed `vcs.Status`, so a caller that read it already (a hook) does not pay for a second
read. `files` is a `FileSet`: `FileSetChanged` analyzes the changed files, `FileSetAll` (the
`--all-files` flag) walks every source file in the tree.

The file walk is git-listed: `Session.attachSources` asks the repository for `git ls-files -z --cached
--others --exclude-standard` (`vcs.ListFiles`) and wraps the provider in `sources.NewListedProvider`, so
ignored build output is never read. If the repository cannot list files it logs a warning and falls
back to a plain filesystem walk with `sources.DefaultSkipDirs`. A `.sql` file in a directory holding an
`atlas.sum` (an Atlas migration directory) is marked generated, as is any file whose own header says so
(`lexical.IsGeneratedSource`); generated-file findings are split out of the result unless
`GovernanceConfig.IncludeGenerated` is set.

Three scopes (`scope.go`) control which analyzers run and what context is built:

| Scope          | Runs                                                                | Used by                                   |
|----------------|---------------------------------------------------------------------|-------------------------------------------|
| `ScopeFull`    | every registered analyzer, full project context                     | `governance` command (default)            |
| `ScopeCommit`  | static-gate analyzers over files with a staged change               | `commitlint`; `governance --scope commit` |
| `ScopeChanged` | static + arch gates over changed files, skipping the full-only gate | the Claude Code `Stop` hook               |

`ScopeCommit` drops every change with nothing staged (`Session.changedFiles` keeps only
`Change.Staged`), so a file edited only in the worktree is not analyzed. The file analyzers (all but
`commit-move-purity`, which reads the staged and HEAD blobs) still read each kept file's
working-tree bytes, so a partially staged file is judged with its unstaged hunks included. It still
gathers staged renames (`commit-move-purity` needs exactly that), and it attaches the file provider
only when a staged change exists (with the default changed-file set), since it never runs the
whole-tree analyzers.

### Stages and the analyzer registry

`StageRunner` (`pipeline.go`) holds analyzers grouped by `core.StageID` and runs them in stage order
(`StageContext` through `StageOutput`), skipping an analyzer that is disabled in `AnalysisConfig`.
`internal/analysis/registry` owns the single ordered table of every analyzer the binary registers
(`specs()`, concatenated from `coreSpecs`, `commitSpecs`, `astSpecs`, `styleSpecs`, `pipelineSpecs`
in run order). Each entry declares its id, a builder, and a `registry.Gate`: `GateStatic` (every scope;
static/AST/commit/rule-injection/output work that stays useful with only a commit message or only changed
files), `GateArch` (architecture analyzers, which need a source set) or `GateFullOnly` (golangci-lint and
branch-name, which only a full run pays for). `analysis/boot.go`'s `gatesFor(scope)` maps a scope to its
gates (`ScopeCommit` = static; `ScopeChanged` = static + arch; `ScopeFull` = all three), and
`registerForScope` builds the `core.LanguageRegistry` (via `langspec.RegisterAll`, the one data-driven
registration for every language) once and registers `registry.Analyzers(..., gates)`. Adding an analyzer
is one entry in the table.

### internal/analyzer/** families

Every analyzer implements the small `core.Analyzer` interface (`ID`, `Stage`, `Analyze`) declared
in `internal/analyzer/core`, which also carries the shared `AnalysisContext`, `Finding`, and the
`Language`/`LanguageRegistry` Strategy for file-extension-based dispatch. `Reporter.Resolved(ctx)`
re-reads a reporter's severity from the run's config, and `core.EachChangedFile` iterates the
changed files as text under the run's skip set, while `core.RunPerGoFile` iterates the parsed Go
files. `core.InPlaceAnalyzer` marks the second contract - an analyzer that rewrites `ctx.Findings`
(rule injection, output) instead of returning new ones. The families:

- `static/*` - text/structure checks (`topoffile`, `bigcomment`, `historynarration` and
  `simplificationpath` read each language's comment syntax through the `LanguageRegistry`; for `.go`
  files `historynarration` reads comments from a Go AST, parsing the file itself when the AST cache
  lacks it, and falls back to the comment scanner only on a parse error): `filesize` (line-count
  ceiling), `textmarkers` (invisible carriers, exotic spaces and AI typographic punctuation),
  `topoffile` and `bigcomment` (comment placement and length, so they cover non-Go files too),
  `historynarration` (comments that narrate history instead of stating the invariant) and
  `simplificationpath` (a simplification comment must name its upgrade path).
- `ast/*` - `go/ast`-based checks: `complexity` (cyclomatic complexity, function size, parameter count,
  flag arguments, magic numbers), `controlflow` (redundant else, deep nesting), `shadowing`, `imports`
  (three-group import order), `naming`, `namedreturns`, `singleusedecl`, `timediscipline`,
  `contextdiscipline`, `pproflabels`, and the style checks `doccomment` (`doc-comment`),
  `bodyheadercomment` (`body-header-comment`), `testcomment` (`test-comment`),
  `singleletter` (`single-letter-name`), `variadicindex` (`variadic-index`),
  `toomanyresults` (`too-many-results`), `ungroupedinterfaces`,
  `packagelevelvar` (`package-level-var`) and `packagefilecount` (`package-file-count`). The
  `recommended` preset leaves all of them enabled (warning level, except `import-groups` and
  `pprof-labels`, which are errors, and `complexity`'s magic-number findings, which are info);
  `minimal` disables `complexity`, `control-flow`, `shadowing`, `named-returns`,
  `context-discipline`, `single-use-declaration`, `test-comment` and every style check, keeping
  `import-groups`, `naming`, `time-discipline` and `pprof-labels`, plus `single-letter-name` for
  upper-case single letters on functions, methods, types, constants, variables and fields (locals
  included). `ast/directive` recognizes directive comments (`//go:`, `//nolint`, `//pulsarules:`,
  ...) so comment checks skip them, and `ast/gotest` holds shared test-function helpers.
- `arch` - multi-file architecture checks: `PackageBoundaryAnalyzer` (inner-layer packages must not
  import outer-layer ones) and `ImportCycleAnalyzer` (package import-cycle detection).
- `commit` - commit-message validation (`commit.Analyzer`): format, emoji catalog and repetition
  window, tool-attribution trailers, delegating emoji lookups to `internal/emoji`.
- `movepurity` - flags a staged rename that carries an edit beyond its package clause and import
  fixups (reported as below `min_similarity` when the score is low), or one staged alongside
  unrelated edits (`movepurity/gosrc` reads Go declarations).
- `branchname` - flags a checked-out branch whose name carries no recognized type prefix
  (Conventional Commit types, the gitflow lines, plus a project's own `extra_types`). Gated to
  `GateFullOnly` - the pre-push run - since a branch name is not a property of a staged change.
- `delegation/golangcilint` - shells out to an external tool and translates its output into
  `core.Finding`s; gated to `GateFullOnly` since spawning a process is only worth the cost on a full run.
- `ruleinjection` and `output` - the in-place stages that attach each finding's rule ID and rule
  summary, then sort findings by severity, file and line and drop adjacent duplicates.
- `core/astcache` - a per-invocation cache of parsed Go ASTs, populated once in `Session.Load` so
  analyzers call `Get(path)` instead of each re-parsing the same file.
- `core/lexical` - comment and header lexing shared by the comment analyzers: `CommentSyntax`, the
  directive grammar, `IsGeneratedSource` and header-line extraction.
- `core/sources` - the file `Provider`, the skip-directory sets, and the git-listed provider wrapper.
- `langspec` - the single data-driven `Language` handler and the spec table behind it (go, clike, hash,
  sql, nix, lua, markup, sfc). The `clike` spec covers the `//` and `/* */` family (`.dart`, `.ts`,
  `.js`, `.java`, `.rs`, `.c`, ...). Only the `go` spec (`package` clause) and the `sql` spec (`CREATE
  SCHEMA`) carry a package-declaration predicate.

### internal/vcs

`vcs.Repository` is the typed, read-only git port the whole governance pipeline depends on
(`Root`, `HeadAuthorEpoch`, `RecentSubjects`, `CurrentBranch`, `WorktreeStatus`, `StagedRenames`,
plus the `BlobReader` and `TreeReader` embeds for HEAD and index content). The concrete repository
also has `ListFiles`, which `analysis.Session` reaches through its own small `fileLister`
interface. `WorktreeStatus` runs `git --no-optional-locks status --porcelain -z
--untracked-files=all`, so a background run never rewrites the user's index and a path with
unusual bytes is not quoted. The package-level `vcs.CommonDir` resolves the git dir every linked
worktree of a repository shares - the git-hook installer joins it with `hooks` to find the dir git
actually runs hook scripts from. `vcs.Open` walks up from a directory to the repository root
(resolving a linked worktree) and returns `ErrNoRepository` rather than a bare error when the
directory is not a git repository, so callers can degrade quietly. Every git read in the product
goes through `gitcmd.go`'s `runGit` helper (and its `runGitWithin` variant, which `WorktreeStatus`
and `ListFiles` use for a 30-second budget on large trees), which runs `git -C <dir> <args>` under a
2-second default timeout with `LC_ALL=C` pinned (so `isEmptyRepoError`'s English-message match
survives a localized git install). The one deliberate exception is `tools/emojigen`, documented in
its own source: it shells out directly because it scans an entire reference repository's history
(`git log --all --max-count=1000000`), which a per-object walk through this package would run orders
of magnitude slower, and `tools/` is not a runtime dependency the vcs port needs to cover.

### internal/hook

`Dispatcher.Dispatch(mode, payload)` (`hook.go`) is the one decode point for a Claude Code hook
payload (`decodeHookPayload`, which degrades a malformed payload to its zero value rather than
failing, since a hook must never block the agent's turn) and routes it to a per-mode handler
(`session-start`, `pre-edit`, `post-edit`, `pre-search`, `user-prompt`, `stop`, `subagent-start`,
`session-end`). It always returns nil; failures surface only through the optional `*slog.Logger`.
`Router.SkillsForFile` resolves which skills a `post-edit` event should announce from
`knowledge.Index`'s own `skills.yaml`/`router.yaml` data (the always-load baseline entries that fit
the edited file's language, any skill whose trigger names its extension, the test dispatch rows
fitting Go for `_test.go` files) instead of a hardcoded per-extension map, so a new skill trigger
routes with no Go change. `SessionTracker` (`session.go`) writes per-session marker files under the
OS temp dir, keyed by event and session ID (an ID that is not a plain token is hashed first), so
`OncePerSession` and content-hashed `FirstEmission` deduplicate repeat emissions within one session;
`Cleanup` scans and removes them at `session-end`. The `checklist` subpackage's `Typed` builds the
per-extension post-edit checklist bullets from the worktree status, and `checks.go`'s
`RunGovernanceCheck` is what the `stop` handler calls into the governance pipeline through: it opens
a `Session`, runs it over the changed files at the scope the caller passes (the `Stop` hook passes
`analysis.ScopeChanged`), and formats findings with `analysis.StyleHook`. `Router.Routes` gates
`pre-edit` and `post-edit`: a `.go` file, or one a skill trigger names by extension; the baseline
alone never fires them for a Dart or TypeScript file. The post-edit verify line names Go commands
only for a `.go` file. `drift.go` runs once per session at `session-start`: it compares
`knowledge.Fingerprint()` (the digest of the standards the binary embeds, see
`knowledge/fingerprint.go`) against the project's `knowledge/` tree on disk and warns when the
binary trails the source it was built from. This is a guard for the development repo itself - a
consumer project has no `knowledge/` tree and stays silent.

### internal/config

`GovernanceConfig` is the top-level runtime configuration: per-analyzer enable/params
(`AnalyzerConfig`), emoji rules (`EmojiConfig`), move-purity thresholds (`MovePurityConfig`),
`IncludeGenerated`, and extra `SkipDirs`. `Defaults()` returns the embedded baseline; `presets.go`
applies a named preset (strict/recommended/minimal) in-memory. `internal/analysis/runner.go`'s
`toAnalysisConfig` converts a `GovernanceConfig` into the `core.AnalysisConfig` the `StageRunner`'s
`enabled()` check reads, projecting `EmojiConfig`/`MovePurityConfig` onto the `commit-lint` and
`commit-move-purity` analyzers' own `Params` maps.

### internal/obs

Hook telemetry is off by default: `obs.New` with an empty `Config.Level` returns a
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
| `knowledge`                      | `embed.go` (embed), `model.go` (Rule/Pattern/Workflow/Skill), `frontmatter.go` (fence parser), `load.go` + `read*.go` (loading), `index.go` (Index), `fingerprint.go` |
| `internal/skill/render`          | `render.go` (Renderer + transclusion), `compose.go`, `router*.go`, `sidecar.go`, `namespace.go` (template funcs)              |
| `internal/skill/validate`        | `validate.go` (`Validate` pipeline + `Check` seam), `checks*.go` (reference, body, profile, emoji, normative integrity checks) |
| `internal/skill/output`          | `install.go`, `selection.go` (Selection), `generate.go`, `package.go`, `remove.go`, `write.go`                                  |
| `internal/skill/hookwire`        | `hooks.go` (install hook), `settings.go` (idempotent settings.json/.local merge, `$CLAUDE_PROJECT_DIR`)             |
| `internal/skill/mcpwire`         | `mcp.go` (merge gopls into `.mcp.json`), `gopls.go` (PATH check + generate gopls-navigation from `gopls mcp -instructions`)   |
| `internal/skill/opencodewire`    | `config.go` (merge `opencode.json`: instructions + gopls mcp, SkillsSubdir), `unwire.go`                                      |
| `internal/skill/agentswire`      | `agents.go` (render the root `AGENTS.md`, scoped to the selected skills), `uninstall.go` (marker-gated removal)               |
| `internal/skill/cursorwire`      | `rules.go` (marker-gated write of one `.mdc` per rule under `.cursor/rules`), `remove.go`                                     |
| `internal/skill/target`          | `target.go` (`Target`/`Context`, `Report` aliasing `internal/report`), `registry.go` (`Registry`), `claude.go`, `opencode.go`, `agents.go`, `cursor.go` - one Strategy per install layout |
| `internal/langdetect`            | `detect.go` (`Detect`: depth-capped marker walk), `resolve.go` (`Resolve`: override > persisted config > detect), `config.go` (`.pulsarules-languages.json` load/save/remove), `language.go`, `set.go` |
| `cmd/pulsarules_cli`             | `main.go` only: parse args, run `bootstrap.DoInjections`, dispatch to `internal/cli`, the one place that calls `os.Exit` |
| `internal/cli`                   | one file per command (generate, list, validate, package, hook, commitlint, governance, clean, evals), `install/`, `uninstall/`, and `cliopts/` (flag parsing, `usage.txt`) |
| `internal/bootstrap`             | `DoInjections` - the composition root that registers knowledge, vcs, obs and the registries in the injector |

`knowledge` is the asset + embed anchor; the `internal/skill/*` packages are cohesive operation
units (most source files carry a name-matched `_test.go`); commands live in `internal/cli`, and
`cmd/pulsarules_cli` is `main.go` only. Installing skills, the hook, and the settings wiring is one
`install` command - hook assets are embedded under `knowledge/templates/hooks/`.

The governance pipeline (see above) adds its own package set:

| Package                          | Responsibility                                                                                      |
|----------------------------------|-----------------------------------------------------------------------------------------------------|
| `internal/analysis`               | `analysis.go` (`Session`: Discover/Load/Analyze), `scope.go` (the 3 scopes), `pipeline.go` (`StageRunner`), `boot.go` (`registerForScope`, `gatesFor`), `runner.go` (`toAnalysisConfig`), `convert.go`, `suppress.go` (generated-file split), `format.go`, `history.go`, `output.go` (`CountBySeverity`), `registry/` (the analyzer table) |
| `internal/analyzer/core`          | `Analyzer`/`InPlaceAnalyzer`/`Language` interfaces, `AnalysisContext`, `Finding`, `Reporter.Resolved`, `EachChangedFile`, `LanguageRegistry`; stdlib plus its own subpackages only: `core/astcache`, `core/lexical` (comment/header lexing), `core/sources` (file sources, skip sets, git-listed provider) |
| `internal/analyzer/static/*`      | `filesize`, `textmarkers`, `topoffile`, `bigcomment`, `historynarration`, `simplificationpath` - text/structure checks |
| `internal/analyzer/ast/*`         | `go/ast`-based checks: complexity, control flow, shadowing, imports, naming, named-returns, single-use-declaration, time-discipline, context-discipline, pprof-labels, and the doc-comment, body-header-comment, test-comment, single-letter-name, variadic-index, result-count, interface-grouping, package-var and package-file-count style checks |
| `internal/analyzer/arch`          | `PackageBoundaryAnalyzer`, `ImportCycleAnalyzer` - multi-file architecture checks                   |
| `internal/analyzer/commit`        | `Analyzer` - commit-message format, emoji, and trailer rules                                        |
| `internal/analyzer/movepurity`    | flags a staged rename that is not a pure move                                                       |
| `internal/analyzer/ruleinjection`, `internal/analyzer/output` | in-place stages: attach each finding's rule ID and rule summary; sort findings by severity, file and line and drop adjacent duplicates |
| `internal/analyzer/branchname`    | flags a branch name with no recognized type prefix; `GateFullOnly` (pre-push) only                  |
| `internal/analyzer/delegation/*`  | `golangcilint` - shells out to an external tool, gated to `GateFullOnly`                            |
| `internal/analyzer/langspec`      | one data-driven `Language` handler plus the spec table for every language (go, clike, hash, sql, nix, lua, markup, sfc) |
| `internal/vcs`                    | `Repository` port, `Open`, `CommonDir`, `Status`/`Rename` types, `files.go` (`ListFiles`), `gitcmd.go`'s `runGit` |
| `internal/execx`                  | `Run` - the one external-process seam (timeout, env, captured stderr, typed `*execx.Error`)          |
| `internal/report`                 | `Report{Notes,Warnings}` with `Note`/`Warn`/`Merge` - the one install/uninstall report value         |
| `internal/jsonconfig`             | `Read` - read a JSON config file, tolerating an absent one, shared by every merge/unmerge site       |
| `internal/hook`                   | `hook.go`/`dispatcher_deps.go` (`Dispatcher`), `router.go` (`Router`), `session.go` (`SessionTracker`), `checks.go` (`RunGovernanceCheck`), `checklist/`, `install/*` (per-target hook installers) |
| `internal/text`                   | `escape` (control-character escaping for printed output), `clean`/`mark` (invisible-text cleaning)    |
| `internal/config`                 | `GovernanceConfig` and its sub-configs, `presets.go`                                                 |
| `internal/obs`                    | `New` (off-by-default `slog.Logger`), tail-truncating log rotation                                   |
| `internal/emoji`                  | `Catalog`, `Anchors`/`themes.go`, `data/{catalog,types,themes}.txt`                                  |
| `internal/gitignore`              | `Ensure` - idempotent `.gitignore` entries                                                           |
| `internal/selfbin`                | `Copy` - copies the running executable to install it as a hook/plugin binary                         |
| `internal/evals`                  | `model.go`/`load.go` (embedded eval scenarios, `data/<skill>.json`), `validate.go` (`ValidateCheck`), `grade.go` (`Grade`) |

## The router skill

`project-router` is mandatory and always-loaded first. Its dispatch table maps a task signal to the
skills to load, in composition order; the rows live in `knowledge/standards/router.yaml` and
`knowledge/templates/skills/router.md.tmpl` renders them. The rows include engineering tasks such as
new app, bootstrap/DI, use case, REST/gRPC endpoint, entity/migration, sqlc query, multi-write tx,
outbox event, worker, permission, external provider, test, module boundary, refactoring and
architecture review, and others for the Dart, Svelte, l10n and agent-delegation areas. The vertical
Dependency Rule (`dependency-rule`) and architecture fitness functions (`fitness-functions`) govern
layer direction and CI enforcement; an `architecture-decision-records` workflow records significant
decisions.

## Skill-effectiveness evals

Rendering a skill and having an analyzer catch violations both prove the pipeline is stable; neither
proves a skill's *text* changes what an agent produces. `internal/evals` measures that, for a starting
set of four skills (`code-minimalism`, `integration-tests`, `commits`, and the `safety` rule composed
into `go-style`), following the eval-scenario method `samber/cc-skills-golang` documents: per skill,
a handful of scenarios, each a realistic `prompt`, the specific `trap` the skill exists to prevent, and
plain-English `assertions` a produced artifact is graded against.

Scenarios live under `internal/evals/data/<skill>.json`, embedded via `//go:embed` like
`internal/emoji/data/*.txt`, not under `knowledge/standards/`, because they are not rendered into
a `SKILL.md`; they only drive this package's own validation and grading, so they stay colocated
with the code that reads them. Every assertion in `model.go`'s `Assertion` carries a `Kind`:
`"machine"` assertions carry a `Check` (`contains`/`not_contains`/`regex_match`/`regex_absent`)
`Grade` runs directly against an artifact's text; `"judge"` assertions carry none and are left for
a human or LLM reader.

`evals.ValidateCheck` matches `validate.Check`'s `func(*knowledge.Index) []string` shape and is
injected via the same `extra` seam `registry.RuleAnalyzersCheck` uses
(`internal/cli/validate.go`): every scenario's `skill` must resolve against `skills.yaml`
(`idx.SkillExists`), every scenario must declare a non-empty `trap` and at least one assertion,
every assertion's `Kind` must be `machine` (with a `Check`) or `judge` (without one), and a
`Check` must have a known type and, for the two regex types, a pattern that compiles.

`evals.Grade(scenario, artifact)` scores every `machine` assertion's `Check` against the artifact
string and reports `pass`/`fail`; `judge` assertions report `needs_judge` untouched, since `Grade`
has no model to read the artifact with.

The harness does not invoke a model. Producing the with-skill and without-skill artifacts a
scenario is graded against is an operator procedure, not something this binary runs: (1) start two
sessions on the same scenario's `prompt`, one with the target skill loaded (or its knowledge
composed into context) and one without any of this repo's guidance; (2) save each session's
produced code/transcript as a plain-text artifact; (3) run `pulsarules_cli evals --artifact <file>`
on both and diff the `machine` tallies (the command is the one call site of `evals.Grade`, so this
step is executable rather than a Go snippet the reader has to write); (4) read every `needs_judge`
assertion against both artifacts and score it by hand (or via a separate LLM-judge call this repo
does not provide); (5) aggregate pass/fail across the scenarios run so far into a with-score, a
without-score, and a delta - and flag the same two anomalies `cc-skills-golang` does: low delta
with a high without-score (the skill may not be earning its bytes) and a low with-score (the
skill's own text needs rewriting).

## Design decisions

Cross-cutting choices whose reason no longer sits beside one line of code.

- `config.AnalyzerConfig` mirrors `core.AnalyzerConfig` field for field on purpose, so
  `internal/analyzer/core` imports only the standard library and its own subpackages;
  `analysis.toAnalysisConfig` is the single converter.
- `os.Exit` is called only in `main`. A command that must set the exit status returns a
  `*cli.ExitError`; only `cmd/pulsarules_cli/main.go` turns it into a process exit.
- There is one composition root: `bootstrap.DoInjections` is the only place that registers
  collaborators in the injector; command handlers resolve what they need from it and nothing else
  reads it.
- `clean`'s `mode` is a named type, not a bool, so no call site can pass the rewriting value by
  accident; only `--write` selects it.
- `execx.Error` carries the invocation and its trimmed stderr, so callers wrap it without
  re-parsing. `execx.Result.Stdout` stays raw for callers to trim.
- `vcs` shells out to the `git` binary, because this package must match git's own status,
  rename and listing output.
- `langdetect.Language` is a plain string so it round-trips through the persisted config and
  `--languages` with no translation table; a monorepo can mix languages, so detection, config and
  the override all deal in a `Set`. An empty detected `Set` means unknown (markers not written
  yet), never "no language", so `Resolve` never persists a detection that found nothing.
- `report.Report` flows from producer through installer to the printer that owns stdout/stderr;
  callers `Merge` it without gating on whether it holds anything.
- A malformed data record is a build defect, so `emoji.NewCatalog` returns an error instead of a
  narrower catalog, and `readRecords` takes an `fs.FS` so tests inject a fabricated one rather
  than touching the embedded data.
- Finding text and paths are untrusted file content, so `internal/text/escape` quotes control
  characters when they are printed, never in the analyzers.

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
compliant with that rule, and the ~48 files that import it are not a reason to call it compliant.
It stays top-level for two reasons: `knowledge/embed.go` does `//go:embed standards templates`,
and `go:embed` can only reach files at or below the package that declares the directive, so the
embed forces `knowledge/` to sit above `standards/`; and burying `standards/` under `pkg/` would
defeat the portability guarantee above: it must stay usable as an engineering handbook on its own,
independent of `pkg/`'s Go-API framing. Revisit this placement if a second module or binary ever
appears in this repo.
