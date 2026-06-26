---
id: code-minimalism
name: Code minimalism
---

## Mandatory workflow

1. Walk the decision ladder and STOP at the first rung that satisfies the task: (1) necessity - does it need to exist? (
    1) stdlib - does `std` do it (`slices`/`maps`/`cmp`/`errors.Join`/`iter`/`min`/`max`)? (3) a language/platform
       feature? (4) an already-imported dependency? (5) one line - make it one line. (6) only then implement the minimum
       that works.
2. Add NO abstraction/interface/option/generic/layer that is not used by >= 2 real call sites today. No "for the future"
   indirection. (A consumer-declared interface a rule REQUIRES - e.g. a use case `Repository` - is mandated, not
   speculative; it stays.)
3. Add NO third-party dependency when `std` or an existing dep covers it; a new dependency is a design-patterns
   decision.
4. Prefer deletion over addition, boring over clever, the newest fitting stdlib over hand-rolled. Fewest files/functions
   that still honour one-concept-per-file.
5. Document a deliberate corner with a `// simplification:` comment naming the known ceiling and the upgrade path. Never
   cut a corner silently.
6. NEVER trim the hard guardrails: input validation at trust boundaries, error wrapping/handling, security controls,
   required observability/audit.
7. Respect the scope guard: this rule stops at the package boundary. It never justifies skipping a facade, the
   transactional outbox, a multi-write transaction, DI, or a mandated consumer interface, and never merges packages.
   When minimalism and a stop-sign rule disagree, the stop sign wins.
8. Leave one runnable check behind for non-trivial logic; trivial one-liners need none.

## Validation checklist

- [ ] Decision ladder walked; stopped at the first sufficient rung.
- [ ] No speculative abstraction/interface/option/generic with < 2 real call sites.
- [ ] No new dependency that `std` or an existing dep already covers.
- [ ] Newest fitting stdlib used instead of hand-rolled.
- [ ] Deliberate simplifications carry a `// simplification:` ceiling + upgrade comment.
- [ ] Validation / error-handling / security / required observability NOT trimmed.
- [ ] No mandated pattern (facade/outbox/transaction/DI) skipped to "simplify"; no packages collapsed.
- [ ] Non-trivial logic has one runnable check.

## Forbidden actions

- Unrequested abstraction, indirection, or config knobs added "for the future".
- Re-implementing what `std` or an already-imported dependency already does.
- Adding a dependency to save a few lines.
- Cutting validation, error handling, security, or required observability for brevity.
- Using minimalism to justify skipping a facade / outbox / transaction / DI, or merging packages.
- A silent simplification (an intentional limit with no `// simplification:` note).

## Expected outputs

- The minimum code that satisfies the task, reusing std/existing deps first.
- Mandated patterns and guardrails intact; speculative indirection absent.
- Every deliberate shortcut marked with a `// simplification:` ceiling + upgrade path.
