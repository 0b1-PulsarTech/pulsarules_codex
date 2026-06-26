---
id: template-engine
name: Template engine
---

## Mandatory workflow

1. Treat variables as DATA, not code: build the registry from stored `VariableDef` rows (names are author-chosen data,
   never hardcoded enums or message text in Go).
2. Build the engine from defs + a resolver: `tplate.New().FromDefs(defs, resolver)`. Make `Resolver` lookups lazy;
   resolve by an explicit `Resolver.Resolve(source)`, never by reflection.
3. Render via `text/template` with `missingkey=error`: a missing variable surfaces as a render error (
   `engine.ApplyTemplateOnMessage`), not a silent blank.
4. Resolve hidden/metadata variables separately via `RetrieveHiddenVariables`; never inline them into the visible body.
   Never evaluate callbacks eagerly.
5. For positional external targets, compile named `{{"{{"}}vars}}` to positional `{{"{{"}}1}},{{"{{"}}2}}` with `CompileToPositional`;
   reject conditions/logic the target does not support.
6. Validate at save, render at send. Keep the engine a library; per-target formats (system render vs
   positional/structured external) are per-target adapters.
7. Keep the engine instance per use, not a package-level cache.

## Validation checklist

- [ ] Variables loaded from stored rows; no hardcoded variable enum or message text.
- [ ] Engine built via `New().FromDefs`; resolution lazy; missing var surfaces as a render error.
- [ ] Positional targets reject unsupported conditions/logic.
- [ ] Hidden variables resolved via `RetrieveHiddenVariables`, never inlined.
- [ ] No reflection resolution; no global/package-level engine caches.

## Forbidden actions

- Hardcoding a variable enum or message text in Go.
- Resolving variables by reflection (use an explicit `Resolver.Resolve(source)`).
- Conditions/logic on a target that does not support them.
- Inlining hidden variables into the visible body; eager callback evaluation.
- Global/package-level engine caches.

## Expected outputs

- A variable-registry engine built from stored defs with lazy resolver lookups.
- Renders via `text/template` with `missingkey=error`; hidden variables resolved separately.
- Per-target adapters for positional external formats that reject unsupported logic.
