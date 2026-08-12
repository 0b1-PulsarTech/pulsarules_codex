<!-- Installed by pulsarules_cli; remove or edit this file to disable. -->

# Skill-router reminder hook — why it exists

> Keep this file with the hook. It explains a non-obvious mechanism; without the
> rationale a future maintainer will reasonably delete it as noise.

This file explains the mechanism ONE binary uses to reach two different hosts. Read
[Shared mechanism](#shared-mechanism-both-hosts) regardless of which host you work on, then jump to
just your host's section - [Claude Code](#claude-code) or [opencode](#opencode) - not both.

## The problem it solves

This repo's engineering contract is encoded as **skills** dispatched by the `project-router` skill.
In practice, autonomous skill activation is unreliable: the agent invokes `project-router`, sees the
dispatch table, but then **skips straight to implementation without actually loading and applying
every matched skill** at the moment it writes code. A skill description is a *suggestion the model
can ignore*; nothing forces the decision.

Concrete failures that motivated this hook:

- Wrote unit tests **without loading `integration-tests`** → used a `now func() time.Time`
  clock-injection field instead of `testing/synctest`, and omitted `t.Parallel()` and the
  `testCases`/`testCase` naming convention.
- Loaded `code-minimalism` / `go-style` / `errors-logging` / `security` **late or not at all** →
  missed the full baseline pass.

## Shared mechanism (both hosts)

A **hook fires deterministically**; a description-based match does not. So instead of trusting the
model to self-route, a single Go binary (copied to `<host>/bin/` at install) emits reminder text at
each lifecycle moment. The binary is dispatched by **mode** - a host-neutral name - not by either
host's own hook-event vocabulary; each host's own wiring (see its section below) maps its hook/trigger
names onto these modes:

| Mode              | When                                                                                                      | What it injects                                                                                                                        |
|-------------------|------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| `session-start`   | once per session                                                                                           | The routing contract: invoke `project-router`, load every matched skill, apply while writing; test conventions; verify-before-commit.  |
| `subagent-start`  | every time a subagent starts, even under a session-start marker the parent already burned                  | The same routing contract, minus the commit/verify tail - a subagent never commits.                                                    |
| `user-prompt`     | **every turn**, no dedup flag                                                                              | A short routing question (`user-prompt.txt`, 230 bytes, ceiling 300, enforced by `TestUserPromptNudgeStaysShortAndDistinct`): asks whether `project-router` has run for this task, and states that skipping the router skips the rules for any Go/SQL/config write. Fires on every turn (no once-per-session gate), unlike `session-start`'s one-shot contract, because mid-session drift off the router happens well past turn one - the cost of firing every turn is why the text stays this short and does not repeat what `contract.txt`/`pre-edit.txt` already say. |
| `pre-edit`        | first write to each file the router matches to a skill (per-file-path flag, then silent for that file; asks the same `Router.SkillsForFile` post-edit uses, not a separate hardcoded extension list) | A prominent "confirm `project-router` ran and the matched skills are loaded and active" reminder (they still carry the full contract), plus the minimalism decision ladder, the newest-stdlib names, the named-domain-type rule, the two-call-site rule, and the scope guard — the clauses no analyzer catches later, carried in substance rather than named as a bare pointer — at the exact write moment, once per distinct file. |
| `pre-search`      | first Go-targeted search of a session (per-session flag, then silent); only when the project has a `go.mod` and `gopls-navigation` is installed | A nudge to use the gopls MCP (`go_search`, `go_symbol_references`, `go_file_context`) instead of textual grep for Go code.              |
| `post-edit`       | after each Write/Edit (deduped per session by content)                                                     | A file-type-aware checklist of the skills to re-validate against the edited file's extension (only those actually installed).          |
| `stop`            | end of every turn, only when a `ScopeChanged` governance run over the dirty tree produces findings (deduped by content) | The findings block, a checklist grouped by the changed file types, and the uncommitted change list. Silent otherwise - it never asks for a commit; that is the operator's call. |
| `subagent-stop`   | never - wired by neither host, and the mode is a no-op                                                     | Nothing. A subagent must not commit (git stays in the main session), so a dirty-tree block aimed at one only derails the work it was spawned to do. |
| `session-end`     | once when the session ends                                                                                 | Nothing — it cleans up the per-session marker files the other modes create.                                                            |

The binary reads exactly three environment variables - `PULSARULES_PROJECT_DIR`,
`PULSARULES_SKILLS_DIR`, and `PULSARULES_LOG_PATH` - and nothing else host-specific, so
`internal/hook`, `internal/obs`, and `internal/cli` never hardcode a host's own skills or log
layout (`.claude/skills` vs `.opencode/skills`, `.claude/hook-execution.log` vs
`.opencode/hook-execution.log`). Each host's own wiring sets `PULSARULES_SKILLS_DIR` to its own
layout; without it, the `post-edit` checklist and the `pre-search` nudge degrade to their generic,
skill-list-free form. Each host's own wiring likewise sets `PULSARULES_LOG_PATH` to its own layout;
without it (an older installed wrapper, or the binary run by hand), `--log-level` has nowhere safe
to write, so `internal/obs` disables logging outright rather than guessing a location.

The reminder text has a single source of truth. `contract.txt` and `contract-tail.txt` render through
`internal/contract` for both `session-start` (contract plus tail) and `subagent-start` (contract only
- a subagent never commits), so both hook variants and the AGENTS.md routing-contract section all
derive from the same asset instead of hand-maintained copies. `pre-edit.txt` and the other mode texts
work the same way: a single embedded template read by whichever host's wiring calls that mode.

### Design constraints shared by both hosts (don't regress these)

- **Non-blocking.** The binary always exits `0`. A non-zero exit from a pre-write hook can *block*
  the edit on some hosts; this mechanism must only ever nudge.
- **Cheap.** The `pre-edit` reminder is gated per file path (a hashed marker under
  `<tmpdir>/skill-hook-pre-edit-file-<hash><session_id>`), so a re-edit of a file already reminded
  about stays silent, but each newly touched file still gets one reminder - not on every edit, and
  not muted after the session's first file either.

## Claude Code

### How it is wired (orchestrator + binary)

`skill-router-reminder.sh` is a thin orchestrator: the settings entry locates it via
`$CLAUDE_PROJECT_DIR` (a Claude Code variable, legitimately named there); the script forwards the
mode (`session-start`/`pre-edit`/...) and the stdin payload to the installed binary, after exporting
`PULSARULES_PROJECT_DIR`, `PULSARULES_SKILLS_DIR=$CLAUDE_PROJECT_DIR/.claude/skills`, and
`PULSARULES_LOG_PATH=$CLAUDE_PROJECT_DIR/.claude/hook-execution.log`. If the binary is absent the
script exits `0` (no-op).

Claude Code's own hook events map onto the shared modes above like this:

| Claude Code hook                                     | Mode           |
|--------------------------------------------------------|----------------|
| `SessionStart`                                          | `session-start`  |
| `SubagentStart`                                          | `subagent-start` |
| `UserPromptSubmit`                                       | `user-prompt`    |
| `PreToolUse` matcher `Write\|Edit\|MultiEdit`            | `pre-edit`       |
| `PreToolUse` matcher `Grep\|Glob\|Bash`                  | `pre-search`     |
| `PostToolUse` matcher `Write\|Edit\|MultiEdit`           | `post-edit`      |
| `Stop`                                                   | `stop`           |
| `SessionEnd`                                             | `session-end`    |
| *(no event - Claude Code fires no `SubagentStop`)*       | `subagent-stop`  |

### Claude Code-specific design constraints

- **Local.** It lives under `.claude/` (gitignored) and is wired into `.claude/settings.json` (or
  `settings.local.json` with `--hooks-scope local`).
- **Watcher caveat.** A freshly-installed hook only goes live after `/hooks` is opened once or the
  session restarts (`SessionStart` then fires next session).

## opencode

`opencode-plugin.js` gives opencode the same governance context Claude Code gets from
`skill-router-reminder.sh`, adapted to opencode's plugin API instead of its hooks JSON. opencode
dispatches plugin hooks generically by name; only `experimental.chat.system.transform`,
`tool.execute.before`, and `tool.execute.after` are hook names opencode's `trigger()` actually
dispatches. `session.created`, `session.idle`, and `session.deleted` are bus-event names, not hook
names - a plugin can register handlers for them and they will never fire.

Before shelling out, the plugin exports `PULSARULES_SKILLS_DIR=<root>/.opencode/skills` and
`PULSARULES_LOG_PATH=<root>/.opencode/hook-execution.log` via its `.env({...})` call, then guards on
the binary's existence via a plain `accessSync` syscall (mirroring `skill-router-reminder.sh`'s
`[ -x "$bin" ] || exit 0` without spawning a subprocess to do it), since `.opencode/bin/` is
gitignored and a fresh clone has the plugin installed with no binary yet.

