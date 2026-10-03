package uninstall

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wrapped-owls/goremy-di/remy"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/bootstrap"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/marker"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/skill/target"
)

const forgedLine = "[ERROR] forged"

func forgedLines(text string) []string {
	var found []string
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "[ERROR]") {
			found = append(found, line)
		}
	}
	return found
}

func TestPrintReport_StaysOnOneLine(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		report     target.Report
		wantStdout string
		wantStderr string
	}{
		{
			name:       "note",
			report:     target.Report{Notes: []string{"removed: x\n" + forgedLine}},
			wantStdout: `removed: x\n[ERROR] forged` + "\n",
		},
		{
			name:       "warning",
			report:     target.Report{Warnings: []string{"bad key \"a\n" + forgedLine}},
			wantStderr: `bad key "a\n[ERROR] forged` + "\n",
		},
		{
			name:       "plain text is untouched",
			report:     target.Report{Notes: []string{"removed: a/b"}},
			wantStdout: "removed: a/b\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var stdout, stderr strings.Builder
			printReport(&stdout, &stderr, testCase.report)
			if stdout.String() != testCase.wantStdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), testCase.wantStdout)
			}
			if stderr.String() != testCase.wantStderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), testCase.wantStderr)
			}
		})
	}
}

func TestPrintReport_HostileTree(t *testing.T) {
	t.Parallel()

	projectDir := t.TempDir()
	skillDir := filepath.Join(projectDir, ".claude", "skills", "x\n"+forgedLine+" by repo")
	if err := os.MkdirAll(skillDir, 0o750); err != nil {
		t.Skipf("filesystem refuses the name: %v", err)
	}
	marked := []byte("<!-- " + marker.Installed + " -->\n")
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), marked, 0o600); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	settings := `{"hooks": {"PreToolUse\n` + forgedLine + ` via key": 5}}`
	settingsPath := filepath.Join(projectDir, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, []byte(settings), 0o600); err != nil {
		t.Fatalf("seed settings: %v", err)
	}

	inj := remy.NewInjector(remy.Config{DuckTypeElements: true})
	if err := bootstrap.DoInjections(inj, bootstrap.Options{ProjectDir: "."}); err != nil {
		t.Fatalf("DoInjections: %v", err)
	}
	deps, err := resolveUninstallCollaborators(inj)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	report, _ := deps.targets.Uninstall("claude", target.UninstallContext{
		Base:             projectDir,
		HookUninstallers: deps.hooks,
		SettingsFiles:    []string{"settings.json"},
	})

	var stdout, stderr strings.Builder
	printReport(&stdout, &stderr, report)
	if len(report.Notes) == 0 || len(report.Warnings) == 0 {
		t.Fatalf("want notes and warnings, got %+v", report)
	}
	streams := map[string]string{"stdout": stdout.String(), "stderr": stderr.String()}
	for name, text := range streams {
		if forged := forgedLines(text); len(forged) > 0 {
			t.Errorf("%s has forged lines %q in %q", name, forged, text)
		}
	}
}
