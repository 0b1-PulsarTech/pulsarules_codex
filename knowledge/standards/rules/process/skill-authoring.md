---
id: skill-authoring
name: Skill authoring
description: Add or change a skill by editing the knowledge source (never the generated mirrors) - a body plus a skills.yaml entry, normative clauses composed from canonical rules/patterns, router wiring, validate and generate for every target, and an analyzer behind any objective rule.
tags:
    - process
    - meta
---

# Skill authoring

> The knowledge base under `knowledge/standards/` is the single source of truth; the per-target trees
> are generated mirrors. Author skills as a slim body plus composable rules/patterns that generate the
> final skill, wire the router, validate and generate for every target, and back objective rules with
> an analyzer.

Applies to: adding or editing any skill, rule, pattern, or workflow in this repository.

{{define "when"}}
- Adding or editing a skill, rule, pattern, or workflow.
- Composing shared clauses into a skill.
- Wiring a new skill into the router.
- Making a skill rule machine-enforced.
{{end}}

{{define "must"}}
1. Edit the SOURCE under `knowledge/standards/`, never a generated mirror (`.claude/skills/**`,
   `.opencode/skills/**`) - those are regenerated and hand edits are lost.
2. A skill is a body plus a registry entry. Body: `skills/<id>.md` with `id`/`name` frontmatter, no
   H1 (the template supplies it), a slim orientation. Registry: a `skills.yaml` entry with
   `description`, `triggers`, `always_load`, `order`.
3. Put the normative clauses in COMPOSABLE pieces - `rules/**` and `patterns/**` - and compose them,
   so rules/patterns generate the skill. A rule/pattern body may only define the canonical section
   keys (`when`, `catalog`, `must`, `recipe`, `approved`, `rejected`, `examples`, `forbidden`,
   `validation`); any other key fails `clauselint`.
4. Wire `router.yaml` (baseline, a dispatch row, and/or an order step) when the skill dispatches on a
   task signal; an unrouted skill renders but never auto-loads.
5. Keep the writing constraints: ASCII only, hyphen `-` never em dash (U+2014), one concept per
   clause, every Markdown file ends with a trailing newline.
6. Validate and generate: run `pulsarules_cli validate` (integrity + clauselint) and `generate`, then
   spot-check the rendered skill and its mirror for every target.
7. Back any objective rule with an analyzer under `internal/analyzer/**`, registered in
   `internal/analysis/boot.go` and gated in `internal/config` - a rule with no analyzer is a
   suggestion, not a guardrail.
{{end}}

{{define "forbidden"}}
- Hand-editing a generated mirror (`.claude/skills/**`, `.opencode/skills/**`).
- A rule/pattern body that defines a non-canonical section key.
- An em dash (U+2014) or non-ASCII punctuation in any skill/rule/pattern source.
- Shipping an "enforced" rule with no analyzer wired into the governance pipeline.
{{end}}

{{define "validation"}}
- [ ] Only `knowledge/standards/**` was edited; no generated mirror was touched.
- [ ] Body has `id`/`name` frontmatter, no H1; a `skills.yaml` entry exists.
- [ ] Normative clauses live in composed rules/patterns using only canonical section keys.
- [ ] Router wired if the skill dispatches; `validate` and `generate` pass; mirrors spot-checked.
- [ ] Any objective rule is backed by a registered analyzer.
{{end}}
