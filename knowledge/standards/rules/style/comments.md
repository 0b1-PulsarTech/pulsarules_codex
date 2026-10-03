---
id: comments
name: Comments
description: Comments exist only for a non-obvious why, as short own-line comments at the line they explain; no doc comments, no file banners, no history narration. Language-neutral.
tags:
    - style
    - comments
analyzers:
    - top-of-file
    - big-comment
    - history-narration
    - test-comment
---

# Comments

> A name and a signature say what code does. A comment is allowed only when a non-obvious why would
> otherwise be lost, and then it is a short own-line comment at the line it explains. What changed
> belongs in the commit message, never in a comment.

Applies to: every source file the project governs (`.go`, `.dart`, `.ts`, `.svelte`, `.proto`, `.sql`,
`.nix`). Go-only checks are labelled as such.

{{define "when"}}
- About to write a comment, a doc comment, or a file header.
- Deleting or moving a comment while refactoring.
- Reviewing a diff for comment noise.
{{end}}

{{define "must"}}
1. Write a comment only where a non-obvious why exists: a constraint, an invariant, a surprising
   bound, a reason the simpler-looking code is wrong. Start it with a capital letter and write it in
   the present tense.
2. Put it on its own line, at the statement it explains, never above a function: reasoning parked
   away from its line drifts from the code and is read once.
3. Keep it to 3 lines at most (`big-comment`); one sentence of why, not a paragraph. Machine-read
   lines (directives, generated markers) are not counted.
4. No doc comment or docstring on a function, method, type, constant, variable or package, public or
   not; the name and signature say it. Delete it; never move it into the body. (Go: `doc-comment`
   enforces funcs, methods, types.) Exceptions: a deprecation marker (Go `// Deprecated:` of at most
   3 lines; Dart `@Deprecated(...)`), a directive-only group, and a `simplification:` marker (see
   [[minimalism]]), which is also exempt from the 3-line cap.
5. No file-top comment (`top-of-file`): no banner, no license block, no package or library docstring
   (Go: a `// Package foo` comment or a `doc.go`). Only a generated-code marker or a tool directive
   may come before the code.
6. No multi-line comment block as the first thing in a function body (Go: `body-header-comment`).
7. No history narration (`history-narration`): phrases such as "was replaced by", "used to be",
   "it used to call", "the old code", "previously we", "was changed to", "no longer needed". The
   commit message carries what changed and why it changed. Passive "is used to" is not history and
   is not flagged.
8. ASCII only (`typographic-markers`, which scans every source extension): no em dash, curly quotes,
   or ellipsis character.
9. Directive and machine comments stay (Go: `//nolint`, `//go:build`, `//go:embed`, `//go:generate`,
   `// goverter:`; Dart: `// ignore_for_file:`, `// ignore:`; any language: generated-code markers).
10. No comments inside test functions or helpers except directive-only groups and an Example's
    `// Output:` (Go: `test-comment`).
{{end}}

{{define "forbidden"}}
- A comment that restates a name, a signature, or the control flow beneath it.
- A comment block longer than 3 lines.
- A free-standing comment block between declarations, outside any function body.
- A comment at the top of a file other than a generated-code marker or a tool directive.
- A doc comment or docstring on a function, method, type, constant, variable or package.
- A multi-line comment block opening a function body.
- A comment narrating what changed or what the code used to do.
- Non-ASCII punctuation in a comment.
- A comment inside a test function or helper other than a directive-only group or an Example's
  `// Output:`.
{{end}}

{{define "validation"}}
- [ ] Every kept comment states a why the code cannot, on its own line at the statement it explains.
- [ ] No comment block over 3 lines; no multi-line block at the top of a function body.
- [ ] No file-top comment beyond a generated marker or a tool directive.
- [ ] No doc comment or docstring on a function, method, type, constant, variable or package.
- [ ] No history narration; what changed is in the commit message.
- [ ] Comments are ASCII only; tests carry no comment but directive-only groups and an Example's
      `// Output:`.
{{end}}
