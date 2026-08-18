<!-- Installed by pulsarules_cli; remove or edit this file to disable. -->

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
| `SubagentStart` (`subagent-start`)                   | every time a subagent starts, even under a session-start marker the parent already burned                  | The same routing contract, minus the commit/verify tail - a subagent never commits.                                                    |
| `UserPromptSubmit` (`user-prompt`)                   | **every turn**, no dedup flag                                                                              | A compact per-turn digest (`user-prompt.txt`, 633 bytes, ceiling 785 - the session-start contract's size, enforced by `TestUserPromptDigestUnder785Bytes`): the router already ran and its skills are loaded, don't reload; pre-edit's per-file reminder goes silent on repeat edits, this is the check that still reaches those turns; route again only if the task shape changed. Replaced a once-per-session "did you route?" nudge that only ever caught turn one - the cost is real (785 bytes on every single turn of every session), so it stays deliberately shorter than that ceiling and does not repeat what `contract.txt`/`pre-edit.txt` already say. |
| `PreToolUse` `Write\|Edit\|MultiEdit` (`pre-edit`)   | first write to each file the router matches to a skill (per-file-path flag, then silent for that file; asks the same `Router.SkillsForFile` post-edit uses, not a separate hardcoded extension list) | A prominent "confirm `project-router` ran and the matched skills are loaded and active" reminder (they still carry the full contract), plus the minimalism decision ladder, the newest-stdlib names, the named-domain-type rule, the two-call-site rule, and the scope guard — the clauses no analyzer catches later, carried in substance rather than named as a bare pointer — at the exact write moment, once per distinct file.                     |
| `PreToolUse` `Grep\|Glob\|Bash` (`pre-search`)       | first Go-targeted search of a session (per-session flag, then silent); only when the project has a `go.mod` and `gopls-navigation` is installed | A nudge to use the gopls MCP (`go_search`, `go_symbol_references`, `go_file_context`) instead of textual grep for Go code.              |
| `PostToolUse` `Write\|Edit\|MultiEdit` (`post-edit`) | after each Write/Edit (deduped per session by content)                                                     | A file-type-aware checklist of the skills to re-validate against the edited file's extension (only those actually installed).          |
| `Stop` (`stop`)                                      | end of every turn, only when a `ScopeChanged` governance run over the dirty tree produces findings (deduped by content) | The findings block, a checklist grouped by the changed file types, and the uncommitted change list. Silent otherwise - it never asks for a commit; that is the operator's call. |
| `SubagentStop` (`subagent-stop`)                     | never - the group is no longer wired, and the mode is a no-op                                              | Nothing. A subagent must not commit (git stays in the main session), so a dirty-tree block aimed at one only derails the work it was spawned to do. |
| `SessionEnd` (`session-end`)                         | once when the session ends                                                                                 | Nothing — it cleans up the per-session marker files the other hooks create.                                                           |

## How it is wired (orchestrator + binary)

`skill-router-reminder.sh` is a thin orchestrator: the settings entry locates it via
`$CLAUDE_PROJECT_DIR` (a Claude Code variable, legitimately named there); the script forwards the
mode (`session-start`/`pre-edit`/...) and the stdin payload to the installed binary (copied to
`.claude/bin/` at install), after exporting `PULSARULES_PROJECT_DIR` and `PULSARULES_SKILLS_DIR` -
the only two variables the binary itself reads, so it never hardcodes a host's own variable name or
a host's own skills layout. The binary emits the `hookSpecificOutput` JSON from the embedded reminder templates
(`contract.txt`, `pre-edit.txt`), so the contract text has a single source of truth instead of a
copy baked into the shell script. `contract.txt` and `contract-tail.txt` are rendered through
`internal/contract` for both `session-start` (contract plus tail) and `subagent-start`
(contract only - a subagent never commits), so the two hook variants and the AGENTS.md routing
contract section all derive from the same asset instead of three hand-maintained copies. If the
binary is absent the script exits `0` (no-op).

Generic Go never hardcodes `.claude/skills` or `.opencode/skills`: `internal/hook` reads
`PULSARULES_SKILLS_DIR` at dispatch time (falling back to an unset/empty value, which degrades the
post-edit checklist and the pre-search nudge to their generic, skill-list-free form). Each host's
own installer wiring sets the variable to its own layout - `skill-router-reminder.sh` exports
`$CLAUDE_PROJECT_DIR/.claude/skills`; `opencode-plugin.js` exports `<root>/.opencode/skills` via its
`.env({...})` call below.

## opencode-plugin.js

`opencode-plugin.js` gives opencode the same governance context Claude Code gets from
`skill-router-reminder.sh`, adapted to opencode's plugin API instead of its hooks JSON. opencode
dispatches plugin hooks generically by name; only `experimental.chat.system.transform`,
`tool.execute.before`, and `tool.execute.after` are hook names opencode's `trigger()` actually
dispatches. `session.created`, `session.idle`, and `session.deleted` are bus-event names, not hook
names - a plugin can register handlers for them and they will never fire.

The plugin appends the `session-start` contract text to `output.system` (a mutable array) on
`experimental.chat.system.transform`. That hook fires at the start of every turn, but the contract
text is NOT re-injected every turn: the underlying `session-start` mode this calls is still gated
once-per-session by the binary itself (`internal/hook/hook.go`'s `emitSessionStart`, behind
`OncePerSession("session-start")`), so the contract is actually injected only on a session's first
request - matching Claude Code's once-per-session `SessionStart` hook, just reached from a trigger
that happens to fire every turn. The same handler also calls the binary's `user-prompt` mode on
every invocation (no once-per-session gate there - see `emitUserPrompt`) and appends its result too,
so opencode gets the per-turn digest (`user-prompt.txt`) every turn even though the full contract
only lands once. `tool.execute.after` forwards to the binary's `post-edit` mode exactly like its Claude Code
counterpart, appending the result to `output.output` (the field the model reads back - never
`console.log`, which only reaches opencode's server log). `tool.execute.before` is deliberately NOT
registered: its output object is only `{args}`, with no field the model reads back, so there is no
channel to deliver a pre-edit nudge through; see the `// simplification:` comment in the script for
the ceiling and upgrade path. The governance/stop check (the dirty-worktree nag) has no opencode
hook to attach to either and is not wired; see its own `// simplification:` comment for the ceiling
and upgrade path. Before shelling out, the plugin guards on the binary's existence via a plain
`accessSync` syscall (mirroring `skill-router-reminder.sh`'s `[ -x "$bin" ] || exit 0` without
spawning a subprocess to do it), since `.opencode/bin/` is gitignored and a fresh clone has the
plugin installed with no binary yet.

## Design constraints (don't regress these)

- **Non-blocking.** The script (and the binary) always exit `0`. A non-zero exit from a `PreToolUse`
  hook can *block* the edit; this hook must only ever nudge.
- **Cheap.** The `pre-edit` reminder is gated per file path (a hashed marker under
  `<tmpdir>/skill-hook-pre-edit-file-<hash><session_id>`), so a re-edit of a file already reminded
  about stays silent, but each newly touched file still gets one reminder - not on every edit, and
  not muted after the session's first file either.
- **Local.** It lives under `.claude/` (gitignored) and is wired into `.claude/settings.json` (or
  `settings.local.json` with `--hooks-scope local`).
- **Watcher caveat.** A freshly-installed hook only goes live after `/hooks` is opened once or the
  session restarts (`SessionStart` then fires next session).
