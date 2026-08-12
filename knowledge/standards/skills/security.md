---
id: security
name: Security
---

Governs the security-sensitive edges of a change: secrets loaded only via typed config and zeroed
after load, no secrets or PII in log lines, input validated at the transport boundary, SQL only
through generated queries, JWT verified at middleware, and container images pinned by digest. Reach
for this whenever code touches secrets, writes a log line that could carry PII or a token, validates
boundary input, builds SQL, or handles JWT/identity - a stray `os.Getenv` or an unredacted log line
is in scope even outside auth code.

The rules below are the composed security rule.
