---
id: skill-authoring
name: Skill authoring
---

Add or change a skill in THIS repository so it renders to every target and passes validation. The
knowledge base under `knowledge/standards/` is the single source of truth; the per-target trees
(`.claude/skills/`, `.opencode/skills/`) are generated mirrors. Author a slim skill body plus
composable `rules`/`patterns` that generate the final skill, wire the router, then validate and
generate. Use whenever you add, edit, or compose a skill, rule, pattern, or workflow.

The procedure - edit the source not the mirror, body plus `skills.yaml` entry, normative clauses in
canonical-section rules/patterns, router wiring, ASCII with trailing newlines, `validate` +
`generate` per target, and an analyzer behind any objective rule - is the composed skill-authoring
rule.
