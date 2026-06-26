---
id: authorization
name: Authorization
---

Authorization governs how a module declares and enforces permissions: a bitwise, append-only
schema per module, checked at the call site via the access gate, never an ad-hoc isAdmin check
inside a use case. Reach for it when protecting an endpoint or use case with a permission,
declaring a new module's schema, granting or revoking access, or marshalling permissions for the
DB column and the JWT claim. It is not the principal-resolution wiring itself (see app-bootstrap)
- it consumes the resolved principal only to decide what that principal may do.

The rules below are the composed authorization rule.
