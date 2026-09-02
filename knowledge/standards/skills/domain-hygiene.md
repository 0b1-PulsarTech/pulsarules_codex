---
id: domain-hygiene
name: Domain hygiene
---

Domain hygiene governs two language-neutral disciplines together: command-query separation (a
command mutates state and returns only an error outcome; a query is side-effect-free and returns a
value, never blended) and logging hygiene (a structured, typed-attribute logger obtained by
injection rather than a library-global, one log per error at the top of the call chain, never
secrets or PII). Reach for it when deciding what a state-changing method returns or when adding any
log statement. It is not the Go-specific idiom for either - the concrete `error`/`errors.Is`/`errors.As`
mechanics and the `log/slog` API live in errors-logging, which composes with this skill in a Go
project.

The rules below are the composed domain-hygiene rule.
