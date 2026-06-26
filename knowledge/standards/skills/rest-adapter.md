---
id: rest-adapter
name: REST adapter
---

Governs exposing a use case over REST: typed `Input`/`Output` DTOs that drive OpenAPI, a handler
that only parses, calls the use case, maps, and responds, and routes registered through a router
contract with a permission guard. Reach for this when adding or editing an HTTP handler or its DTOs.
It is not usecase-layout - the use case underneath stays transport-agnostic; rest-adapter is only the
thin shell around it. Pairs with transport-interop for the domain-imports-no-net/http rule and with
authorization for the permission guard.

The rules below are the composed rest-adapter rule.
