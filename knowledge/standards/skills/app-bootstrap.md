---
id: app-bootstrap
name: App bootstrap
---

App bootstrap governs the composition root: a thin main(), config loading, injector wiring, and
the per-request principal. Reach for it when writing or editing main()/bootstrap files,
registering a service, repository, or facade in the injector, or choosing singleton vs
per-request-factory lifetime for a new collaborator. It is not where a use case's business rules
live, and not the permission schema itself (see authorization) - it only wires those pieces into
the graph, plus the new-app skeleton when standing up a fresh binary.

The rules below are the composed app-bootstrap rule.
