---
id: source-files
name: Source files
description: Language-neutral file hygiene - one concept per file under a line cap, no AI text marker (typographic punctuation, invisible carrier, exotic space) and no AI provenance key in a source or markdown file.
tags:
    - style
    - source-files
analyzers:
    - file-size
    - text-markers
    - typographic-markers
---

# Source files

> A source file holds one concept under a line cap, and carries no AI text marker: no typographic
> punctuation, no invisible character, no AI provenance key. The checks scan every governed
> extension, so one standard serves a Go, Dart, TypeScript, JavaScript or Svelte file alike.

Applies to: every governed source extension (e.g. `.go`, `.dart`, `.ts`, `.tsx`, `.js`, `.mjs`,
`.cjs`, `.jsx`, `.svelte`, `.vue`, `.py`, `.rs`) and every `.md` file.

{{define "when"}}
- Adding to a file that is growing, or deciding whether to split it.
- Writing a comment, string, or markdown prose that could carry a non-ASCII character.
- Pasting text from a chat or a document into a source file.
{{end}}

{{define "must"}}
1. Keep one concept per file (~180 lines, ~468 for a test file); split before a file grows past
   that (`file-size`).
2. No AI text marker in a source or markdown file. Two classes, and they are handled differently.
   Typographic punctuation - em dash U+2014, en dash, ellipsis U+2026, curly quotes - is reported
   and never rewritten, because inside a string literal it is data; replace it with its ASCII form
   by hand (`typographic-markers`). In markdown the fence is machine-readable, so a fenced block or
   an inline code span is skipped outright - the character there is content being shown, not prose
   style. Invisible carriers - zero-width characters, bidi controls, soft hyphens, exotic spaces -
   are removed by `pulsarules_cli clean --write`, and only the ones no neighbouring context can
   justify (`text-markers`). A marker that a test needs as data is written as the language's escape
   (Go `\u2014`), never as the character, so the fixture never trips the check that validates it.
3. How hard a marker lands depends on its class. An invisible carrier no neighbouring context can
   justify blocks: the bidi controls among them are the Trojan Source vector (CVE-2021-42574), where
   the diff a reviewer reads is not the code the compiler sees. One that may be load-bearing - a
   joiner between emoji, a byte order mark - only advises, because this check cannot judge it.
   Typographic punctuation blocks by default; a project that treats ASCII punctuation as house style
   rather than a defect lowers it through the analyzer's `severity` param (`error`, `warning`,
   `info`), which `--typographic-severity` bakes into the generated hooks at install time.
4. No AI provenance key in a markdown frontmatter block (`generator`, `ai_generated`, `claude`,
   `synthid`, `c2pa`, ...). Prose that merely names a vendor is not provenance and is not flagged.
{{end}}

{{define "forbidden"}}
- A file past its line cap instead of split by concept.
- An AI text marker in a source or markdown file: typographic punctuation, an invisible carrier, an
  exotic space, or an AI provenance key in a markdown frontmatter block.
{{end}}

{{define "validation"}}
- [ ] No file past ~180 lines (~468 for a test file); a long file was split by concept.
- [ ] No AI text marker in a source or markdown file; a marker a test needs as data is an escape.
- [ ] A blocking marker finding was fixed, not silenced; any `severity` the project lowered is a
  deliberate house-style choice, never a way past a bidi control.
{{end}}
