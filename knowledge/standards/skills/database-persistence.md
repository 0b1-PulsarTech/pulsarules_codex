---
id: database-persistence
name: Database persistence
---

Database persistence governs the full chain from schema to repository: an ent schema,
Atlas-generated migrations, sqlc queries, and a repository that converts generated rows to
domain DTOs via goverter before returning. Reach for it when adding or modifying a database
entity, writing a .sql query, generating migrations, creating or editing a repository, or
setting up repository tests. It is not where multi-write atomicity is decided (see transactions)
or where a side effect leaving the write is published (see eventing-outbox) - a repository under
this skill does one write and returns domain types, nothing generated escapes it.

The rules below are the composed database-persistence rule.
