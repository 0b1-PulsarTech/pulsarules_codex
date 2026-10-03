---
id: naming
name: Naming
description: MixedCaps identifiers, lowercase single-word packages, correct acronym casing, no stuttering.
tags:
    - go
    - style
linters:
    - stylecheck
    - revive
analyzers:
    - naming
    - single-letter-name
---

# Naming

> MixedCaps identifiers, lowercase single-word packages, correct acronym casing, no stuttering.

Applies to: all Go identifiers, packages, files, and tests.

{{define "when"}}
- Naming a package, type, function, constant, file, or test.
- Reviewing names for clarity and consistency.
{{end}}

{{define "must"}}
1. MixedCaps for exported, mixedCaps for unexported. No `snake_case` identifiers.
2. Packages: lowercase, single word, no underscores, no generic names (`util`, `common`, `helpers`).
3. Spell acronyms `ID`, `HTTP`, `URL`, `JWT`, `RPC`, `API` (never `Id`, `Http`, `Url`, `Jwt`).
4. Do not stutter: `secretfriend.SecretFriend`, not `secretfriend.SecretFriendService`; avoid
   `apperr.NewError` -> `apperr.New`.
5. Files are `snake_case.go`; group action verbs (`create_thing.go`, `list_things.go`).
6. Colocate tests as `<source>_test.go` next to `<source>.go`.
7. One concept per file (~180 lines, ~468 for a `_test.go`); split before exceeding it.
8. No identifier is a single letter, anywhere: variables, loop variables (even a pure index),
   parameters, results, receivers, struct fields, interface methods, functions, types, type
   parameters (receiver type parameters included), constants, labels, import aliases and the package
   name. A short abbreviation is fine (`srv`, `row`, `idx`); only `_` and a `testing` T/B/F/M/TB
   parameter are exempt (`single-letter-name`).
9. Name a consumer-declared collaborator port for its role - what the consumer needs from it
   (`Repository`, `Store`, `Resolver`, `Sender`, `Notifier`) - never a generic `Port` or a
   transport-leaking name like `metaAPIPort`, even when the implementation is an HTTP-API client.
10. Getters do not carry a `Get` prefix (`Name()`, not `GetName()`); reserve the `Is`/`Has`/`Can`
    prefixes for boolean-returning predicates (`IsValid()`, `HasChildren()`, `CanRetry()`).
11. A boolean variable or struct field name reads as a true/false question - prefixed
    `is`/`has`/`can` (`isActive`, `hasChildren`, `canRetry`), never a bare noun or adjective. This
    generalises the anti-`isAdmin` special case in [[authorization]] (which forbids a permission
    boolean outright) to every boolean identifier: even where a boolean is the right type, its name
    must ask the question it answers.
{{end}}

{{define "forbidden"}}
- `snake_case` identifiers; generic `util`/`common`/`helpers` package names.
- `Id`/`Http`/`Url`/`Jwt` casing.
- Stuttering type/function names (`foo.FooService`).
- A single-letter identifier of any kind, receivers and loop variables included.
- Misleading abbreviations.
- A generic `Port`/`metaAPIPort` collaborator-port name; name the port for its role.
- A `Get`-prefixed getter (`GetName()` instead of `Name()`).
- A boolean variable or field whose name does not read as an `is`/`has`/`can` question (`admin bool`
  instead of `isAdmin`, `active bool` instead of `isActive`).
{{end}}

{{define "validation"}}
- [ ] No `snake_case` identifiers; no `util`/`common`/`helpers` packages.
- [ ] Acronyms cased correctly (`ID`, `HTTP`, `URL`).
- [ ] No stuttering type/func names.
- [ ] Files `snake_case.go`; tests colocated as `<source>_test.go`.
- [ ] No single-letter identifier of any kind; only `_` and a testing T/B/F/M/TB parameter.
- [ ] Getters carry no `Get` prefix; boolean predicates use `Is`/`Has`/`Can`.
- [ ] Boolean variables/fields read as an `is`/`has`/`can` true/false question.
{{end}}
