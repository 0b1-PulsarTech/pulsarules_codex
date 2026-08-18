---
id: template-engine
name: Template engine
---

Governs rendering text from a variable registry: variables are DATA built from stored `VariableDef`
rows, never hardcoded Go enums; lookups are lazy through a `Resolver`; rendering goes through
`text/template` with `missingkey=error`, so a missing variable is a render error, not a silent
blank. Reach for this when building user-defined template authoring/rendering, or compiling a
template to a positional external format (for example, numbered HSM slots). Hidden/metadata
variables resolve separately and never inline into the visible body. proposal-window builds on this
for bounded numeric values but is a separate skill.

The rules below are the composed template-engine rule.
