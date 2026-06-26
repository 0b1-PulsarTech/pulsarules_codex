---
id: proposal-window
name: Proposal window
---

## Mandatory workflow

1. Build a `Window{Min, Max, Step, Currency}` from a stored template row; gate any proposed value with
   `Window.Validate` (`ErrBelowMin`/`ErrAboveMax`/`ErrOffStep`/`ErrCurrency`, mapped to 422).
2. Clamp AI-suggested values to policy with `Window.Clamp` (snaps to the nearest in-range, step-aligned value); generate
   UI step selectors with `Window.Steps`.
3. Represent money as minor-unit `int64` (`money.FromMinor(1_500_00, money.BRL)`); never float.
4. Produce rigid `Proposal`/`Suggestion` structs (validated before render, never auto-applied, never re-parsed from AI
   free text).
5. Version windows: a logical `template_id` with rows per `version` chained by `supersedes_id`. Edit a version in place
   only while no record references it; once referenced, an edit creates a new `version`. A record pins the exact
   `template_version_id` it was validated against.
6. Keep sent records append-only; renegotiation is a new revision row (`revision_number++`, `supersedes_id` -> prior,
   same `root_id`); only the terminal revision closes.

## Validation checklist

- [ ] `Window` built from a stored template row; value gated by `Validate`; AI drift handled with `Clamp`; UI selectors
  from `Steps`.
- [ ] Money is minor-unit `int64`; never float.
- [ ] AI output is a rigid `Proposal`/`Suggestion`, validated before render, never auto-applied.
- [ ] Template windows versioned; edit-in-place only if unreferenced; record pins `template_version_id`.
- [ ] Sent records append-only; renegotiation is a new revision in a `root_id` thread.

## Forbidden actions

- Money as float (use minor-unit `int64`).
- Accepting a value without `Window.Validate`.
- Re-parsing numbers from AI free text; auto-applying a `Suggestion`.
- Defining default windows at package level (they come from stored rows).
- Editing a sent record in place (renegotiation is a new revision), or a referenced template version in place (bump
  `version`).

## Expected outputs

- A `Window` from a stored template that validates/clamps/steps a value; money as minor-unit `int64`.
- Rigid AI `Proposal`/`Suggestion` output, validated before render, never auto-applied.
- Versioned, append-only records; renegotiation tracked as revisions in a `root_id` thread.
