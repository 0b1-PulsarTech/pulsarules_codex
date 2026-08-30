---
id: dependency-rule-go
name: The Dependency Rule - Go package denylist
description: The Go instantiation of the Dependency Rule - internal/domain/{entities,usecases} imports no database/sql, SQL driver, ORM/ent runtime, net/http, grpc, or proto package; depguard enforces the denylist; the composition root is cmd/<app>.
tags:
    - go
    - architecture
    - layout
linters:
    - depguard
analyzers:
    - arch-boundary
---

# The Dependency Rule - Go package denylist

> In this repo's Go layout, `internal/domain/{entities,usecases}` imports no `database/sql`, SQL
> driver, ORM/ent runtime, `net/http`, `google.golang.org/grpc`, proto package, or web framework;
> `depguard` enforces the denylist; the concrete detail is wired at `cmd/<app>`. This is the Go
> package-level instantiation of [[dependency-rule]]'s inward-only principle.

Applies to: a Go project laid out per [[code-placement]]. Composes with [[dependency-rule]].

{{define "when"}}
- Adding an import to a domain (`entities`/`usecases`) package.
- Configuring or reviewing the `depguard` denylist.
- Reviewing a diff for an inward-pointing dependency violation in Go.
{{end}}

{{define "must"}}
1. `internal/domain/{entities,usecases}` imports no `database/sql`, SQL driver, ORM/ent runtime,
   `net/http`, `google.golang.org/grpc`, proto package, web framework, or other infra/transport
   package - only the standard library, the domain itself, and its own ports.
2. `depguard` encodes the domain import denylist in the linter config, so a violation fails CI, not
   just review.
3. Pick the concrete detail (driver, framework, transport) only at the composition root
   (`cmd/<app>`); the domain stays unaware of which one was chosen.
{{end}}

{{define "forbidden"}}
- A domain (`entities`/`usecases`) package importing infra, transport, a DB driver, an ORM runtime,
  a proto package, or a web framework.
- A domain import denylist that exists only in prose, with no `depguard` entry.
{{end}}

{{define "validation"}}
- [ ] No infra/transport/driver/ORM/proto/framework import inside `internal/domain/`.
- [ ] `depguard` encodes the domain import denylist.
- [ ] The concrete detail is chosen at `cmd/<app>`, not in the domain.
{{end}}
