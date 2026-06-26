---
id: security
name: Security
---

## Mandatory workflow

1. Load secrets from env into a typed `Config` via the config loader (e.g. `confloader`). Never `os.Getenv` outside the
   config package. Never commit secrets; `.env` is local-only and gitignored.
2. Zero the secret source field after registering it so cleartext does not linger in a long-lived struct. Treat `Config`
   as read-only after boot.
3. Validate all network inputs at the boundary (HTTP body/query/path, gRPC request, webhook payload) using typed
   binding/validation before calling the use case. Use cases assume valid input and enforce business invariants,
   returning a domain invalid/conflict error.
4. Never log Authorization headers, JWTs, raw request/response bodies, or customer PII. IDs are fine; log a redacted
   summary (`slog.Int("body_size", n)`) when unsure.
5. Run all DB access through generated queries (`sqlc`); never string-concatenate SQL (gosec G201). Migrations with
   `DROP`/irreversible `ALTER` need explicit reviewer sign-off.
6. JWT/identity: keys generated once by a config-gen tool; middleware verifies the signed identity and stores claims on
   context; use cases consume the principal, never re-parse a token. Token expiration is mandatory.
7. Pin container base images by digest (no `:latest`); pin dependencies (`go mod tidy`).
8. Obey gosec; justify any suppression inline: `// #nosec G<n>: <reason>`.

## Validation checklist

- [ ] Secrets loaded via the config loader only; no stray `os.Getenv`; source fields zeroed.
- [ ] No secrets/tokens/PII/raw bodies in any log line.
- [ ] All inputs validated at the transport boundary before the use case.
- [ ] SQL only via generated queries; zero string-built queries (gosec G201 clean).
- [ ] JWT verified at middleware; use cases read the principal, not raw tokens; expiry enforced.
- [ ] Container base images pinned by digest; deps pinned in `go.mod`.

## Forbidden actions

- Committing secrets; `os.Getenv` outside the config package.
- Logging secrets, Authorization headers, JWTs, raw bodies, or PII.
- Validating input inside the use case (boundary responsibility) or passing unvalidated input through.
- String-concatenated SQL; hand-rolled JSON bypassing proto contracts.
- `:latest` or unpinned base images; mutating `Config` after boot.
- Re-parsing a JWT inside a use case.

## Expected outputs

- Secrets confined to a typed, zeroed, read-only `Config`; no stray `os.Getenv`.
- All inputs validated at the boundary; SQL only via generated queries.
- JWT verified at middleware; containers pinned by digest.
