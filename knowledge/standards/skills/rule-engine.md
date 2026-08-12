---
id: rule-engine
name: Rule engine
---

Governs data-driven, stored JSON rule selection: a domain-agnostic `Rule`/`Evaluator` pair,
`And`/`Or` composites, a `RuleTree` with branches, and a `Registry` of type-keyed constructors the
consuming app registers - no `switch`, no reflection. Reach for this when implementing dynamic/stored
rule configs (automation, conditions, branching, segments), registering a new rule type, or auditing
why a rule did or didn't fire. Validation happens at write time against the registry, not lazily at
evaluation. This is not a simple conditional or feature-flag check - it is for rules stored and
evaluated as data.

The rules below are the composed rule-engine rule.
