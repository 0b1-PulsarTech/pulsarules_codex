---
id: transport
name: Transport-agnostic use cases
description: Domain code imports no net/http/grpc/proto; handlers only parse -> call use case -> map -> respond.
tags:
    - go
    - transport
dependencies:
    - errors
linters:
    - depguard
---

# Transport-agnostic use cases

> Domain code imports no `net/http`, `grpc`, or proto; handlers only parse -> call use case -> map
> -> respond; the same use case is callable from HTTP, gRPC, GraphQL, CLI, or a background job
> unchanged.

Applies to: use cases and transport handlers.

{{define "when"}}
- Writing an HTTP or gRPC handler.
- Designing request/response DTOs vs domain entities.
- Deciding where validation vs business logic lives.
{{end}}

{{define "must"}}
1. The use case's `Input` struct, the use case, and its `Output`/entities import no `net/http`,
   `google.golang.org/grpc`, proto package, or web framework.
2. Handler shape is strictly: parse request to validate wire format (required fields, max length)
   -> call `uc.Execute(ctx, input)` -> map entities to a response DTO -> write status/body. No
   business rules, SQL, or domain invariants in the handler.
3. The handler knows nothing about DB rows or domain validation; the use case knows nothing about
   HTTP status, JSON, or gRPC status. Domain errors flow out as the domain-error type and middleware
   maps them (see [[errors]]).
4. Wire-format validation happens at the boundary; business invariants are enforced in the use case.
{{end}}

{{define "forbidden"}}
- Importing `net/http`/grpc/proto/web-framework inside the domain.
- Business rules, permission checks, or domain validation in a handler.
- Returning raw generated row types / proto / `*sql.Rows` from a use case.
- A use case that can only be called from one transport.
{{end}}

{{define "validation"}}
- [ ] No `net/http`/grpc/proto/web-framework import inside `internal/domain/`.
- [ ] Handler only parses, calls the use case, maps, and responds.
- [ ] Wire-format validation at the boundary; business invariants in the use case.
- [ ] Domain errors flow out as the domain-error type; no status codes written from the use case.
{{end}}