opencode's own hooks map onto the shared modes like this:

- **`experimental.chat.system.transform`** fires at the start of every turn. It appends the
  `session-start` contract text to `output.system` (a mutable array), but the contract text is NOT
  re-injected every turn: the `session-start` mode itself is still gated once-per-session by the
  binary (`internal/hook/hook.go`'s `emitSessionStart`, behind `OncePerSession("session-start")`), so
  the contract is actually injected only on a session's first request - matching Claude Code's
  once-per-session `SessionStart` hook, just reached from a trigger that happens to fire every turn.
  The same handler also calls the binary's `user-prompt` mode on every invocation (no once-per-session
  gate there - see `emitUserPrompt`) and appends its result too, so opencode gets the per-turn nudge
  (`user-prompt.txt`) every turn even though the full contract only lands once.
- **`tool.execute.after`** forwards to the binary's `post-edit` mode exactly like Claude Code's
  `PostToolUse`, appending the result to `output.output` (the field the model reads back - never
  `console.log`, which only reaches opencode's server log).
- **`tool.execute.before`** is deliberately NOT registered: its output object is only `{args}`, with
  no field the model reads back, so there is no channel to deliver a `pre-edit` nudge through; see the
  `// simplification:` comment in the script for the ceiling and upgrade path.
- The governance/`stop` check (the dirty-worktree nag) has no opencode hook to attach to either and is
  not wired; see its own `// simplification:` comment for the ceiling and upgrade path.
- `pre-search` and `session-end` are likewise not wired in opencode today - only the three hooks
  opencode's `trigger()` actually dispatches are used.

### opencode-specific design constraints

- **Local.** It lives under `.opencode/` (gitignored) and is registered as an opencode plugin.
