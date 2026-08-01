---
id: domain-modeling
name: Domain modeling
---

Actively build and sharpen the project's domain model as you design: challenge terms, invent
edge-case scenarios, and write the glossary and decisions down the moment they crystallize.
Merely reading a glossary for vocabulary is not this skill - any skill can do that. This skill is
for when you are changing the model, not just consuming it. It pairs with the
`architecture-decision-records` workflow for the decisions worth recording.

## File structure

Most repositories have a single context:

```
/
  CONTEXT.md            <- the glossary, and nothing else
  docs/adr/             <- architectural decision records
  ...
```

If a `CONTEXT-MAP.md` exists at the root, the repository has multiple contexts and the map points
to where each `CONTEXT.md` lives (for example per module under `internal/<module>/`). Create these
files lazily - only when you have the first term or the first decision to write.
