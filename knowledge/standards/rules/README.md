# `rules/`

Mandatory, enforceable rules for Go services. Each file describes one decision area; together they
form the engineering contract for any new or modified Go code. Rules are phrased as tool-agnostic
principles; where a rule is automateable, the `## Linters` section names the linter.

Rules are grouped into themed subdirectories to reduce directory noise. The ID in each file's
frontmatter is the canonical reference — file location does not affect how skills compose rules.

## Generic section defines

Each rule/pattern body declares its sections as bare, generic `{{define}}` blocks on their own lines
(the preamble - H1, summary, `Applies to:` - stays as loose text above them):

```
{{define "when"}}
- A function returns an error.
{{end}}

{{define "must"}}
1. Return `(T, error)` as the last value ...
{{end}}
```

The canonical section keys are `when`, `catalog`, `must`, `recipe`, `approved`, `rejected`,
`examples`, `forbidden`, `validation`. The validator rejects any other define name (it would never
render). Bodies are logic-free named text rendered with `missingkey=error`.

## Merged composition

The skill's curated sidecar is the `##`-level head (the actionable synthesis). Below it, when a skill
composes several rules/patterns, the renderer's controller parses each body (one namespace per file,
so the repeated `must`/`forbidden` names do not collide) and **merges same-keyed sections** as the
source detail, under a single `## Applicable rules and patterns` parent (`### <Section>` then
`#### <Name>`):

```
## Applicable rules and patterns

### Must

#### Effective Go subset
1. Format with the project formatter ...

#### Naming
1. MixedCaps for exported ...
```

A skill opts out with `no_merge: true` in `skills.yaml`, which instead groups by source first
(`### <Rule>` then `#### <Section>`). The same per-file generic names are the seam for future
multi-language variants.

## `style/` — how code looks and reads

| Rule                                         | Topic                                          |
|----------------------------------------------|-------------------------------------------------|
| [effective-go.md](style/effective-go.md)     | Effective Go subset we enforce                 |
| [naming.md](style/naming.md)                 | Go naming for packages, types, files           |
| [imports.md](style/imports.md)               | Three-group import layout + aliasing           |
| [types.md](style/types.md)                   | Named structs and small consumer interfaces    |
| [minimalism.md](style/minimalism.md)         | Function-scope decision ladder                 |
| [flag-arguments.md](style/flag-arguments.md) | No flag/output args; small arity               |

## `architecture/` — where code lives and how layers connect

| Rule                                                                             | Topic                                            |
|-----------------------------------------------------------------------------------|----------------------------------------------------|
| [code-placement.md](architecture/code-placement.md)                             | apps/libs/tools/build vs cmd/internal/pkg        |
| [code-placement-monorepo.md](architecture/code-placement-monorepo.md)           | Monorepo variant of code placement               |
| [code-placement-inner-modules.md](architecture/code-placement-inner-modules.md) | Inner-module variant of code placement           |
| [dependency-injection.md](architecture/dependency-injection.md)                 | Constructor-based DI; bootstrap is the switchboard |
| [dependency-rule.md](architecture/dependency-rule.md)                           | Inward-only layer dependencies                   |
| [fitness-functions.md](architecture/fitness-functions.md)                       | Architecture invariants as automated CI checks   |
| [frameworks-as-plugins.md](architecture/frameworks-as-plugins.md)               | Frameworks/DB/web as details behind ports        |
| [interop.md](architecture/interop.md)                                           | Cross-module calls via facades                   |
| [module-boundaries.md](architecture/module-boundaries.md)                       | Cohesion, coupling, and the main sequence        |
| [transport.md](architecture/transport.md)                                       | Transport-agnostic use cases                     |

## `domain/` — Go runtime behavior and cross-cutting concerns

| Rule                                                                   | Topic                                           |
|------------------------------------------------------------------------|--------------------------------------------------|
| [authorization.md](domain/authorization.md)                           | Bitwise reflection-free permissions            |
| [command-query-separation.md](domain/command-query-separation.md)     | Commands return error; queries side-effect-free |
| [concurrency.md](domain/concurrency.md)                               | Goroutine ownership, `context` discipline      |
| [config.md](domain/config.md)                                         | Typed `Config` via a config loader             |
| [errors.md](domain/errors.md)                                         | Sentinel errors, `%w` wrapping                 |
| [logging.md](domain/logging.md)                                       | `log/slog` structured logging                  |
| [security.md](domain/security.md)                                     | Secrets, validation, JWT, SQL, containers      |
| [startup.md](domain/startup.md)                                       | Zero side effects in `init()`                  |

## `infra/` — specific infrastructure and tooling

| Rule                                         | Topic                                            |
|----------------------------------------------|--------------------------------------------------|
| [build.md](infra/build.md)                   | CGO-free build; Taskfile; reproducible codegen   |
| [database.md](infra/database.md)             | ent schema + sqlc queries + Atlas migrations     |
| [eventing.md](infra/eventing.md)             | Transactional outbox for cross-boundary effects  |
| [grpc.md](infra/grpc.md)                     | Consuming/serving gRPC against a proto module    |
| [http-clients.md](infra/http-clients.md)     | One outbound HTTP gateway; per-request timeouts  |

## `process/` — engineering habits and code health

| Rule                                         | Topic                                                       |
|----------------------------------------------|-------------------------------------------------------------|
| [code-smells.md](process/code-smells.md)     | Smell catalog + Go-linter remedies                          |
| [commits.md](process/commits.md)             | Emoji-prefixed Conventional Commits                         |
| [testing.md](process/testing.md)             | Colocated table-driven tests; real-DB integration tests     |

## How rules relate to skills

Rules are the knowledge; skills are generated. A skill composes one or more rules (and patterns) -
see `skills.yaml` and `pulsarules_codex-installer list skills`. Multiple rules can collapse into one
capability skill (e.g. `effective-go` + `naming` + `imports` + `types` -> `go-style`).
