package install

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/host/codex"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func readCodexFile(tb testing.TB, path string) string {
	tb.Helper()
	body, err := os.ReadFile(path) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	return string(body)
}

func TestCodex_ReinstallKeepsTheBytesAndTheUsersHooks(t *testing.T) {
	t.Parallel()

	codexDir := filepath.Join(t.TempDir(), codex.RootDir)
	hooksFile := filepath.Join(codexDir, codex.HooksFile)
	const mine = `{"hooks":{"Stop":[{"hooks":[{"type":"command","command":"my-stop.sh"}]}]}}`
	if err := os.MkdirAll(codexDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hooksFile, []byte(mine), 0o600); err != nil {
		t.Fatal(err)
	}
	installed, err := Codex(codexDir, testTemplates(t), codex.Location{})
	if err != nil {
		t.Fatalf("Codex: %v", err)
	}
	if want := "wired hooks into " + hooksFile; !slices.Contains(installed.Notes, want) {
		t.Errorf("Notes = %v, want %q", installed.Notes, want)
	}
	first := readCodexFile(t, hooksFile)
	if _, err = Codex(codexDir, testTemplates(t), codex.Location{}); err != nil {
		t.Fatalf("second Codex: %v", err)
	}
	if again := readCodexFile(t, hooksFile); again != first {
		t.Errorf("reinstall changed hooks.json:\n%s\nvs\n%s", first, again)
	}
	assertCodexGitignore(t, codexDir)

	removed, err := UninstallCodex(codexDir)
	if err != nil {
		t.Fatalf("UninstallCodex: %v", err)
	}
	if want := "removed hook wiring from " + hooksFile; !slices.Contains(removed.Notes, want) {
		t.Errorf("Notes = %v, want %q", removed.Notes, want)
	}
	assertOnlyUserHooks(t, codexDir)
}

func assertCodexGitignore(tb testing.TB, codexDir string) {
	tb.Helper()
	got := readCodexFile(tb, filepath.Join(codexDir, ".gitignore"))
	if !strings.Contains(got, "/bin/\n/hook-execution.log\n") || strings.Contains(got, "/hooks/") {
		tb.Errorf(".gitignore = %q, want /bin/ and the log but never the committed hooks", got)
	}
}

func assertOnlyUserHooks(tb testing.TB, codexDir string) {
	tb.Helper()
	got := readCodexFile(tb, filepath.Join(codexDir, codex.HooksFile))
	if !strings.Contains(got, "my-stop.sh") || strings.Contains(got, "skill-router-reminder") {
		tb.Errorf("uninstall must leave only the user's hooks:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(codexDir, "hooks")); !errors.Is(err, fs.ErrNotExist) {
		tb.Errorf("uninstall left the hooks dir, stat err = %v", err)
	}
}

func TestCodex_GlobalWritesNoGitignoreAndAnAbsoluteCommand(t *testing.T) {
	t.Parallel()

	codexDir := filepath.Join(t.TempDir(), "codex-home")
	loc := codex.Location{GlobalDir: codexDir, UserSkills: filepath.Join(t.TempDir(), "skills")}
	if _, err := Codex(codexDir, testTemplates(t), loc); err != nil {
		t.Fatalf("Codex: %v", err)
	}
	hooks := readCodexFile(t, filepath.Join(codexDir, codex.HooksFile))
	if !strings.Contains(hooks, filepath.Join(codexDir, "hooks", "skill-router-reminder.sh")) {
		t.Errorf("hooks.json does not name the installed script:\n%s", hooks)
	}
	if _, statErr := os.Stat(
		filepath.Join(codexDir, ".gitignore"),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("global install wrote a .gitignore, stat err = %v", statErr)
	}
	if _, err := UninstallCodex(codexDir); err != nil {
		t.Fatalf("UninstallCodex: %v", err)
	}
	if _, statErr := os.Stat(
		filepath.Join(codexDir, codex.HooksFile),
	); !errors.Is(
		statErr,
		fs.ErrNotExist,
	) {
		t.Errorf("uninstall left hooks.json, stat err = %v", statErr)
	}
}

func TestCodex_UnparseableHooksFile(t *testing.T) {
	t.Parallel()

	codexDir := t.TempDir()
	if err := os.WriteFile(
		filepath.Join(codexDir, codex.HooksFile),
		[]byte(`[1]`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	_, installErr := Codex(codexDir, testTemplates(t), codex.Location{})
	if !errors.Is(installErr, jsondoc.ErrUnparseable) ||
		!strings.HasPrefix(installErr.Error(), "wire codex hooks: ") {
		t.Errorf("Codex err = %v, want a wrapped ErrUnparseable", installErr)
	}
	_, uninstallErr := UninstallCodex(codexDir)
	if !errors.Is(uninstallErr, jsondoc.ErrUnparseable) ||
		!strings.HasPrefix(uninstallErr.Error(), "unwire codex hooks: ") {
		t.Errorf("UninstallCodex err = %v, want a wrapped ErrUnparseable", uninstallErr)
	}
}

const sharedHooks = `{"hooks":{}}`

func linkedHooksDir(tb testing.TB) (codexDir, link, target string) {
	tb.Helper()
	codexDir = filepath.Join(tb.TempDir(), codex.RootDir)
	if err := os.MkdirAll(codexDir, 0o750); err != nil {
		tb.Fatal(err)
	}
	target = filepath.Join(tb.TempDir(), "shared-hooks.json")
	if err := os.WriteFile(target, []byte(sharedHooks), 0o600); err != nil {
		tb.Fatal(err)
	}
	link = filepath.Join(codexDir, codex.HooksFile)
	if err := os.Symlink(target, link); err != nil {
		tb.Fatal(err)
	}
	return codexDir, link, target
}

func assertLeftAlone(tb testing.TB, warnings []string, mustMention string) {
	tb.Helper()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "left it alone") ||
		!strings.Contains(warnings[0], mustMention) {
		tb.Errorf("Warnings = %q, want one saying it was left alone and %q", warnings, mustMention)
	}
}

func TestCodex_SymlinkedHooksFileIsLeftAlone(t *testing.T) {
	t.Parallel()

	codexDir, link, target := linkedHooksDir(t)
	installed, err := Codex(codexDir, testTemplates(t), codex.Location{})
	if err != nil {
		t.Fatalf("Codex must not abort over a symlinked hooks file: %v", err)
	}
	assertLeftAlone(t, installed.Warnings, "install --print-hooks --target codex --project ")
	if _, err = os.Stat(filepath.Join(codexDir, "hooks", "skill-router-reminder.sh")); err != nil {
		t.Errorf("the script must still install: %v", err)
	}
	removed, err := UninstallCodex(codexDir)
	if err != nil {
		t.Fatalf("UninstallCodex must not abort over a symlinked hooks file: %v", err)
	}
	assertLeftAlone(t, removed.Warnings, "Remove our hook entries")
	if stat, statErr := os.Lstat(link); statErr != nil || stat.Mode()&os.ModeSymlink == 0 {
		t.Errorf("the symlink must survive, got %v, %v", stat, statErr)
	}
	if got := readCodexFile(t, target); got != sharedHooks {
		t.Errorf("the target = %q, want it untouched", got)
	}
}
