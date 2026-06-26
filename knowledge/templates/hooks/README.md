# Skill-router reminder hook — why it exists

> Keep this file with the hook. It explains a non-obvious mechanism; without the
> rationale a future maintainer will reasonably delete it as noise.

## The problem it solves

This repo's engineering contract is encoded as Claude Code **skills** dispatched by the
`project-router` skill. In practice, Claude Code's **autonomous skill activation is unreliable**:
the agent invokes `project-router`, sees the dispatch table, but then **skips straight to
implementation without actually loading and applying every matched skill** at the moment it writes
code. A skill description is a *suggestion the model can ignore*; nothing forces the decision.

Concrete failures that motivated this hook:

- Wrote unit tests **without loading `integration-tests`** → used a `now func() time.Time`
  clock-injection field instead of `testing/synctest`, and omitted `t.Parallel()` and the
  `testCases`/`testCase` naming convention.
- Loaded `code-minimalism` / `go-style` / `errors-logging` / `security` **late or not at all** →
  missed the full baseline pass.

## Why a hook (and not just docs / skill descriptions)

A **hook fires deterministically**; a description-based match does not. So instead of trusting the
model to self-route, we inject the contract at each relevant lifecycle moment:

| Hook                                                 | When                                                                                                      | What it injects                                                                                                                        |
|------------------------------------------------------|------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| `SessionStart` (`session-start`)                     | once per session                                                                                           | The routing contract: invoke `project-router`, load every matched skill, apply while writing; test conventions; verify-before-commit.  |
| `UserPromptSubmit` (`user-prompt`)                   | first prompt of a session (per-session flag, then silent)                                                  | A one-shot "did you route through `project-router`?" nudge before any Go/SQL/config work.                                              |
| `PreToolUse` `Write\|Edit\|MultiEdit` (`pre-edit`)   | first `.go`/`.sql` write of a session (per-session flag, then silent)                                      | A pointed "the matched skills must be loaded *now*" reminder — at the exact failure moment.                                            |
| `PreToolUse` `Grep\|Glob\|Bash` (`pre-search`)       | first Go-targeted search of a session (per-session flag, then silent); only when the project has a `go.mod` and `gopls-navigation` is installed | A nudge to use the gopls MCP (`go_search`, `go_symbol_references`, `go_file_context`) instead of textual grep for Go code.              |
| `PostToolUse` `Write\|Edit\|MultiEdit` (`post-edit`) | after each Write/Edit (deduped per session by content)                                                     | A file-type-aware checklist of the skills to re-validate against the edited file's extension (only those actually installed).          |
| `Stop` (`stop`)                                      | end of every turn, only when a `ScopeChanged` governance run over the dirty tree produces findings (deduped by content) | The findings block, a checklist grouped by the changed file types, and the uncommitted change list. Silent otherwise - it never asks for a commit; that is the operator's call. |
| `SubagentStop` (`subagent-stop`)                     | never - the group is no longer wired, and the mode is a no-op                                              | Nothing. A subagent must not commit (git stays in the main session), so a dirty-tree block aimed at one only derails the work it was spawned to do. |
| `SessionEnd` (`session-end`)                         | once when the session ends                                                                                 | Nothing — it cleans up the per-session marker files the other hooks create.                                                           |

## How it is wired (orchestrator + binary)

`skill-router-reminder.sh` is a thin orchestrator: it forwards the mode (`session-start`/`pre-edit`)
and the stdin payload to the installed `pulsarules_codex-installer` binary (copied to
`.claude/bin/` at install) via `"$CLAUDE_PROJECT_DIR/.claude/bin/pulsarules_codex-installer" hook
"$1"`. The binary emits the `hookSpecificOutput` JSON from the embedded reminder templates
(`session-start.txt`, `pre-edit.txt`), so the contract text has a single source of truth instead of a
copy baked into the shell script. If the binary is absent the script exits `0` (no-op).

## Design constraints (don't regress these)

- **Non-blocking.** The script (and the binary) always exit `0`. A non-zero exit from a `PreToolUse`
  hook can *block* the edit; this hook must only ever nudge.
- **Cheap.** The `pre-edit` reminder is gated by a per-session flag (`<tmpdir>/skill-route-<session_id>`)
  so it fires once per session, not on every edit.
- **Local.** It lives under `.claude/` (gitignored) and is wired into `.claude/settings.json` (or
  `settings.local.json` with `--hooks-scope local`).
- **Watcher caveat.** A freshly-installed hook only goes live after `/hooks` is opened once or the
  session restarts (`SessionStart` then fires next session).
