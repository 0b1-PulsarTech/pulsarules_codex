# Delegated commands

A project can hand a check to its own tool and have the output reported as governance findings.
The only format is `dart-analyze-json` (`dart analyze --format=json`).

```yaml
version: 1
delegated:
  - id: dart-analyze            # [a-z0-9][a-z0-9_-]*, at most 64 bytes, unique
    dir: apps/mobile            # optional; relative, must stay inside the project
    format: dart-analyze-json
    rule: dart-style            # optional RuleID attached to findings
    run: [dart, analyze, --format=json]
```

## Trust

`delegated.run` is a command the tool executes. A committed `.pulsarules.yaml` therefore has the
same trust as a committed `Taskfile`: review it like one. The id may not be a built-in analyzer id
or `config`. `dir` is resolved through symlinks and must remain inside the project.

## When it runs

Only at full scope: plain `pulsarules_cli governance` and the pre-push hook. The pre-push hook is
opt-in: `install` sets up `commit-msg` and `pre-commit` by default, so add it with
`install --git-hooks commit-msg,pre-commit,pre-push`. Delegated commands do not run in the
pre-commit hook (commit scope) or the Stop hook (changed scope). A full scope without
`--all-files` still analyses the whole project.

The command runs without a shell, in its own process group (Unix), with a 5 minute timeout and a
32 MiB cap per output stream. For Dart, exit codes 1, 2 and 3 are normal diagnostics. Governance
does not forward an interrupt: if governance is killed the tool may keep running. Windows has no
group kill.

## Findings

| Id | Severity | Meaning |
| --- | --- | --- |
| `<id>/<code>` | mapped from the tool | a diagnostic (for example `dart-analyze/unused_import`) |
| `<id>/unavailable` | INFO | the binary is missing, or a precondition failed (see below) |
| `<id>/config` | ERROR | the entry is invalid |
| `<id>/timeout` | ERROR | the 5 minute limit was hit |
| `<id>/failed` | ERROR | the tool failed; includes a stderr tail, which may contain secrets |
| `<id>/dropped` | worst dropped | some diagnostics are on paths outside the project or missing |

A private `pulsarules.yaml` may tune an id declared only in the versioned file, for example
`analyzers: {dart-analyze: {severity: error}}` or `enabled: false`; no `delegated` entry is needed
there.

Set `enabled` or `severity` for the whole entry under `analyzers:<id>`, and for one diagnostic or
a path under `overrides` with a sub-id such as `dart-analyze/unused_import`. A disabled entry is
not run.

Paths: a diagnostic in a gitignored file is silently omitted. One outside the project or missing
is counted in `<id>/dropped` when other diagnostics are kept (for example `... 2 diagnostics on
paths outside the project or missing (first: "/etc/passwd":1 lint_out)`). A symlink alias of a
listed file is kept at the alias path. When every diagnostic is outside the project or missing,
nothing is kept and the output is one `<id>/failed: exit N but no diagnostics inside the project`
instead of `<id>/dropped`.

## Dart precondition

Before running a Dart format, each pubspec directory must contain `.dart_tool/package_config.json`,
or have one in a parent directory up to the project root (a workspace).
If not, the run is skipped with `<id>/unavailable`: "pubspec.yaml in . has no
.dart_tool/package_config.json; run `dart pub get` (then `dart run build_runner build`); <id> was
skipped". Only the first failing pubspec is named. The check is by format, so a wrapper such as
`sh -c "dart pub get && dart analyze ..."` is skipped too.

When generated code is missing (`uri_has_not_been_generated`, or `uri_does_not_exist` on a
generated suffix such as `.g.dart`, `.freezed.dart`, `.mocks.dart`, `.gr.dart`, `.config.dart`,
`.pb*.dart`, `.template.dart`, or a part/import/export of one found by scanning the workspace),
every ERROR is dropped, including promoted lints, and one `<id>/unavailable` notice says
"generated code is missing (...); run `dart run build_runner build`; N derived errors dropped".
Warnings, infos and lints are kept.

## Known limitations

- A chain of two symlinks (an ignored link, then a tracked link, then a target outside the
  project) is not resolved and is not loud: the diagnostic is silently omitted, and a real ERROR
  can pass. Point a tracked symlink straight at a file inside the project, or avoid link chains.
- A Go-quoted absolute path in a `first:` excerpt, and a clipped path, are shown unmarked.
- The `failed` stderr tail can contain secrets; do not publish CI logs blindly.
- No Ctrl-C propagation, as above.
