---
id: dependency-rule
name: Dependency rule
---

Dependency rule governs the vertical direction of source dependencies across any layered
architecture: the domain (entities + use cases) imports nothing from infrastructure, transport, a
persistence driver, or a framework, and a boundary is crossed inward only through a port the inner
layer declares. Reach for it when adding an import to a domain module, deciding where a
framework-touching type belongs, or reviewing a diff for an inward-pointing violation. It is
language-neutral Clean Architecture - the horizontal counterpart is code-placement/module-boundaries
(which top-level directory a file lives in); this rule is about which direction an import may point,
regardless of what language or framework the project is written in.

The rules below are the composed dependency-rule rule.
