---
id: domain-modeling
name: Domain modeling
description: Actively sharpen the domain model - challenge terms against CONTEXT.md, sharpen fuzzy language, stress-test with concrete scenarios, cross-check code, update CONTEXT.md inline, and offer an ADR only when hard-to-reverse, surprising, and a real trade-off.
tags:
    - process
    - domain
---

# Domain modeling

> Challenge terms against `CONTEXT.md`, sharpen fuzzy or overloaded language into a precise
> canonical term, stress-test it with concrete scenarios, cross-check it against the code, and
> update `CONTEXT.md` the moment a term resolves - not in a batch. Record an ADR only when the
> decision is hard to reverse, surprising, and the result of a real trade-off.

Applies to: any task that changes the domain model, not merely consumes it.

{{define "when"}}
- Pinning down domain terminology or a ubiquitous language.
- A term is overloaded, fuzzy, or conflicts with the glossary.
- Recording an architectural decision as an ADR.
- Another skill needs the domain model kept current.
{{end}}

{{define "must"}}
1. Challenge against the glossary. When a term conflicts with `CONTEXT.md`, call it out at once:
   "Your glossary defines cancellation as X, but you seem to mean Y - which is it?"
2. Sharpen fuzzy language. When a term is vague or overloaded, propose a precise canonical term:
   "You said account - do you mean the Customer or the User? Those are different types."
3. Stress-test with concrete scenarios. Invent edge cases that force precision about the boundaries
   between concepts before any code is written.
4. Cross-reference with code. When someone states how something works, check the code agrees; if it
   does not, surface the contradiction rather than papering over it.
5. Update `CONTEXT.md` inline the moment a term resolves - do not batch. Keep it free of
   implementation detail: it is a glossary, not a spec or a scratch pad.
6. Offer an ADR only when all three hold: hard to reverse, surprising without context, and the
   result of a real trade-off. If any is missing, skip it. Record it via the
   `architecture-decision-records` workflow.
{{end}}

{{define "forbidden"}}
- Treating `CONTEXT.md` as a spec, a scratch pad, or a home for implementation decisions.
- Batching glossary updates instead of capturing them the moment a term resolves.
- Accepting an overloaded term ("account", "user", "cancel") without pinning its canonical meaning.
- Recording an ADR for an easily reversed or self-evident choice.
{{end}}

{{define "validation"}}
- [ ] New or sharpened terms landed in `CONTEXT.md` as they resolved, not in a batch afterward.
- [ ] `CONTEXT.md` holds glossary only - no implementation detail, spec, or decisions.
- [ ] Every term conflict and code contradiction found was surfaced, not silently resolved.
- [ ] An ADR was offered only when hard-to-reverse, surprising, and a real trade-off; recorded
  through the ADR workflow.
{{end}}

{{define "outputs"}}
- Sharpened, canonical domain terms other skills and code can name seams after.
{{end}}
