---
id: git-history
name: Git history
---

Git history governs surgically rewriting an already-made history: folding fix-ups into the
commit that created the file, splitting a mixed move-plus-implementation commit, and verifying
the final tree is unchanged. Reach for it when rebuilding a squash or a temp-commit history into
clean, one-logical-change commits, or reordering and rewording a range before review. It is not
for writing the commit you are about to make right now (see commits, which it composes for
subject format), and it is never delegated to a subagent - rebase, fold, and any force-push stay
under direct control in the main session.

The rules below are the composed git-history rule, plus the commits rule it builds on.
