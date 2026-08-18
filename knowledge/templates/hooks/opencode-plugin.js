// pulsarules_codex governance plugin for opencode.
// Installed by pulsarules_cli; remove this file to disable.
//
// The binary path below must match internal/hook/install/opencodehook.go's
// binaryRel constant (".opencode/bin/pulsarules_cli"); it is a literal here
// because this file has no way to read a Go constant.
//
// Only hooks opencode's trigger() actually dispatches are registered below -
// verified by reading the installed opencode binary. session.created,
// session.idle and session.deleted are bus-event names, not hook names: a
// plugin registering handlers for them looks correct but never fires. That
// was the bug this file replaces; do not reintroduce those keys.
// tool.execute.before IS a real trigger()-ed hook, but its output object is
// only {args} - no field the model reads back - so it is deliberately not
// registered either; see the simplification note below.
//
// The JSON payload shape below (session_id, tool_input.file_path) must match
// hookPayload in internal/hook/wire.go's decodeHookPayload - see the comment
// there pointing back here.
import { accessSync, constants as fsConstants } from "node:fs";

export const PulsarulesGovernance = async ({ directory, worktree, $ }) => {
  const root = worktree || directory;
  const bin = `${root}/.opencode/bin/pulsarules_cli`;

  // .opencode/bin/ is gitignored, so a fresh clone has this plugin installed
  // but no binary yet. Stay silent rather than shelling out to a path that
  // does not exist - mirrors the [ -x "$bin" ] || exit 0 guard in
  // skill-router-reminder.sh. accessSync is a plain syscall, not a
  // subprocess: shelling out to the "test" utility instead would spawn a
  // process and, wherever it is absent from PATH, silently disable every
  // hook even though the binary exists.
  try {
    accessSync(bin, fsConstants.X_OK);
  } catch {
    return {};
  }

  async function runHook(mode, sessionId, filePath) {
    const payload = JSON.stringify({
      session_id: sessionId,
      tool_input: { file_path: filePath ?? "" },
    });
    try {
      const result = await $`${bin} hook ${mode} < ${Buffer.from(payload)}`
        .env({
          ...process.env,
          PULSARULES_PROJECT_DIR: root,
          PULSARULES_SKILLS_DIR: `${root}/.opencode/skills`,
          PULSARULES_LOG_PATH: `${root}/.opencode/hook-execution.log`,
        })
        .quiet();
      const text = result.stdout?.toString().trim();
      if (!text) return;
      const parsed = JSON.parse(text);
      return parsed?.hookSpecificOutput?.additionalContext;
    } catch (e) {
      // A hook must never break the user's turn, so a failed invocation is
      // logged and swallowed here - never rethrown.
      console.error("pulsarules_codex: hook", mode, "failed:", e);
    }
  }

  return {
    // Fires at the start of every turn, but that does NOT give the contract
    // per-turn presence: the "session-start" mode this calls is still gated
    // OncePerSession by the binary (internal/hook/hook.go's
    // emitSessionStart), so the contract text is actually injected only on
    // this session's first request - same once-per-session behavior as
    // Claude Code's SessionStart hook, just reached from a hook that happens
    // to fire every turn. "user-prompt" carries the per-turn digest instead
    // (internal/hook/emit_turn.go's emitUserPrompt has no OncePerSession
    // gate), so it is called on every invocation here, matching Claude
    // Code's UserPromptSubmit firing every turn.
    "experimental.chat.system.transform": async (input, output) => {
      // A second opencode call site triggers this hook with no sessionID at
      // all; without this guard that decodes to the shared "nosession"
      // marker (NewSessionTrackerFromID's empty-string fallback) and burns
      // it globally instead of leaving this call a no-op.
      if (!input.sessionID) return;
      const contract = await runHook("session-start", input.sessionID);
      if (contract) output.system.push(contract);
      const digest = await runHook("user-prompt", input.sessionID);
      if (digest) output.system.push(digest);
    },

    // tool.execute.after's args live on input ({tool, sessionID, callID,
    // args}); the output parameter here is the tool RESULT ({title,
    // metadata, output}), not the args, so filePath is read from
    // input.args. The context is appended to output.output, the field the
    // model actually reads back. console.log would only reach opencode's
    // server log, never the conversation - the same "fires but never
    // reaches the model" bug this plugin exists to kill.
    "tool.execute.after": async (input, output) => {
      if (input.tool !== "write" && input.tool !== "edit") return;
      const ctx = await runHook("post-edit", input.sessionID, input.args?.filePath ?? "");
      if (!ctx) return;
      if (typeof output.output === "string") {
        output.output += `\n\n${ctx}`;
      } else {
        output.output = ctx;
      }
    },

    // simplification: tool.execute.before's output object is only {args} -
    // no field the model reads back - so a pre-edit nudge has nowhere to go
    // here, and calling the binary's pre-edit mode just to console.log the
    // result would be the same silently-inert bug this plugin exists to
    // kill. Ceiling: under opencode, the pre-edit reminder Claude Code's
    // PreToolUse gives before a Go/SQL write has no equivalent; the model
    // only sees the post-edit checklist after the fact. Upgrade path: if
    // opencode ever adds a context channel to tool.execute.before, wire
    // pre-edit through it the same way session-start and post-edit are
    // wired above - do not register a handler here until it does.

    // simplification: opencode has no hook that fires when a turn/session
    // goes idle (session.idle is a bus event, not a trigger()-ed hook name;
    // see the file-header note), so the governance/stop check - the
    // dirty-worktree nag Claude Code's Stop hook gives - has nothing to
    // attach to here. Ceiling: under opencode, a dirty worktree with
    // governance findings goes unreported until the user runs
    // pulsarules_cli governance by hand. Upgrade path: if opencode ever adds
    // an idle/turn-finished entry to the triggered hook-name list, wire a
    // handler here calling the binary's stop mode the same way runHook is
    // used above.

    // simplification: opencode's trigger()-ed hook list (see the file-header
    // note) has no subagent-spawn event, so a spawned opencode subtask has no
    // call site to run the binary's subagent-start mode from - unlike Claude
    // Code, which fires SubagentStart. Ceiling: under opencode, a subtask
    // starts with no contract injected at all. Upgrade path: if opencode
    // ever adds a subagent/subtask-start entry to the triggered hook-name
    // list, wire a handler here calling runHook("subagent-start", ...) the
    // same way session-start is wired above.
  };
};
