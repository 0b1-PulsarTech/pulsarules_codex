---
id: authorization
name: Authorization
---

## Mandatory workflow

1. In the module package, define a zero-size phantom marker type with a stable, never-reused `ID()` (e.g.
   `const moduleID = 0x0007`).
2. Declare the append-only schema: `DefineModule[Marker]("view", "send", "manage")`. New names go at the end; never
   reorder/remove/renumber.
3. Mint typed handles via `Schema.Perm("send")` (type `Permission[Marker]`). Never construct a `Permission[M]` by hand.
4. Enforce the permission at the CALL SITE with the `permitek/access` gate
   (`Guard`/`GuardQuery`/`GuardCommand`), passing a typed permission constant; it returns `apperr.ErrForbidden` before
   the operation runs and is the authorization authority. The use case stays PURE - no permission check inside it.
   Inject the principal for IDENTITY/AUDIT only; do NOT keep an `associatedUser` field a use case never reads. (`Has` is
   the underlying predicate the gate calls; use cases don't call it for authz.)
5. Grant/revoke functionally: `Grant[Marker](perms, PermX)` / `Revoke[...]`. Admin = `registry.Admin()` (all declared
   bits), never an `isAdmin` boolean.
6. Build the registry at boot and inject it: `NewRegistry()` then `reg.Register(Schema)` for every module (fold errors
   with `errors.Join`); register as a singleton. Never a package-level global.
7. Marshal one format for both sinks: `registry.Marshal(set)` for the permissions DB column and the base64 JWT claim;
   `Unmarshal` rebuilds the `Set` when loading the principal per request.
8. The call-site `access` gate is the enforced check; a transport-route guard may still fail-fast coarsely, but the gate
   is authoritative. Permission changes take effect on the next request (loaded per request, not cached in the token).
9. Cap a module at 64 permissions; split into sub-modules via a composite constraint when it genuinely approaches the
   limit.

## Validation checklist

- [ ] Module has a zero-size marker with a stable `ID()`; schema is `DefineModule` and append-only.
- [ ] Handles minted via `Schema.Perm`; no hand-built `Permission[M]`.
- [ ] Permission enforced at the call site via the `access` gate (`Guard`/`GuardQuery`/`GuardCommand`) with a typed
  constant; `apperr.ErrForbidden` on failure; use case pure (no `Has`/`associatedUser`).
- [ ] No `isAdmin` column/boolean; admin via `registry.Admin()`.
- [ ] Registry built at boot and injected; never global; no `reflect`.
- [ ] One `Marshal` format for both the DB column and the JWT claim.

## Forbidden actions

- `reflect` anywhere in permission handling.
- Constructing `Permission[M]` by hand; reordering/removing/renumbering schema names or module ids.
- A package-level/global `Registry`; the permission engine importing an app module.
- `isAdmin` boolean/column.
- A permission check inside the use case, or an `associatedUser` field a use case never reads (gate at the call site).
- Re-parsing the JWT inside the use case; more than 64 permissions in one module.
- Writing HTTP 403 from the use case (the gate returns `apperr.ErrForbidden`).

## Expected outputs

- A per-module, append-only, bitwise, reflection-free permission schema.
- Authorization enforced at the call site via the `access` gate; the use case pure; admin via `registry.Admin()`.
- One marshal format for the DB column and the JWT claim; the registry injected, never global.
