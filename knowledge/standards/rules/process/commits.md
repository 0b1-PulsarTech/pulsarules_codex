---
id: commits
name: Commits
description: Emoji-prefixed Conventional Commits; one logical change per commit; no tool-attribution trailers; each commit builds and passes tests standalone.
tags:
    - git
    - commits
analyzers:
    - commit-lint
    - commit-move-purity
    - branch-name
---

# Commits

> Emoji-prefixed Conventional Commits: `:<emoji>: <type>[(<scope>)]: <subject>`, imperative subject
> under 72 chars, one logical change per commit, no AI/tool-attribution trailers, each commit
> builds and passes tests standalone.

Applies to: every commit in every project.

{{define "when"}}
- Writing a new commit message.
- Squashing/rebasing or cleaning history before review.
- Documenting a breaking change.
- Committing AI-assisted changes.
{{end}}

{{define "must"}}
1. Format the subject as `:<emoji>: <type>[(<scope>)]: <subject>` - a gitmoji shortcode, then a
   Conventional Commit type, optional scope, then an imperative subject that starts with an Uppercase
   letter, has no trailing period, and stays under 72 chars.
2. Take the emoji from the project catalog (`internal/emoji/data/catalog.txt`, the
   GitHub shortcodes at Emoji version 1.0 or older, minus country flags). Anything outside it is
   REJECTED by the hook, because newer emoji render as raw text in the tools the history is read
   through. When in doubt prefer an emoji that predates 2011.
3. `:robot:`, `:test_tube:` (use `:tea:`), `:compass:` and `:sparkles:` are PROHIBITED outright,
   including on AI-assisted commits.
4. Pick the emoji for the AREA the change touches, not for its type - a `feat` extending an existing
   area takes that area's emoji. Keep an area recognisable through a family rather than one fixed
   mascot, in specificity order: {{ emojiAnchors }}.
5. NO emoji may repeat within five commits. The hook blocks it, and rotating inside an area's family
   is how you keep both variety and recognisability. Root commit is `:ghost:`, merges are
   `:volcano:`; both are exempt.
6. Use a type from: feat, fix, chore, refactor, docs, test, perf, build, ci, style, revert. Add a
   scope (the module/feature) when it sharpens the message.
7. One logical change per commit. Do not mix a refactor with a feature.
8. A move is its own commit, and it comes first. When a change relocates or renames a file AND also
   edits it, commit the pure move/rename first, then the edits in a following commit. The move commit
   MAY carry the mechanical consequences of relocating - the `package` clause, import paths, an
   import alias - because without them the commit would not compile (item 11); any other edit waits
   for the next commit. A mixed diff hides the edit inside the rename and degrades git's rename
   detection, so the reviewer loses the only signal that the content changed.
9. A commit IS its subject. Default to subject-only: the subject plus the diff already carry the
   change. A body is an EXCEPTION, written only when a reader needs something absent from both - a
   non-obvious why, a constraint that forced the shape, an alternative that was rejected. If you
   cannot name what the body adds beyond the subject and the diff, do not write one. An exceptional
   body stays short (the hook rejects an oversized one) and wraps at 100 columns. Footer references:
   `Refs: #42`, `Closes: #99`.
10. Breaking change: add a `BREAKING CHANGE:` footer with migration notes.
11. Each commit must compile and pass tests standalone (docs-only commits are exempt from the test
    bar). A commit that cannot - a deliberate step in a series, or a history rebuild where a file
    would be born referencing a package that does not exist yet - DECLARES it by marking the
    description `[wip]`: `:<emoji>: <type>(<scope>): [wip] <Subject>`. The marker IS the exemption,
    so the log says which steps are partial and `git bisect` can skip them. An unmarked commit is
    always held to the bar; `[wip]` never appears on a commit meant to stand alone, and a series
    ends with an unmarked commit that restores green.
12. Do NOT append `Co-Authored-By`, `Claude-Session`, or any other tool-attribution trailer, even
    for AI-assisted commits.
13. Name the BRANCH `<type>/<description>`, with an optional `(<scope>)` between them -
    `feat/branch-name-check`, `fix(hook)/worktree-hooks-dir`, `feature/add_branch-check`. The type is
    a commit type from item 6 or a gitflow line (`feature`, `release`, `hotfix`, `bugfix`,
    `support`). The pre-push check blocks a branch with no recognized prefix, so a tool-generated
    name (`claude/...`) does not reach a remote. `main`, `master` and `develop` are exempt: a trunk
    names a line of history, not a change. A project with its own vocabulary beyond both sets adds it
    through the analyzer's `extra_types`, which install bakes into the hook.
{{end}}

{{define "forbidden"}}
- Combining refactor + feature (or multiple logical changes) in one commit.
- Commits that do not compile or fail tests.
- A subject that starts lowercase or ends with a period; subjects over 72 chars; missing emoji or type.
- An emoji outside the project catalog, or any of `:robot:`, `:test_tube:`, `:compass:`, `:sparkles:`.
- Repeating an emoji used in any of the last five commits.
- A move/rename commit that also edits the moved file beyond the mechanical consequences of the
  relocation (package clause, import paths, an import alias).
- `Co-Authored-By` / `Claude-Session` / tool-attribution trailers (the hook rejects them).
- A body written by habit rather than necessity; oversized bodies; bodies that merely restate the
  subject or the diff.
{{end}}

{{define "validation"}}
- [ ] Subject matches `:<emoji>: <type>[(<scope>)]: <subject>`, Uppercase first letter, no period, under 72.
- [ ] Emoji in the catalog, not prohibited, absent from the last five commits, and named for the area.
- [ ] Type is a valid Conventional Commit type.
- [ ] One logical change per commit (no refactor+feature mix).
- [ ] A move/rename is its own commit, staged first, with no edits beyond the mechanical consequences
  of relocating.
- [ ] Subject-only unless a body was strictly necessary; any body names what the subject and diff cannot.
- [ ] Commit compiles and passes tests standalone, or declares `[wip]` (unless docs-only).
- [ ] Every `[wip]` series ends with an unmarked commit that restores green.
- [ ] No tool-attribution trailer.
{{end}}
