---
id: git-history
name: Git history
---

## Mandatory workflow

1. SAFETY FIRST. Work on a fresh branch. NEVER rewrite or force-push a pushed/shared branch (e.g.
   `main`) without explicit human consent - integration is the human's call. Before ANY rewrite create a
   backup branch (`backup/<name>`); keep it until integration is confirmed.
2. DECOMPOSE, don't re-package. As you split a squash, run the matched skills (start with
   `project-router`) over the implementation and apply the genuine improvements you detect (bugs, missing
   tests, idiom drift). The rewrite is a review, not a repackage.
3. One logical change per commit; subjects follow [[commits]] (emoji-prefixed Conventional Commits,
   varied gitmoji, no tool-attribution trailers).
4. INTRODUCE EVERY FILE IN ITS FINAL FORM. Any later change that only brings a pre-existing file to the
   shape it should have had at creation - hygiene (blank lines, package-doc removal), DI inversion,
   `RegisterSingleton`->`RegisterConstructor`, generic type-param removal, an added port/method, a
   conformance refactor (e.g. `regexp`->`strutils`), a serialization/tag fix, AND config edits
   (`go.mod`/`go.work`/`Taskfile`) - FOLDS into that file's creation commit; drop the standalone fix-up
   commit.
5. Genuinely-new functionality stays its own commit (folding it would be anachronistic - it depends on
   things that do not exist yet). A MIXED commit is hunk-split: the fix-up part folds, the new part stays.
6. A move/rename (`:truck:`) commit is PURE RENAMES ONLY. Never mix a relocation with new implementation;
   split the new code into separate `feat` commits placed after the move.
7. THE FINAL-TREE INVARIANT (the safety proof). After every rewrite, `git diff <pre-rewrite-snapshot>
   HEAD` is EMPTY - the rewrite only moved commit boundaries, no content changed - OR it equals EXACTLY
   the intended new edits and nothing else. An unexpected non-empty diff means content was lost or
   misplaced; fix it before moving on.
8. MECHANISM. Snapshot the desired final tree as a ref (e.g. `foldsrc`). Run `git rebase -i --root` with
   a scripted todo (`GIT_SEQUENCE_EDITOR="cp todo.txt"`): `drop` each fix-up commit and `exec git checkout
   <foldsrc> -- <files> && git commit --amend --no-edit` at each file's creation commit. Generate the
   file->creation map with `git log --diff-filter=A --reverse -- <file>`; use `git show --no-renames
   --name-status` so moves classify as add+delete (checkout the new path, then `git rm -q
   --ignore-unmatch` the old). Resolve any conflict by checking out the `foldsrc` version. A commit
   emptied by folding is dropped.
9. Per-commit build is RELAXED: intermediate commits need not compile (a port may reference a type added
   in a later commit). The FINAL tree plus a full build+lint+test sweep is the gate, not each commit.
10. Reword for emoji variety across the range so no single gitmoji dominates (see [[commits]]).

## Validation checklist

- [ ] Backup branch created; the pushed/shared branch untouched (no force-push without consent).
- [ ] `git diff <pre-rewrite-snapshot> HEAD` empty (pure reorg) OR exactly the intended edits.
- [ ] Every fix-up folded into its file's creation commit; no standalone "bring to lint bar / invert DI /
  add port / clean config" commit remains.
- [ ] Each `:truck:` move shows only renames (`git show --name-status`); new code is separate commits.
- [ ] Full sweep green on the final HEAD (build + lint + tests).
- [ ] Subjects emoji-prefixed Conventional Commits, varied emoji, no tool-attribution trailers.

## Forbidden actions

- Rewriting or force-pushing a pushed/shared branch (`main`) without explicit human consent.
- A rebase that changes the final tree unintentionally (always verify the invariant).
- A standalone commit that only "fixes up" a file to the form it should have had at creation.
- A `:truck:`/move commit that also adds new implementation.
- Delegating any part of a history rewrite to a subagent (see `[[agent-orchestration]]`) - rebase,
  fold, reorder, and force-push stay in the main session under direct control.
- Deleting backup refs before integration is confirmed.

## Expected outputs

- A clean incremental history: one logical change per commit, every file introduced in its final form,
  moves kept pure.
- A verified final-tree invariant (empty diff or exactly the intended edits) proving no content was lost.
- An intact `backup/<name>` ref and an untouched shared branch until a human confirms integration.
