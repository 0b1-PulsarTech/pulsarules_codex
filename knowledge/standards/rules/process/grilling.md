---
id: grilling
name: Grilling
description: Stress-test a plan or decision one question at a time - always recommend an answer, research facts via tools instead of asking, walk the decision tree in dependency order, and do not build before a shared understanding is confirmed.
tags:
    - process
    - planning
---

# Grilling

> Interview the user relentlessly about a plan, decision, or idea until you reach a shared
> understanding: one question at a time, each with a recommended answer, facts looked up rather than
> asked, decisions resolved in dependency order, and no building before the user confirms.

Applies to: planning or stress-testing a decision before implementation.

{{define "when"}}
- Stress-testing a plan, decision, or design before building anything non-trivial.
- The user uses a "grill" trigger phrase.
- A task has open decisions with dependencies between them.
{{end}}

{{define "must"}}
1. Ask ONE question at a time and wait for the answer before the next. Several at once collapses the
   decision tree and bewilders.
2. For every question give your RECOMMENDED answer and a one-line reason. Never present an open
   question with no default.
3. Separate facts from decisions: if a fact can be found by exploring the environment (filesystem,
   git history, tools, running code), look it up - do not ask. Only genuine decisions go to the user.
4. Walk the tree in dependency order - resolve the decisions that unblock others first, so later
   answers do not invalidate earlier ones.
5. Do not act, write code, or finalize a plan until the user confirms a shared understanding.
{{end}}

{{define "forbidden"}}
- Asking multiple questions in one turn.
- Asking the user a fact a tool could have answered.
- Presenting a question with no recommendation.
- Starting to build before shared understanding is explicitly confirmed.
{{end}}

{{define "validation"}}
- [ ] Questions asked one at a time, each with a recommended answer.
- [ ] Every fact was resolved with a tool, not asked.
- [ ] Decisions resolved in dependency order down the tree.
- [ ] No implementation or final plan before the user confirmed shared understanding.
{{end}}
