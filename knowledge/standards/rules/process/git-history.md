---
id: git-history
name: Git history
description: Surgical git history rewriting - branch safety and the force-push consent gate, decompose don't re-package, fold every fix-up into its file's creation commit, keep moves pure, verify the final-tree invariant, and the scripted rebase mechanism.
tags:
    - git
    - history
analyzers:
    - commit-move-purity
---

# Git history

> Rewriting history is a review, not a repackage: work on a fresh branch behind a backup ref, never
> force-push a shared branch without consent, fold every fix-up into the commit that created the
> file, keep move/rename commits pure, and prove the rewrite is safe with the final-tree invariant.

Applies to: rebuilding a squash or temp history into clean incremental commits.

{{define "when"}}
- Rebuilding a squashed or temp commit into clean incremental history.
- Cleaning, reordering, or splitting commits before review.
- Folding a later change into the commit that introduced a file.
- Splitting a commit that mixes a move/rename with new implementation.
{{end}}

{{define "must"}}
1. SAFETY FIRST. Work on a fresh branch. NEVER rewrite or force-push a pushed/shared branch (e.g.
   `main`) without explicit human consent - integration is the human's call. Before ANY rewrite
   create a backup branch (`backup/<name>`); keep it until integration is confirmed.
2. DECOMPOSE, don't re-package. As you split a squash, run the matched skills (start with
   `project-router`) over the implementation and apply the genuine improvements you detect (bugs,
   missing tests, idiom drift). The rewrite is a review, not a repackage.
3. FOLD EVERY FIX-UP INTO THE COMMIT THAT CREATED THE FILE. Any later change that only brings a
   pre-existing file to the shape it should have had at creation - hygiene (blank lines, package-doc
   removal), DI inversion, `RegisterSingleton`->`RegisterConstructor`, generic type-param removal, an
   added port/method, a conformance refactor (e.g. `regexp`->`strutils`), a serialization/tag fix,
   AND config edits (`go.mod`/`go.work`/`Taskfile`) - FOLDS into that file's creation commit; drop
   the standalone fix-up commit. Fold only as far back as dependency order allows: a file cannot be
   born referencing a package that does not exist yet, so a fold that would break the build either
   stops at the earliest commit where it compiles, or the step declares itself `[wip]`
   (see `[[commits]]`). "Final form" is the goal, not an absolute - the dependency graph decides how
   much of it any one commit can carry.
4. Genuinely-new functionality stays its own commit (folding it would be anachronistic - it depends
   on things that do not exist yet). A MIXED commit is hunk-split: the fix-up part folds, the new
   part stays.
5. A move/rename (`:open_file_folder:`) commit is PURE RENAMES ONLY while decomposing a rewrite. Never mix a
   relocation with new implementation; split the new code into separate `feat` commits placed after
   the move.
6. THE FINAL-TREE INVARIANT (the safety proof). After every rewrite, `git diff
   <pre-rewrite-snapshot> HEAD` is EMPTY - the rewrite only moved commit boundaries, no content
   changed - OR it equals EXACTLY the intended new edits and nothing else. An unexpected non-empty
   diff means content was lost or misplaced; fix it before moving on.
{{end}}

{{define "recipe"}}
1. MECHANISM. Snapshot the desired final tree as a ref (e.g. `foldsrc`). Run `git rebase -i --root`
   with a scripted todo (`GIT_SEQUENCE_EDITOR="cp todo.txt"`): `drop` each fix-up commit and `exec
   git checkout <foldsrc> -- <files> && git commit --amend --no-edit` at each file's creation
   commit. Generate the file->creation map with `git log --diff-filter=A --reverse -- <file>`; use
   `git show --no-renames --name-status` so moves classify as add+delete (checkout the new path,
   then `git rm -q --ignore-unmatch` the old). Resolve any conflict by checking out the `foldsrc`
   version. A commit emptied by folding is dropped.
2. A step that cannot compile MARKS itself `[wip]` (see `[[commits]]`) rather than the rewrite
   silently exempting every commit: the log then names which steps are partial. The FINAL tree plus
   a full build+lint+test sweep is still the gate. Run the per-commit sweep in a THROWAWAY WORKTREE
   - a killed sweep in the main checkout strands it on a detached HEAD.
3. Never `cherry-pick -x` while rebuilding: it appends a `(cherry picked from commit ...)` line,
   which is a tool-attribution trailer and blows the body cap. Lint every rewritten message.
{{end}}

{{define "forbidden"}}
- Rewriting or force-pushing a pushed/shared branch (`main`) without explicit human consent.
- Deleting backup refs before integration is confirmed.
- A rebase that changes the final tree unintentionally (always verify the invariant).
- A standalone commit that only "fixes up" a file to the form it should have had at creation.
- Delegating any part of a history rewrite to a subagent (see `[[agent-orchestration]]`) - rebase,
  fold, reorder, and force-push stay in the main session under direct control.
{{end}}

{{define "validation"}}
- [ ] Backup branch created; the pushed/shared branch untouched (no force-push without consent).
- [ ] `git diff <pre-rewrite-snapshot> HEAD` empty (pure reorg) OR exactly the intended edits.
- [ ] Every fix-up folded into its file's creation commit; no standalone "bring to lint bar / invert
  DI / add port / clean config" commit remains.
- [ ] Each `:open_file_folder:` move shows only renames (`git show --name-status`); new code is separate
  commits.
- [ ] Full sweep green on the final HEAD (build + lint + tests).
- [ ] Every commit that does not build declares `[wip]`; the series ends unmarked and green.
- [ ] Every rewritten message linted; no `(cherry picked from commit ...)` line survives.
{{end}}
