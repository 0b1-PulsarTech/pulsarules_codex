---
id: permitek-schema
name: Per-module permission schema
description: A zero-size phantom marker with a stable ID; an append-only DefineModule schema; typed Permission handles; checks via Has; a registry built at boot and injected; one marshal format for DB column and JWT claim.
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
> `Permission[M]` handles minted via `Schema.Perm`; checks in the use case via the free `Has`
> function; a registry built at boot and injected; one marshal format for the DB column and the JWT
> claim. Reflection-free; caps a module at 64 permissions.

Reference tools: a bitwise, reflection-free permission engine (`permitek`); Go 1.26 recursive
generic types.

{{define "recipe"}}
```go
package thing

type moduleMod struct{} // zero-size phantom marker

const moduleID = 0x0003 // stable, never reused

func (moduleMod) ID() permits.ModuleID { return moduleID }

// Append-only; new names at the end; never reorder/remove/renumber.
var Schema = permits.DefineModule[moduleMod]("view", "create", "manage")

var (
    PermThingView   = Schema.Perm("view")
    PermThingCreate = Schema.Perm("create")
    PermLeadManage = Schema.Perm("manage")
)
```

Enforce at the CALL SITE with the `permitek/access` gate; the use case stays PURE (no permission check
inside it). The principal is injected for identity/audit, not as the authz decision:

```go
// At the call site (application service / command handler), gate before invoking the use case:
if err := access.GuardCommand(ctx, principal, thing.PermThingCreate); err != nil {
    return entities.Thing{}, err // apperr.ErrForbidden
}
out, err := uc.Create(ctx, in) // uc has no permission check and no associatedUser field
```

`Guard`/`GuardQuery`/`GuardCommand` call the underlying `permits.Has` predicate; use cases never call
`Has` for authorization themselves.

Grant/revoke (functional; admin = all declared bits):

```go
perms = permits.Grant[thing.moduleMod](perms, thing.PermThingCreate)
admin := registry.Admin()
```

Registry at boot (fold errors; inject as singleton):

```go
reg := permits.NewRegistry()
for _, schema := range []permits.Schema{thing.Schema, funnel.Schema} {
    if err := reg.Register(schema); err != nil {
        return fmt.Errorf("register permission schema: %w", err)
    }
}
remy.RegisterInstance(inj, reg)
```

One marshal format for both sinks:

```go
bytes := reg.Marshal(user.Permissions)        // DB VARBINARY column
b64 := base64.StdEncoding.EncodeToString(bytes) // JWT "perm" claim
```
{{end}}

<!-- No when/forbidden/validation blocks here on purpose: every line this pattern used to
     carry was a subset of [[authorization]]'s own, and the two only ever render into the same
     skill, so the reader met each obligation twice. The rule owns them; this file owns the
     recipe. -->
