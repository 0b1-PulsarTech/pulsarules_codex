---
id: grilling
name: Grilling
---

Interview the user relentlessly about a plan, decision, or idea until you reach a shared
understanding, before building anything non-trivial. Walk each branch of the decision tree, resolving
the dependencies between decisions one at a time. Pair with `improve-codebase-architecture` (which
runs this loop on a chosen refactor) and `domain-modeling` (to capture terms as they crystallize).

The discipline - one question at a time, each with a recommendation, facts researched not asked,
decisions in dependency order, nothing built before confirmation - is the composed grilling rule.

## Mandatory workflow

1. Ask ONE question at a time and wait for the answer; several at once collapses the decision tree.
2. Give your RECOMMENDED answer and a one-line reason with every question. Never present an open
   question with no default.
3. Look facts up instead of asking: if the filesystem, git history, or a running tool can answer it,
   it is not a question for the user. Only genuine decisions go to them.
4. Walk the tree in dependency order - resolve what unblocks the rest first, so later answers cannot
   invalidate earlier ones.
5. Do not act, write code, or finalize a plan until the user confirms a shared understanding.

## Validation checklist

- [ ] Questions asked one at a time, each with a recommended answer.
- [ ] Every fact was resolved with a tool, not asked.
- [ ] Decisions resolved in dependency order down the tree.
- [ ] No implementation or final plan before the user confirmed shared understanding.

## Forbidden actions

- Asking multiple questions in one turn.
- Asking the user a fact a tool could have answered.
- Presenting a question with no recommendation.
- Starting to build before shared understanding is explicitly confirmed.

## Expected outputs

- A confirmed, written-down decision per branch of the tree, reached one question at a time.
- No implementation before that confirmation.
