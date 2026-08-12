// pulsarules_codex governance plugin for opencode.
export const PulsarulesGovernance = async ({ directory, worktree, $ }) => {
  const root = worktree || directory;
  const bin = root + "/.opencode/bin/pulsarules_cli";

  async function runHook(mode) {
    try {
      const result = await $([bin, "hook", mode]);
      const text = result.stdout?.toString().trim();
      if (!text) return;
      const parsed = JSON.parse(text);
      const ctx = parsed?.hookSpecificOutput?.additionalContext;
      if (ctx) return ctx;
    } catch (e) {}
  }

  return {
    "session.created": async () => {
      const ctx = await runHook("session-start");
      if (ctx) console.log(ctx);
    },

    "tool.execute.before": async (input) => {
      if (input.tool !== "write" && input.tool !== "edit") return;
      const ctx = await runHook("pre-edit");
      if (ctx) console.log(ctx);
    },

    "tool.execute.after": async (input) => {
      if (input.tool !== "write" && input.tool !== "edit") return;
      const ctx = await runHook("post-edit");
      if (ctx) console.log(ctx);
    },

    "session.idle": async () => {
      const ctx = await runHook("stop");
      if (ctx) console.log(ctx);
    },
  };
};
