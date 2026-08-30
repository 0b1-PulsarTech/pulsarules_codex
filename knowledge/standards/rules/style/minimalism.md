---
id: minimalism
name: Code minimalism (function scope)
description: The necessity -> stdlib -> platform -> existing-dep -> one-line -> implement decision ladder at function scope; never architecture; never trims validation/error-handling/security/observability or mandated patterns. Language-neutral; see minimalism-go for the Go tooling instantiation.
tags:
    - minimalism
analyzers:
    - simplification-path
---

# Code minimalism (function scope)

> Write the minimum that works inside a function/package: a necessity -> stdlib -> platform ->
> existing-dep -> one-line -> implement decision ladder. Scope is function/implementation only -
> never architecture. It never justifies skipping a facade/outbox/transaction/DI or collapsing
> packages, and never cuts validation/error-handling/security/observability.

Applies to: function and method bodies, helper/abstraction/dependency decisions, in any language.
See `minimalism-go` for the Go-concrete tooling (gopls, golangci-lint) that instantiates it.

{{define "when"}}
- Writing or editing any function/method body.
- About to add a new abstraction, interface, option struct, generic, or helper.
- About to add a third-party dependency.
- Reviewing a diff for bloat / over-engineering / speculative generality.
{{end}}

{{define "must"}}
1. Understand the problem before choosing a fix's shape: trace the failing path and name the cause.
   The decision ladder below governs how small a fix is, never whether you diagnosed one first.
2. Walk the decision ladder and stop at the first rung that satisfies the task: (1) necessity - does
   it need to exist at all? (2) stdlib - does the platform standard library do it? (3) language/
   platform feature? (4) reuse what this codebase already has - an existing helper in this package,
   or one already reachable through a facade port this package consumes (see [[module-boundaries]],
   [[interop]]) - never a new deep import across a module boundary just to reach it, that is a
   facade decision, not a minimalism one. (5) an already-imported dependency? (6) one line - make it
   one line. (7) only then implement the minimum that works.
3. A bug fix targets the root cause, not the symptom: before patching a shared symbol, find every
   caller (a references search, cross-reference tool, or IDE find-usages), then land the fix once at
   the shared definition instead of at each call site. Where the fix needs a guard, add an early
   return of an error, never a `bool`/status parameter the caller branches on (see
   [[flag-arguments]], [[command-query-separation]]); a smaller diff is not the correctness argument,
   a test that fails against the old code is (see [[verification]]).
4. Add no abstraction/interface/option/generic/layer that isn't used by >= 2 real call sites today.
   No "for the future" indirection. (A consumer-declared interface a rule requires - e.g. a use case
   `Repository` - is mandated, not speculative; it stays.)
5. Add no third-party dependency when the platform standard library or an existing dep covers it; a
   new dependency is a design-patterns decision.
6. Prefer deletion over addition, boring over clever, the newest fitting stdlib/platform feature over
   hand-rolled. Fewest files/functions that still honour one-concept-per-file.
7. Document a deliberate corner with a `simplification:` marker comment naming the known ceiling +
   upgrade path. Never cut a corner silently.
8. NEVER trim the hard guardrails: input validation at trust boundaries, error wrapping/handling (no
   data loss), security controls, required observability/audit.
9. Scope guard: this rule stops at the package boundary. It never justifies skipping a facade, the
   transactional outbox, a multi-write transaction, DI, or a mandated consumer interface, and never
   merging packages. When minimalism and a stop-sign rule disagree, the stop sign wins.
10. Non-trivial logic leaves one runnable check behind (smallest failing test); trivial one-liners
    need none.
11. Delete dead code rather than keeping it: no unused functions, variables, or parameters, and no
    commented-out code blocks - version control remembers the old version, so a commented-out block
    is just noise. Never silence a linter/analyzer finding just to keep a symbol nothing calls.
{{end}}

{{define "forbidden"}}
- Unrequested abstraction, indirection, or config knobs added "for the future".
- Re-implementing what the platform standard library or an already-imported dependency already does.
- Adding a dependency to save a few lines.
- Cutting validation, error handling, security, or required observability for brevity.
- Using minimalism to justify skipping a facade / outbox / transaction / DI, or merging packages.
- A silent simplification (an intentional limit with no `simplification:` marker).
- Dead code: unused symbols/parameters or commented-out blocks left in the tree; a suppressed
  linter/analyzer finding used to retain an uncalled symbol.
{{end}}

{{define "validation"}}
- [ ] The problem was understood (cause traced) before the decision ladder picked a fix's shape.
- [ ] Decision ladder walked; stopped at the first sufficient rung.
- [ ] Existing in-package or facade-reachable code was reused before writing new code; no new deep
  import across a module boundary to reach it.
- [ ] A shared-symbol bug fix found every caller via a references search and landed once at the root;
  any added guard is an early-return error, not a bool/status flag.
- [ ] No speculative abstraction/interface/option/generic with < 2 real call sites.
- [ ] No new dependency that the standard library or an existing dep already covers.
- [ ] Newest fitting stdlib/platform feature used instead of hand-rolled.
- [ ] Deliberate simplifications carry a `simplification:` ceiling + upgrade comment.
- [ ] Validation / error-handling / security / required observability not trimmed.
- [ ] No mandated pattern (facade/outbox/transaction/DI) skipped to "simplify"; no packages collapsed.
- [ ] Non-trivial logic has one runnable check.
- [ ] No dead code: no unused symbols/parameters, no commented-out blocks.
{{end}}
