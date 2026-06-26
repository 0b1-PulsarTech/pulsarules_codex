---
id: errors-logging
name: Errors & logging
---

Errors and logging governs two disciplines together: wrapping and comparing errors (%w,
sentinels, the domain-error contract) and structured slog logging with typed attributes and no
PII. Reach for it whenever you return, wrap, or map an error, define a sentinel, create a domain
error that maps to a status code, or add any log statement. It is not distributed tracing or
span instrumentation (see observability) - logging here is a single slog line per error at the
top of the call chain, log or return, never both.

The rules below are the composed errors-logging rule.
