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

## Mandatory workflow

1. Challenge against the glossary. When a term conflicts with `CONTEXT.md`, call it out at once:
   "Your glossary defines cancellation as X, but you seem to mean Y - which is it?"
2. Sharpen fuzzy language. When a term is vague or overloaded, propose a precise canonical term:
   "You said account - do you mean the Customer or the User? Those are different types."
3. Stress-test with concrete scenarios. Invent edge cases that force precision about the
   boundaries between concepts before any code is written.
4. Cross-reference with code. When someone states how something works, check the code agrees; if
   it does not, surface the contradiction rather than papering over it.
5. Update `CONTEXT.md` inline the moment a term resolves - do not batch. Keep it free of
   implementation detail: it is a glossary, not a spec or a scratch pad.
6. Offer an ADR only when all three hold: hard to reverse, surprising without context, and the
   result of a real trade-off. If any is missing, skip it. Record it via the
   `architecture-decision-records` workflow.

## Validation checklist

- [ ] New or sharpened terms landed in `CONTEXT.md` as they resolved, not in a batch afterward.
- [ ] `CONTEXT.md` holds glossary only - no implementation detail, spec, or decisions.
- [ ] Every term conflict and code contradiction found was surfaced, not silently resolved.
- [ ] An ADR was offered only when hard-to-reverse, surprising, and a real trade-off; recorded
      through the ADR workflow.

## Forbidden actions

- Treating `CONTEXT.md` as a spec, a scratch pad, or a home for implementation decisions.
- Batching glossary updates instead of capturing them the moment a term resolves.
- Accepting an overloaded term ("account", "user", "cancel") without pinning its canonical meaning.
- Recording an ADR for an easily reversed or self-evident choice.

## Expected outputs

- A `CONTEXT.md` glossary that matches how the code actually behaves.
- Sharpened, canonical domain terms other skills and code can name seams after.
- ADRs only for decisions a future reader would otherwise wonder about.
