---
id: minimalism
name: Code minimalism (function scope)
description: The necessity -> stdlib -> platform -> existing-dep -> one-line -> implement decision ladder at function scope; never architecture; never trims validation/error-handling/security/observability or mandated patterns.
tags:
    - go
    - minimalism
dependencies:
    - effective-go
linters:
    - cyclop
    - funlen
    - nestif
    - unused
    - unparam
---

# Code minimalism (function scope)

> Write the minimum that works INSIDE a function/package: a necessity -> stdlib -> platform ->
> existing-dep -> one-line -> implement decision ladder. Scope is function/implementation ONLY -
> never architecture. It never justifies skipping a facade/outbox/transaction/DI or collapsing
> packages, and never cuts validation/error-handling/security/observability.

Applies to: function and method bodies, helper/abstraction/dependency decisions.

{{define "when"}}
- Writing or editing any function/method body.
- About to add a new abstraction, interface, option struct, generic, or helper.
- About to add a third-party dependency.
- Reviewing a diff for bloat / over-engineering / speculative generality.
{{end}}

{{define "must"}}
1. Walk the decision ladder and STOP at the first rung that satisfies the task: (1) necessity - does
   it need to exist at all? (2) stdlib - does `std` do it (`slices`/`maps`/`cmp`/`errors.Join`/`iter`/
   `min`/`max`)? (3) language/platform feature? (4) an already-imported dependency? (5) one line -
   make it one line. (6) only then implement the minimum that works.
2. Add NO abstraction/interface/option/generic/layer that isn't used by >= 2 real call sites today.
   No "for the future" indirection. (A consumer-declared interface a rule REQUIRES - e.g. a use case
   `Repository` - is mandated, not speculative; it stays.)
3. Add NO third-party dependency when `std` or an existing dep covers it; a new dependency is a
   design-patterns decision.
4. Prefer deletion over addition, boring over clever, the newest fitting stdlib over hand-rolled.
   Fewest files/functions that still honour one-concept-per-file.
5. Document a deliberate corner with a `// simplification:` comment naming the known ceiling +
   upgrade path. Never cut a corner silently.
6. NEVER trim the hard guardrails: input validation at trust boundaries, error wrapping/handling (no
   data loss), security controls, required observability/audit.
7. SCOPE GUARD: this rule stops at the package boundary. It never justifies skipping a facade, the
   transactional outbox, a multi-write transaction, DI, or a mandated consumer interface, and never
   merging packages. When minimalism and a stop-sign rule disagree, the stop sign wins.
8. Non-trivial logic leaves one runnable check behind (smallest failing test); trivial one-liners
   need none.
9. Delete dead code rather than keeping it: no unused functions, variables, parameters (`unused`,
   `unparam`), and no commented-out code blocks - version control remembers the old version, so a
   commented-out block is just noise. Never `//nolint:unused` to keep a symbol nothing calls.
{{end}}

{{define "forbidden"}}
- Unrequested abstraction, indirection, or config knobs added "for the future".
- Re-implementing what `std` or an already-imported dependency already does.
- Adding a dependency to save a few lines.
- Cutting validation, error handling, security, or required observability for brevity.
- Using minimalism to justify skipping a facade / outbox / transaction / DI, or merging packages.
- A silent simplification (an intentional limit with no `// simplification:` note).
- Dead code: unused symbols/parameters or commented-out blocks left in the tree; `//nolint:unused`
  to retain an uncalled symbol.
{{end}}

{{define "validation"}}
- [ ] Decision ladder walked; stopped at the first sufficient rung.
- [ ] No speculative abstraction/interface/option/generic with < 2 real call sites.
- [ ] No new dependency that `std` or an existing dep already covers.
- [ ] Newest fitting stdlib used instead of hand-rolled.
- [ ] Deliberate simplifications carry a `// simplification:` ceiling + upgrade comment.
- [ ] Validation / error-handling / security / required observability NOT trimmed.
- [ ] No mandated pattern (facade/outbox/transaction/DI) skipped to "simplify"; no packages collapsed.
- [ ] Non-trivial logic has one runnable check.
- [ ] No dead code: no unused symbols/parameters (`unused`/`unparam`), no commented-out blocks.
{{end}}
