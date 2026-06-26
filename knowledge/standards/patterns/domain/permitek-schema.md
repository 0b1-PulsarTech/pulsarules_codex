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

{{define "when"}}
- A module declaring its own permission schema.
- Granting/revoking permissions.
- Marshalling permissions for the DB column or the JWT claim.
{{end}}

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

{{define "forbidden"}}
- `reflect` anywhere in permission handling.
- Constructing `Permission[M]` by hand; reordering/removing/renumbering schema names or module ids.
- A package-level/global `Registry`; more than 64 permissions in one module.
- A permission check inside the use case, or an unused `associatedUser` field; gate at the call site.
- `isAdmin` boolean/column.
{{end}}

{{define "validation"}}
- [ ] Module has a zero-size marker with stable `ID()`; schema is `DefineModule` and append-only.
- [ ] Handles minted via `Schema.Perm`; permission enforced at the call site via the `access` gate;
  the use case is pure (no `Has`/`associatedUser`).
- [ ] Registry built at boot and injected; no `reflect`; no `isAdmin`.
- [ ] One marshal format for both the DB column and the JWT claim.
{{end}}
