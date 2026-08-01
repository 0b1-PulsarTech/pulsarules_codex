---
id: fixture-builder
name: Fixture builder
description: Immutable, chainable test-fixture builders in a shared <pkg>test helper - New* seeds a valid baseline, each With* returns a copy so a base fixture branches without shared mutation, and a conformance assertion pins the produced type to the interface under test.
tags:
    - go
    - testing
dependencies:
    - testing
---

# Fixture builder

> Build test data with a small chainable builder, not a pile of struct literals copied across tests.
> `New*` seeds a valid baseline; each `With*` overrides one field and returns the builder so calls
> chain. Put it in a shared `<pkg>test` helper package so every test package reuses it instead of
> copy-pasting a fake. The subtle rule: make the builder immutable - each `With*` returns a copy - so
> a shared base fixture can be branched per test without one case mutating another's data.

Reference: `terectek_comms` `libs/tereckernel/pkg/webwrap/webwraptest` (`Reader`, `NewReader`,
`WithHeader`/`WithBody`) - a shared test double built exactly this way (though value-copy immutability
is the correctness upgrade this pattern insists on).

{{define "when"}}
- Several tests (or several packages) need the same shaped fixture with small per-test variations.
- A test double / canned reader is being copied into more than one test package.
- A base fixture is branched into variants and must not share mutable state across cases.
{{end}}

{{define "recipe"}}
```go
type Builder struct { // value type; methods take a value receiver and return a copy
    user User
}

func New() Builder { // seed a valid baseline so most tests call New() with one override
    return Builder{user: User{ID: "u-1", Name: "Ada", Status: StatusActive}}
}

func (builder Builder) WithStatus(status Status) Builder { // copy in, copy out
    builder.user.Status = status
    return builder
}

func (builder Builder) WithName(name string) Builder {
    builder.user.Name = name
    return builder
}

func (builder Builder) Build() User { return builder.user }
```

Branch a shared base without cross-test mutation; `active` is Grace as built, `banned` is Grace
with the status changed, and `base` stays untouched:

```go
base := userfixture.New().WithName("Grace")
active := base.Build()
banned := base.WithStatus(StatusBanned).Build()
```

Pin a built test double to the interface it must satisfy:

```go
var _ webwrap.RequestReader = (*Reader)(nil)
```
{{end}}

{{define "forbidden"}}
- A mutable builder (pointer receiver mutating in place and returning `self`) shared as a base - one
  test's `With*` silently changes another's fixture. Use a value receiver returning a copy.
- Copy-pasting the same fake/builder into each test package instead of a shared `<pkg>test` helper.
- A `New*` that seeds an invalid baseline, forcing every test to set required fields before use.
- A builder for a slice/map field that stores the caller's reference (aliasing) instead of cloning it.
{{end}}

{{define "validation"}}
- [ ] The builder lives in a shared `<pkg>test` helper, reused across packages, not copy-pasted.
- [ ] `New*` seeds a valid baseline; `With*` overrides one thing and returns the builder.
- [ ] Immutable: a branched base fixture is unaffected by a later `With*` on a sibling (value copy).
- [ ] Reference-typed fields are cloned, not aliased from the caller.
- [ ] A produced test double carries `var _ Iface = (*impl)(nil)`.
{{end}}
