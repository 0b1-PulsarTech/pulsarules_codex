---
id: authorization
name: Authorization
description: Bitwise, reflection-free, per-module permission schemas; checks at the call site via the access gate (never ad-hoc isAdmin); a registry built at boot and injected; one marshal format for DB column and JWT claim.
tags:
    - go
    - security
    - authorization
dependencies:
    - errors
    - dependency-injection
---

# Authorization

> Bitwise, reflection-free, per-module permission schemas; checks at the call site via the `access`
> gate (never an ad-hoc `isAdmin`); a registry built at boot and injected; one marshal format for
> the DB column and the JWT claim.

Applies to: protecting endpoints/use cases and declaring a module's permissions. Canonical reference
engine: a bitwise, reflection-free per-module permission library (source repos name it `permitek`).

{{define "when"}}
- Protecting an endpoint or use case with a permission.
- A module declaring its own permission schema.
- Granting/revoking permissions to an operator.
- Marshalling permissions for the DB column or the JWT claim.
- Wiring the permission registry at bootstrap.
{{end}}

{{define "must"}}
1. In the module package, define a zero-size phantom marker type with a stable, never-reused
   `ID()` derived from the module name (`ModuleIDFromName("thing")`); renaming it invalidates stored
   grants.
2. Declare the append-only schema: `DefineModule[Marker]("view", "send", "manage")`. New names go at
   the end; never reorder/remove/renumber.
3. Mint typed handles via the schema's `Perm("send")` once, in a constructor-built value (type
   `Permission[Marker]`). Never construct a `Permission[M]` by hand.
4. Enforce the permission at the call site with the `permitek/access` gate
   (`Guard`/`GuardQuery`/`GuardCommand`/`Check`): it reads the caller's `Set` from ctx (stored by
   `access.NewContext` in the auth middleware) and returns an error wrapping `access.ErrForbidden`
   (`access.ErrUnauthenticated` with no `Set` on ctx) before the operation runs; the transport's
   error middleware maps it. The required permission is a typed `Permission[M]` handle from the
   injected `Perms` (`perms.Create`). The gate is the authorization authority.
   4a. The use case stays pure: it contains no permission check. Inject the principal for
   identity/audit (who acted), not as the authz decision; do not keep an `associatedUser` field on
   a use case that never reads it. (`Has` is still the underlying predicate the gate calls; use
   cases don't call it for authz.)
5. Grant/revoke by assigning the result back (`granted = Grant(granted, perms.X)` / `Revoke(...)`);
   they write through the Set's map, so copy first if the original must stay unchanged (no exported
   clone: `raw, err := registry.Marshal(set)`, then `registry.Unmarshal(raw)`; only registered
   modules survive). Admin = `registry.Admin()` (all declared bits), never an `isAdmin` boolean.
6. Build the registry at boot and inject it: `NewRegistry()` then `reg.Register(thing.Schema())`
   for every module (fold errors with `errors.Join`); register as a singleton. Never a
   package-level global.
7. Marshal one format for both sinks: `registry.Marshal(set)` for the permissions DB column and the
   base64 JWT claim; `Unmarshal` rebuilds the `Set` when loading the principal per request.
8. The call-site `access` gate is the enforced check; a transport-route guard may still fail-fast
   coarsely, but the gate is authoritative. Permission changes take effect on the next request
   (loaded per request, not cached in the token).
9. Cap a module at 64 permissions; split into sub-modules via a composite constraint when it
   genuinely approaches the limit.
{{end}}

{{define "forbidden"}}
- `reflect` anywhere in permission handling.
- Constructing `Permission[M]` by hand; reordering/removing/renumbering schema names or module ids.
- A package-level/global `Registry`; the permission engine importing an app module.
- `isAdmin` boolean/column.
- A permission check inside the use case, or an `associatedUser` field a use case never reads
  (gate at the call site; keep the use case pure).
- Re-parsing the JWT inside the use case; more than 64 permissions in one module.
- Writing HTTP 403 from the use case (the gate returns `access.ErrForbidden`).
{{end}}

{{define "validation"}}
- [ ] Module has a zero-size marker with a stable `ID()`; schema is `DefineModule` and append-only.
- [ ] Handles minted via the schema's `Perm`; no hand-built `Permission[M]`.
- [ ] Permission enforced at the call site via the `access` gate
  (`Guard`/`GuardQuery`/`GuardCommand`/`Check`) with a typed `Permission[M]` handle from the
  injected `Perms`; `access.ErrForbidden` on failure.
- [ ] Use case is pure (no permission check); principal injected only for identity/audit; no unused
  `associatedUser` field.
- [ ] No `isAdmin` column/boolean; admin via `registry.Admin()`.
- [ ] Registry built at boot and injected; never global; no `reflect`.
- [ ] One `Marshal` format for both the DB column and the JWT claim.
{{end}}
