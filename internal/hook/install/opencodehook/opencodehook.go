package opencodehook

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/fsperm"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/gitignore"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/selfbin"
)

// binaryRel is the path to the installer binary relative to the project root,
// placed there by InstallBinary.
const binaryRel = ".opencode/bin/pulsarules_cli"

// pluginName is the file written into .opencode/plugins/.
const pluginName = "pulsarules-governance.js"

// Install writes the governance plugin into <dir>/.opencode/plugins/ and
// copies the running binary into <dir>/.opencode/bin/. The plugin hooks into
// tool.execute.after, session.idle, and session.created events, forwarding each
// to the installer binary's hook command. It also ensures .opencode/.gitignore
// ignores the binary directory.
func Install(dir string) error {
	pluginsDir := filepath.Join(dir, ".opencode", "plugins")
	if err := os.MkdirAll(pluginsDir, fsperm.DirPrivate); err != nil {
		return fmt.Errorf("mkdir plugins: %w", err)
	}
	pluginPath := filepath.Join(pluginsDir, pluginName)
	if err := os.WriteFile(pluginPath, []byte(pluginScript), fsperm.FilePrivate); err != nil {
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

// InstallBinary copies the running installer binary into <dir>/.opencode/bin/
// so the plugin can invoke it. A copy failure degrades to a no-op plugin.
func InstallBinary(dir string) error {
	dst := filepath.Join(dir, ".opencode", "bin", "pulsarules_cli")
	if err := selfbin.Copy(dst); err != nil {
		return fmt.Errorf("copy installer binary: %w", err)
	}
	return nil
}

// pluginScript is the JavaScript plugin that opencode loads at startup. It
// subscribes to governance-relevant events and shells out to the installer
// binary, which produces the hookSpecificOutput JSON that opencode injects as
// additional context. The binary path is resolved relative to the project
// directory so the plugin works in worktrees too.
const pluginScript = `// pulsarules_codex governance plugin for opencode.
// Installed by pulsarules_cli; remove this file to disable.
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
