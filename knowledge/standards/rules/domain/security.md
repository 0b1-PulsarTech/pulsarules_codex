---
id: security
name: Security
description: Secrets via typed config (zeroed after load), no secrets/PII in logs, validate inputs at the boundary, SQL via generated queries, JWT verified at middleware, images pinned by digest.
tags:
    - go
    - security
dependencies:
    - config
    - database
linters:
    - gosec
analyzers:
    - golangci-lint
    - pprof-labels
---

# Security

> Secrets only via typed config (zeroed after load), no secrets/PII in logs, validate all inputs at
> the transport boundary, SQL only through generated queries (no string concat), JWT/identity
> verified at middleware, base images pinned by digest.

Applies to: any code touching secrets, input, SQL, auth, or containers.

{{define "when"}}
- Loading secrets or credentials.
- Writing a log line that could touch PII or tokens.
- Attaching `runtime/pprof` goroutine labels for profiling or tracing correlation.
- Validating user/network input at a boundary.
- Constructing or executing SQL.
- Working with JWT/identity verification.
- Editing a Dockerfile/Containerfile or pinning dependencies.
{{end}}

{{define "must"}}
1. Load secrets from env into a typed `Config` via a config loader (source repos use
   `confloader`/`confloader`). Never `os.Getenv` outside the config package. Never commit secrets;
   `.env` is local-only and gitignored.
2. Zero the secret source field after registering it so cleartext does not linger in a long-lived
   struct. Treat `Config` as read-only after boot.
3. Validate all network inputs at the boundary (HTTP body/query/path, gRPC request, webhook
   payload) using typed binding/validation before calling the use case. Use cases assume valid input
   and enforce business invariants, returning a domain invalid/conflict error.
4. Never log Authorization headers, JWTs, raw request/response bodies, or customer PII. IDs are
   fine; log a redacted summary (`slog.Int("body_size", n)`) when unsure.
5. All DB access goes through generated queries (sqlc); never string-concatenate SQL (gosec G201).
   Migrations with `DROP`/irreversible `ALTER` need explicit reviewer sign-off.
6. JWT/identity: keys generated once by a config-gen tool; middleware verifies the signed identity
   and stores claims on context; use cases consume the principal, never re-parse a token. Token
   expiration is mandatory.
7. Pin container base images by digest (no `:latest`); pin dependencies (`go mod tidy`).
8. Obey gosec; justify any suppression inline: `// #nosec G<n>: <reason>`.
9. Never put PII or a secret in a `runtime/pprof` goroutine label. A module declaring `go 1.27`
   prints a goroutine's pprof labels in its traceback header
   (`goroutine 1 [running] {user_id: "..."}:`), so a value attached purely for profiling
   correlation becomes cleartext in the crash dump that `recover()` middleware, a supervisor, or a
   crash reporter forwards straight into logs. It leaks through two paths:
   `pprof.SetGoroutineLabels` with no restore leaks on its own goroutine's panic, and
   `pprof.Do` - whose deferred restore does protect its own goroutine - still leaks under
   `GOTRACEBACK=all` when a different goroutine panics while the labeled one is alive. Label with an
   opaque correlation id (trace/span id) instead of a user id, email, or token. Do not treat
   `GODEBUG=tracebacklabels=0` as the fix: it is a deployment flag the consumer can lose, not a
   property of the code.
{{end}}

{{define "forbidden"}}
- Committing secrets; `os.Getenv` outside the config package.
- Logging secrets, Authorization headers, JWTs, raw bodies, or PII.
- Validating input inside the use case (boundary responsibility) or passing unvalidated input
  through.
- String-concatenated SQL; hand-rolled JSON bypassing proto contracts.
- `:latest` or unpinned base images; mutating `Config` after boot.
- A user id, email, token, or any PII as a `runtime/pprof` goroutine label value - Go 1.27 prints
  labels in panic traceback headers.
- Re-parsing a JWT inside a use case.
{{end}}

{{define "validation"}}
- [ ] Secrets loaded via the config loader only; no stray `os.Getenv`; source fields zeroed.
- [ ] No secrets/tokens/PII/raw bodies in any log line.
- [ ] All inputs validated at the transport boundary before the use case.
- [ ] SQL only via generated queries; zero string-built queries (gosec G201 clean).
- [ ] JWT verified at middleware; use cases read the principal, not raw tokens; expiry enforced.
- [ ] Container base images pinned by digest; deps pinned in `go.mod`.
- [ ] No `runtime/pprof` goroutine label carries PII or a secret; correlation labels are opaque ids.
{{end}}
