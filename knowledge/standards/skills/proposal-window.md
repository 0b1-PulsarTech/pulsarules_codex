---
id: proposal-window
name: Proposal window
---

Governs a bounded numeric value window - `Validate`/`Clamp`/`Steps` against a stored template's
min/max/step - and the rigid `Proposal`/`Suggestion` structs that carry an AI-suggested value. Reach
for this when validating or clamping a bounded value, generating UI step selectors, or versioning a
template and tracking renegotiation as append-only revisions. Money in the window is always
minor-unit `int64`, never float. Builds on template-engine for rendering but is not that skill:
template-engine renders text, this skill bounds and versions a number.

The rules below are the composed proposal-window rule.
