---
id: proposal-window
name: Proposal window (bounded numeric value)
description: A Window (min/max/step/currency) that Validate/Clamp/Steps a value; money as minor-unit int64; rigid Proposal/Suggestion structs; versioned windows; append-only sent records renegotiated as revisions.
tags:
    - go
    - engine
composes:
    - template-engine
---

# Proposal window (bounded numeric value)

> A `Window` (min/max/step/currency) built from a stored template row that `Validate`/`Clamp`/`Steps`
> a value; money as minor-unit `int64` (never float); rigid `Proposal`/`Suggestion` structs (never
> re-parsed free text, never auto-applied); versioned windows (template_id+version+supersedes_id,
> edit-in-place only if unreferenced, pinned per record); append-only sent records renegotiated as
> revisions in a `root_id` thread.

Reference tools: a bounded-value engine; a money type (minor-unit int64).

{{define "when"}}
- Validating a bounded numeric value against a template's bounds.
- Clamping an AI-suggested value to policy.
- Generating UI step selectors for numeric values.
- Building a rigid `Proposal`/`Suggestion` AI output.
- Versioning a template and tracking renegotiation as revisions.
{{end}}

{{define "must"}}
1. Build a `Window{Min, Max, Step, Currency}` from a stored template row; gate any proposed value with
   `Window.Validate` (`ErrBelowMin`/`ErrAboveMax`/`ErrOffStep`/`ErrCurrency`, mapped to 422).
2. Clamp AI-suggested values to policy with `Window.Clamp` (snaps to the nearest in-range, step-aligned
   value); generate UI step selectors with `Window.Steps`.
3. Represent money as minor-unit `int64` (`money.FromMinor(1_500_00, money.BRL)`); never float.
4. Produce rigid `Proposal`/`Suggestion` structs (validated before render, never auto-applied, never
   re-parsed from AI free text).
5. Version windows: a logical `template_id` with rows per `version` chained by `supersedes_id`. Edit a
   version in place only while no record references it; once referenced, an edit creates a new
   `version`. A record pins the exact `template_version_id` it was validated against.
6. Keep sent records append-only; renegotiation is a new revision row (`revision_number++`,
   `supersedes_id` -> prior, same `root_id`); only the terminal revision closes.
{{end}}

{{define "recipe"}}
```go
window := window.Window{
    Min: 1_000_00, Max: 50_000_00, Step: 500_00, Currency: money.BRL,
}

if err := window.Validate(proposed); err != nil {
    // ErrBelowMin / ErrAboveMax / ErrOffStep / ErrCurrency -> map to 422
}
clamped := window.Clamp(aiSuggested) // snaps to nearest in-range, step-aligned
steps := window.Steps()              // allowed values for UI selectors
```

Money is minor-unit `int64`, never float:

```go
amount := money.FromMinor(1_500_00, money.BRL)
```

Rigid AI output (validated before render; never auto-applied):

```go
type Proposal struct {
    TemplateID    entities.ID
    ProposedValue money.Money
    ValidUntil    time.Time
    Variables     []tplate.ResolvedVar
}
```

Versioned windows: a logical `template_id` with rows per `version` chained by `supersedes_id`; edit
a version in place only while no record references it; once referenced, an edit creates a new
`version`. A record pins the exact `template_version_id` it was validated against. A sent record is
append-only; renegotiation is a new revision row (`revision_number++`, `supersedes_id` -> prior,
same `root_id`); only the terminal revision closes.
{{end}}

{{define "forbidden"}}
- Money as float (use minor-unit `int64`).
- Accepting a value without `Window.Validate`.
- Re-parsing numbers from AI free text; auto-applying a `Suggestion`.
- Defining default windows at package level (they come from stored rows).
- Editing a sent record in place (renegotiation is a new revision), or a referenced template
  version in place (bump `version`).
{{end}}

{{define "validation"}}
- [ ] `Window` built from a stored template row; value gated by `Validate`; AI drift handled with
  `Clamp`; UI selectors from `Steps`.
- [ ] Money is minor-unit `int64`; never float.
- [ ] AI output is a rigid `Proposal`/`Suggestion`, validated before render, never auto-applied.
- [ ] Template windows versioned; edit-in-place only if unreferenced; record pins
  `template_version_id`.
- [ ] Sent records append-only; renegotiation is a new revision in a `root_id` thread.
{{end}}
