package codex

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/hookset"
	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func installStubbed(tb testing.TB, codexDir string, loc Location) {
	tb.Helper()
	_, templates, err := knowledge.Load("")
	if err != nil {
		tb.Fatalf("load knowledge: %v", err)
	}
	if _, err = InstallHook(templates, codexDir, loc); err != nil {
		tb.Fatalf("InstallHook: %v", err)
	}
	const stubBinary = "#!/bin/sh\nprintf '%s\\n' \"$@\" \"$PULSARULES_PROJECT_DIR\" " +
		"\"$PULSARULES_SKILLS_DIR\" \"$PULSARULES_LOG_PATH\" \"$PULSARULES_HOOK_DIALECT\" " +
		"> \"$RECORD\"\ncat >> \"$RECORD\"\n"
	binary := filepath.Join(codexDir, "bin", "pulsarules_cli")
	if err = os.WriteFile(
		binary,
		[]byte(stubBinary),
		0o700,
	); err != nil { //nolint:gosec // test stub.
		tb.Fatalf("write stub binary: %v", err)
	}
}

func runShell(tb testing.TB, args []string, workDir, record string) {
	tb.Helper()
	ctx, cancel := context.WithTimeout(tb.Context(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) //nolint:gosec // test command.
	cmd.Dir = workDir
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=/home/tester", "RECORD=" + record,
		"CLAUDE_PROJECT_DIR=/must/not/matter",
	}
	cmd.Stdin = strings.NewReader("payload\n")
	if out, err := cmd.CombinedOutput(); err != nil || len(out) > 0 {
		tb.Fatalf("%v: err=%v output=%q, want a quiet success", args, err, out)
	}
}

func readRecord(tb testing.TB, record string) string {
	tb.Helper()
	got, err := os.ReadFile(record) //nolint:gosec // test fixture.
	if err != nil {
		return ""
	}
	return string(got)
}

func TestInstallHook_RealTemplate(t *testing.T) {
	t.Parallel()

	_, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("load knowledge: %v", err)
	}
	codexDir := filepath.Join(t.TempDir(), RootDir)
	if _, err = InstallHook(templates, codexDir, Location{}); err != nil {
		t.Fatalf("InstallHook: %v", err)
	}
	got, err := os.ReadFile( //nolint:gosec // test fixture.
		filepath.Join(codexDir, hookset.HooksSubdir, hookset.ScriptName),
	)
	if err != nil {
		t.Fatalf("read rendered script: %v", err)
	}
	script := string(got)
	if strings.Contains(script, "{{") || strings.Contains(script, "git rev-parse") {
		t.Errorf("rendered script has a placeholder or walks through git:\n%s", script)
	}
	for _, want := range []string{
		`bin="$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)/bin/pulsarules_cli"`,
		`export PULSARULES_PROJECT_DIR="$(CDPATH= cd -- "$(dirname "$0")/../.." && pwd)"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("rendered script missing %q:\n%s", want, script)
		}
	}
}

//nolint:paralleltest // an exec racing a concurrent stub write fails with ETXTBSY.
func TestProjectScript_FindsItsRootFromItsOwnPath(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is not available")
	}
	testCases := []struct {
		name    string
		workDir func(root string) string
	}{
		{"from the root", func(root string) string { return root }},
		{
			"from a nested directory",
			func(root string) string { return filepath.Join(root, "a", "b") },
		},
		{"from outside the project", func(string) string { return os.TempDir() }},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "a", "b"), 0o750); err != nil {
				t.Fatal(err)
			}
			installStubbed(t, filepath.Join(root, RootDir), Location{})
			record := filepath.Join(t.TempDir(), "record")
			script := filepath.Join(root, RootDir, "hooks", hookset.ScriptName)
			runShell(t, []string{"bash", script, "post-edit"}, testCase.workDir(root), record)

			want := "hook\npost-edit\n" + root + "\n" + root + "/.agents/skills:/home/tester/.agents/skills\n" +
				root + "/.codex/hook-execution.log\n\npayload\n"
			if got := readRecord(t, record); got != want {
				t.Errorf("record = %q, want %q", got, want)
			}
		})
	}
}

func TestWireHooks_ThenUnwire(t *testing.T) {
	t.Parallel()

	codexDir := filepath.Join(t.TempDir(), RootDir)
	if err := WireHooks(codexDir, Location{}); err != nil {
		t.Fatalf("WireHooks: %v", err)
	}
	if _, err := os.Stat(filepath.Join(codexDir, HooksFile)); err != nil {
		t.Fatalf("WireHooks wrote no hooks file: %v", err)
	}
	changed, err := UnwireHooks(codexDir)
	if err != nil || !changed {
		t.Fatalf("UnwireHooks = %v, %v, want a change", changed, err)
	}
	if _, err = os.Stat(filepath.Join(codexDir, HooksFile)); !os.IsNotExist(err) {
		t.Errorf("an emptied hooks file must go, stat err = %v", err)
	}
	if changed, err = UnwireHooks(codexDir); err != nil || changed {
		t.Errorf("second UnwireHooks = %v, %v, want a quiet no-op", changed, err)
	}
}

func TestUninstallHook_RemovesTheScriptAndBinary(t *testing.T) {
	t.Parallel()

	codexDir := filepath.Join(t.TempDir(), RootDir)
	installStubbed(t, codexDir, Location{})
	if _, err := UninstallHook(codexDir); err != nil {
		t.Fatalf("UninstallHook: %v", err)
	}
	if entries, err := os.ReadDir(codexDir); err != nil || len(entries) != 0 {
		t.Errorf("codex dir after UninstallHook = %v, %v, want it empty", entries, err)
	}
}

func TestHooksFile_SymlinkIsRefused(t *testing.T) {
	t.Parallel()

	codexDir := filepath.Join(t.TempDir(), RootDir)
	if err := os.MkdirAll(codexDir, 0o750); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "shared.json")
	if err := os.WriteFile(target, []byte(`{"hooks":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(codexDir, HooksFile)); err != nil {
		t.Fatal(err)
	}
	if err := WireHooks(codexDir, Location{}); !errors.Is(err, ErrSymlink) {
		t.Errorf("WireHooks err = %v, want ErrSymlink", err)
	}
	if _, err := UnwireHooks(codexDir); !errors.Is(err, ErrSymlink) {
		t.Errorf("UnwireHooks err = %v, want ErrSymlink", err)
	}
	if got := readRecord(t, target); got != `{"hooks":{}}` {
		t.Errorf("the target = %q, want it untouched", got)
	}
}
