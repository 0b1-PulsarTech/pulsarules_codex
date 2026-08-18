---
id: commits
name: Commits
---

Commits governs the shape of a single commit: an emoji-prefixed Conventional Commit subject, one
logical change per commit, and no tool-attribution trailers. Reach for it whenever you are about
to write a commit message, including for a breaking change or an AI-assisted change. It is not
for rebuilding a squash or an already-tangled history into clean commits - that is git-history,
which composes this rule for its subject format. The router loads it last, alongside
verification-before-done, as the definition-of-done gate before you commit.

The rules below are the composed commits rule.
