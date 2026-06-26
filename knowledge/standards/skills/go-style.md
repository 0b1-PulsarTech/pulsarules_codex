---
id: go-style
name: Go style
---

## Mandatory workflow

1. Format before finishing: run the project formatter (`gofumpt` + `goimports` + `golines`). Never hand-format.
2. Name with MixedCaps (exported) / mixedCaps (unexported). Packages: lowercase, single word, no underscores, no generic
   names (`util`, `common`, `helpers`).
3. Spell acronyms `ID`, `HTTP`, `URL`, `JWT`, `RPC`, `API` (never `Id`, `Http`, `Url`). Do not stutter (
   `apperr.NewError` -> `apperr.New`).
4. Files are `snake_case.go`; group action verbs (`create_thing.go`, `list_things.go`); colocate `<source>_test.go`. One
   concept per file (~300 lines).
5. Doc-comment every exported symbol (type/func/var/const), starting with its name. Comments explain WHY, not WHAT.
   No `// Package foo` package docstrings. No doc comments on unexported symbols; add a single-line `// why` only when
   non-obvious (hidden constraint, workaround, non-obvious invariant).
6. Imports in three blocks separated by blank lines: standard library, external, this module. Alias proto as
   `<service>v<n>` (`foov1`, `foov1grpc`); otherwise avoid aliases. Never alias the standard library.
7. Accept interfaces, return concrete structs. Declare interfaces in the consumer package with the smallest method set
   needed; keep them small. Name a collaborator port for its ROLE (`Repository`/`Store`/`Resolver`/`Sender`), never a
   generic `Port`/`metaAPIPort`, even when the implementation is an HTTP-API client.
8. Use named structs at package boundaries; `map[string]any` only at the raw I/O edge, translated to a typed value
   immediately.
9. Pick value or pointer receivers and apply consistently per type. Use `any`, never `interface{}`. Design zero values
   to be safe.
10. Use early returns over deep nesting; keep functions under ~80 lines and cyclomatic complexity under ~15.
11. Reach for the newest fitting stdlib (`slices`/`maps`/`cmp`, `errors.Join`, `min`/`max`/`clear`, `for range int`,
    `iter.Seq`) instead of hand-rolled equivalents. Target the module's pinned Go version.
12. Add a compile-time `var _ Iface = (*impl)(nil)` assertion to any file whose type implements an interface.
13. No naked returns in long functions; no shadowing of `err`/`ctx`/predeclared identifiers in inner scopes.
14. Drop explicit generic type params where Go INFERS them: `fuegoport.GET(reg, "/me", h.GetMe)`, not
    `fuegoport.GET[struct{}, OperatorResponse](reg, "/me", h.GetMe)`.
15. No em dash (U+2014) anywhere in source - use a hyphen. No `regexp` for simple string work (prefix/suffix/contains/
    split/trim/case): use `strings` or a small generic `[T ~string]` helper in a `pkg/strutils`-style package.
16. Model an optional value with its ZERO VALUE (`0`, `""`, `false`, a nil slice/map), never a pointer. Reach for
    `*T` only when a caller must distinguish "never set" from "set to the zero value" AND that distinction changes
    behaviour. Optionality alone does not earn a pointer.

## Validation checklist

- [ ] Formatter run; no manual formatting remains.
- [ ] No `snake_case`/`mixedCaps` package names; no `util`/`common`/`helpers`.
- [ ] Acronyms cased correctly; no stuttering type/func names.
- [ ] Every exported symbol has a doc comment beginning with its name; no package docstrings; no doc comments on unexported symbols.
- [ ] Imports are three grouped blocks; proto aliased `<service>v<n>`; std lib unaliased.
- [ ] No `map[string]any` crossing a package boundary; named structs used instead.
- [ ] Interfaces declared consumer-side, small; no fat interface exported from the implementor.
- [ ] Receivers consistent per type; `any` used (not `interface{}`); zero values safe.
- [ ] Optional values carry the zero value, not `*T`, unless "unset" must differ from zero.
- [ ] No naked returns in long funcs; no `err`/`ctx`/predeclared shadowing.
- [ ] Newest fitting stdlib used where it fits (`slices.Equal`, `min`/`max`, `iter.Seq`).
- [ ] Interface-implementing types carry a `var _ Iface = (*impl)(nil)` assertion.
- [ ] Collaborator ports named for their role (no generic `Port`/`metaAPIPort`).
- [ ] Inferable generic type params omitted; no em dash (U+2014); no `regexp` for simple string work.

## Forbidden actions

- Hand-formatting code or bypassing the formatter.
- `snake_case` identifiers; generic `util`/`common`/`helpers` package names; stuttering names.
- `Id`/`Http`/`Url`/`Jwt` casing.
- Returning interface types from public functions without a documented `ireturn` exception.
- `map[string]any` crossing package boundaries; fat interfaces exported from the implementor.
- `interface{}` instead of `any`; `fmt.Println`/`log.Println` in non-test code.
- A generic `Port`/`metaAPIPort` collaborator-port name.
- Spelling out generic type params the compiler can infer; an em dash (U+2014) in source; `regexp` for simple
  string work.
- `*T` for an optional field or parameter whose zero value already means "absent".
- Exported symbols without doc comments; comments that restate the signature; package docstrings; doc comments on unexported symbols.
- Re-implementing what the stdlib already does.

## Expected outputs

- Formatted `.go` files with three-group imports and correct naming.
- Doc comments on all exported symbols; WHY-comments where behaviour is non-obvious.
- Consumer-declared small interfaces; named boundary structs; safe zero values.
