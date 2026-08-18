---
id: code-minimalism
name: Code minimalism
---

Code minimalism governs the smallest correct implementation at function scope: walk the
necessity -> stdlib -> platform -> existing-dep -> one-line -> implement ladder before writing
new code. Reach for it whenever you are about to add an abstraction, interface, option, generic,
or helper, add a third-party dependency, or review a diff for bloat. It never applies at
architecture scope, and it never justifies skipping a mandated pattern or guardrail - a facade,
the outbox, a transaction, DI, input validation - when minimalism and a stop-sign rule disagree,
the stop sign wins.

The rules below are the composed code-minimalism rule.
