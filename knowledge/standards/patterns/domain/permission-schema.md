---
id: permission-schema
name: Per-module permission schema
description: A zero-size phantom marker with a stable ID; an append-only DefineModule schema; typed Permission handles; checks at the call site via the access gate; a registry built at boot and injected; one marshal format for DB column and JWT claim.
tags:
    - go
    - security
    - authorization
dependencies:
    - authorization
    - dependency-injection
---

# Per-module permission schema

> A zero-size phantom marker type with a stable `ID()`; an append-only `DefineModule` schema; typed
> `Permission[M]` handles minted via the schema's `Perm`, injected or declared as package-level enum
> handles; checks at the call site via the `access` gate; a registry built at boot and injected; one
> marshal format for the DB column and the JWT claim. Reflection-free; caps a module at 64
> permissions.

Reference tools: a bitwise, reflection-free permission engine; Go 1.26 recursive
generic types.

{{define "recipe"}}
```go
package thing

import permits "example.com/permission"

type moduleMod struct{} // zero-size phantom marker

func (moduleMod) ID() permits.ModuleID { return permits.ModuleIDFromName("thing") }

func Schema() permits.Schema[moduleMod] {
    // Append-only; new names at the end; never reorder/remove/renumber.
    return permits.DefineModule[moduleMod]("view", "create", "manage")
}

type Perms struct {
    View, Create, Manage permits.Permission[moduleMod]
}

func NewPerms(schema permits.Schema[moduleMod]) Perms {
    return Perms{
        View:   schema.Perm("view"),
        Create: schema.Perm("create"),
        Manage: schema.Perm("manage"),
    }
}
```

The id derives from the name: renaming `"thing"` changes it and invalidates stored grants; two names
folding to one 16-bit id make `Register` fail with `ErrDuplicateModule`.

Build `Perms` once at the composition root from `Schema()` and inject it where a permission is
named. A package-level `var Schema = permits.DefineModule[moduleMod](nameView, nameManage)` with
`PermView = Schema.Perm(nameView)` handles is also allowed as an enum, not global state, when
`Schema` and the handles are in the same file, `Schema` is built only from constants (literals or
`const` of the package), and every handle calls `Schema.Perm` with constant arguments. The handles
may sit in a `var ( ... )` block or stand alone, so a single-permission module passes too. The
Registry is never package-level.

Enforce at the call site with the `permission/access` gate; the use case stays pure (no permission
check inside it). The principal is injected for identity/audit, not as the authz decision:

```go
// At the call site (application service / command handler), perms is injected:
create := access.Guard(perms.Create, uc.Create) // uc has no permission check
out, err := create(ctx, in)
// or, inline: if err := access.Check(ctx, perms.Create); err != nil { return ..., err }
```

`Guard`/`GuardQuery`/`GuardCommand`/`Check` call the underlying `Has` predicate and return an error
wrapping `access.ErrForbidden` (or `access.ErrUnauthenticated` with no `Set` on `ctx`); the
transport's error middleware maps them by `errors.Is`, ahead of the generic status extraction (see
the permission-gate exception in the [[errors]] rule). Use cases never call `Has` themselves.

The gate reads the caller's grants from `ctx`, not from the principal: the auth middleware
base64-decodes the JWT `perm` claim, rebuilds the `Set` with `registry.Unmarshal`, and stores it
with `ctx = access.NewContext(ctx, set)` before any guarded route or call runs. With no `Set` on
`ctx` every check returns an error wrapping `access.ErrUnauthenticated`.

Grant/revoke by assigning the result back (`granted = Grant(granted, perms.X)` / `Revoke(...)`);
they write through the Set's map, and the library has no exported clone, so copy it first through
the codec (`raw, err := registry.Marshal(set)`, then `registry.Unmarshal(raw)`; only registered
modules survive) if the original must stay unchanged. Admin = all declared bits:

```go
granted = permits.Grant(granted, perms.Create) // assign the result back
admin := registry.Admin()
```

Registry at boot (fold errors; inject as singleton):

```go
reg := permits.NewRegistry()
if err := errors.Join(reg.Register(thing.Schema()), reg.Register(funnel.Schema())); err != nil {
    return fmt.Errorf("register permission schemas: %w", err)
}
remy.RegisterInstance(inj, reg)
remy.RegisterInstance(inj, thing.NewPerms(thing.Schema()))
```

One marshal format for both sinks:

```go
raw, err := reg.Marshal(user.Permissions) // DB VARBINARY column
if err != nil {
    return fmt.Errorf("marshal permissions: %w", err)
}
b64 := base64.StdEncoding.EncodeToString(raw) // JWT "perm" claim
```
{{end}}
