package opencode

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs/gitignore"
)

const binaryRel = ".opencode/bin/pulsarules_cli"

const pluginName = "pulsarules-governance.js"

func Install(dir string) error {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, disk.DirPrivate); err != nil {
		return fmt.Errorf("mkdir plugins: %w", err)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	if err := os.WriteFile(pluginPath, []byte(pluginScript), disk.FilePrivate); err != nil {
		return fmt.Errorf("write plugin: %w", err)
	}
	if err := InstallBinary(dir); err != nil {
		return fmt.Errorf("install binary: %w", err)
	}
	if err := gitignore.Ensure(filepath.Join(dir, ".opencode"), "/bin/"); err != nil {
		return fmt.Errorf("ensure opencode gitignore: %w", err)
	}
	return nil
}

func InstallBinary(dir string) error {
	dst := filepath.Join(dir, ".opencode", "bin", "pulsarules_cli")
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

const pluginScript = `// pulsarules_codex governance plugin for opencode.
export const PulsarulesGovernance = async ({ directory, worktree, $ }) => {
  const root = worktree || directory;
  const bin = root + "/` + binaryRel + `";

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
`
