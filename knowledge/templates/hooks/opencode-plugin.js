import { accessSync, constants as fsConstants } from "node:fs";

function makeRunHook({ bin, root, $ }) {
  return async function runHook(mode, sessionId, filePath) {
    // The JSON payload shape (session_id, tool_input.file_path) must match
    // hookPayload in internal/hook/wire.go's decodeHookPayload.
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
    } catch (error) {
      // A hook must never break the user's turn, so a failed invocation is
      // logged and swallowed here - never rethrown.
      console.error("pulsarules_codex: hook", mode, "failed:", error);
    }
  };
}

// simplification: pre-edit stays unwired. tool.execute.before's output object is only
// {args}: no field the model reads back, so calling pre-edit there would be the same
// silently-inert bug this plugin exists to kill; register no handler until it has one.
// stop and subagent-start have no opencode hook either (bus events never fire here).
// Ceiling: no pre-edit reminder, no dirty-worktree nag, subtasks start with no contract.
// Revisit when opencode adds a hook whose output the model reads back.
export const PulsarulesGovernance = async ({ directory, worktree, $ }) => {
  const root = worktree || directory;
  // Must match opencodehook.go's binDirRel and selfbin.Name.
  const bin = `${root}/.opencode/bin/pulsarules_cli`;

  // .opencode/bin/ is gitignored, so a fresh clone has no binary yet; stay
  // silent instead of shelling out to a path that may not exist.
  try {
    accessSync(bin, fsConstants.X_OK);
  } catch {
    return {};
  }

  const runHook = makeRunHook({ bin, root, $ });

  return {
    // "session-start" is OncePerSession-gated in the binary despite firing
    // every turn here; "user-prompt" has no such gate.
    "experimental.chat.system.transform": async (input, output) => {
      // Some opencode call sites invoke this hook with no sessionID; without
      // this guard it falls into the shared "nosession" tracker key globally.
      if (!input.sessionID) return;
      const contract = await runHook("session-start", input.sessionID);
      if (contract) output.system.push(contract);
      const nudge = await runHook("user-prompt", input.sessionID);
      if (nudge) output.system.push(nudge);
    },

    // output here is the tool result ({title, metadata, output}), not args;
    // filePath comes from input.args. output.output is what the model reads
    // back - console.log would only reach the server log.
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
  };
};

// Installed by pulsarules_cli. Uninstall removes only files carrying this marker.
