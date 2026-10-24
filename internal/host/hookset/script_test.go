package hookset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func TestInstall(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	if _, err := Install(fakeTemplates(), claudeDir, testValues()); err != nil {
		t.Fatalf("Install: %v", err)
	}

	script := filepath.Join(claudeDir, "hooks", "skill-router-reminder.sh")
	stat, err := os.Stat(script)
	if err != nil {
		t.Fatalf("stat script: %v", err)
	}
	if stat.Mode().Perm()&0o100 == 0 {
		t.Errorf("script not executable: mode %v", stat.Mode().Perm())
	}
	if _, err = os.Stat(filepath.Join(claudeDir, "hooks", "README.md")); err != nil {
		t.Errorf("README not installed: %v", err)
	}
}

func TestInstall_ForeignReadmeBackup(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")
	if err := os.MkdirAll(hooksDir, 0o750); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	readme := filepath.Join(hooksDir, "README.md")
	foreign := "# My own notes about this hooks dir\n"
	if err := os.WriteFile(readme, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign readme: %v", err)
	}

	ledger, err := Install(fakeTemplates(), claudeDir, testValues())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	backupPath := readme + ownedfile.BackupSuffix
	wantMsg := ownedfile.BackupMessage(readme, backupPath)
	if len(ledger.BackedUp) != 1 || ledger.BackedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", ledger.BackedUp, wantMsg)
	}
	got, readErr := os.ReadFile(backupPath) //nolint:gosec // test fixture.
	if readErr != nil {
		t.Fatalf("read backup: %v", readErr)
	}
	if string(got) != foreign {
		t.Errorf("backup content = %q, want %q", got, foreign)
	}

	secondLedger, err := Install(fakeTemplates(), claudeDir, testValues())
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if len(secondLedger.BackedUp) != 0 {
		t.Errorf(
			"second Install backedUp = %v, want none (it owns the file now)",
			secondLedger.BackedUp,
		)
	}
}

func seedFile(tb testing.TB, path, content string) {
	tb.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		tb.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", path, err)
	}
}

func assertContent(tb testing.TB, path, want string) {
	tb.Helper()

	got, err := os.ReadFile(path) //nolint:gosec // test fixture.
	if err != nil {
		tb.Fatalf("read %q: %v", path, err)
	}
	if string(got) != want {
		tb.Errorf("%q content = %q, want %q", path, got, want)
	}
}

func TestInstall_NextNumberedSlot(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	readme := filepath.Join(claudeDir, "hooks", "README.md")
	current := "# current notes\n"
	seedFile(t, readme, current)
	priorBackup := readme + ownedfile.BackupSuffix
	priorContent := "# an even earlier version of the user's notes\n"
	seedFile(t, priorBackup, priorContent)

	ledger, err := Install(fakeTemplates(), claudeDir, testValues())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	newBackup := readme + ownedfile.BackupSuffix + ".1"
	wantMsg := ownedfile.BackupMessage(readme, newBackup)
	if len(ledger.BackedUp) != 1 || ledger.BackedUp[0] != wantMsg {
		t.Fatalf("backedUp = %v, want [%q]", ledger.BackedUp, wantMsg)
	}
	assertContent(t, priorBackup, priorContent)
	assertContent(t, newBackup, current)
}

func Test_renderScript(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		body    string
		values  ScriptValues
		want    string
		wantErr bool
	}{
		{
			name:   "substitutes every path",
			body:   `bin={{.Bin}} skills={{.SkillsDirs}} log={{.Log}} root={{.ProjectRoot}}`,
			values: ScriptValues{Bin: "b", SkillsDirs: "s", Log: "l", ProjectRoot: "r"},
			want:   `bin=b skills=s log=l root=r`,
		},
		{
			name:   "defer block renders only when set",
			body:   `{{if .DeferTo}}defer={{.DeferTo}}{{end}}`,
			values: ScriptValues{},
			want:   ``,
		},
		{
			name:   "defer block renders its value",
			body:   `{{if .DeferTo}}defer={{.DeferTo}}{{end}}`,
			values: ScriptValues{DeferTo: "d"},
			want:   `defer=d`,
		},
		{
			name:    "malformed template fails to parse",
			body:    `{{.Bin`,
			wantErr: true,
		},
		{
			name:    "unknown field fails to render",
			body:    `{{.Missing}}`,
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, renderErr := renderScript("t", []byte(testCase.body), testCase.values)
			if testCase.wantErr {
				if renderErr == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if renderErr != nil {
				t.Fatalf("renderScript: %v", renderErr)
			}
			if string(got) != testCase.want {
				t.Errorf("got %q, want %q", got, testCase.want)
			}
		})
	}
}
