package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/ownedfile"
)

func TestWarnUnparseable(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		err         error
		wantHandled bool
	}{
		{
			name:        "unparseable config is a warning",
			err:         fmt.Errorf("strip hooks: %w", jsondoc.ErrUnparseable),
			wantHandled: true,
		},
		{name: "any other error is not handled", err: errors.New("disk full")},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var report Report
			if got := warnUnparseable(&report, testCase.err); got != testCase.wantHandled {
				t.Fatalf("handled = %v, want %v", got, testCase.wantHandled)
			}
			wantWarnings := 0
			if testCase.wantHandled {
				wantWarnings = 1
			}
			if len(report.Warnings) != wantWarnings {
				t.Errorf("Warnings = %q, want %d", report.Warnings, wantWarnings)
			}
		})
	}
}

func seedSkill(tb testing.TB, claudeDir, id, body string) {
	tb.Helper()
	dir := filepath.Join(claudeDir, "skills", id)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(body), 0o600); err != nil {
		tb.Fatalf("seed %q: %v", id, err)
	}
}

func seedShadowTree(
	tb testing.TB, projectIDs []string, userSkills map[string]string,
) (userDir, claudeDir string) {
	tb.Helper()
	userDir, claudeDir = tb.TempDir(), filepath.Join(tb.TempDir(), ".claude")
	for id, body := range userSkills {
		seedSkill(tb, userDir, id, body)
	}
	for _, id := range projectIDs {
		seedSkill(tb, claudeDir, id, "mine\n")
	}
	return userDir, claudeDir
}

func TestWarnShadowedSkills(t *testing.T) {
	t.Parallel()

	const precedence = "take precedence over this project's copies " +
		"(remove them with uninstall --global, or accept the global versions)"
	ours := "<!-- " + ownedfile.Marker + " -->\n"
	testCases := []struct {
		name       string
		projectIDs []string
		userSkills map[string]string
		want       string
	}{
		{
			name:       "our global copy is reported",
			projectIDs: []string{"go-style", "errors-logging"},
			userSkills: map[string]string{"go-style": ours, "errors-logging": "by hand\n"},
			want:       "1 of 2 installed skills also installed in %s " + precedence + ": go-style",
		},
		{
			name:       "every installed skill shadowed is one line",
			projectIDs: []string{"go-style", "errors-logging"},
			userSkills: map[string]string{"go-style": ours, "errors-logging": ours},
			want:       "all 2 installed skills are also installed in %s and " + precedence,
		},
		{
			name:       "the project copy of gopls-navigation counts when it exists",
			projectIDs: []string{"gopls-navigation"},
			userSkills: map[string]string{"gopls-navigation": ours},
			want:       "all 1 installed skills are also installed in %s and " + precedence,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userDir, claudeDir := seedShadowTree(t, testCase.projectIDs, testCase.userSkills)
			ctx := Context{UserClaudeDir: userDir, IDs: testCase.projectIDs}

			var report Report
			warnShadowedSkills(ctx, claudeDir, &report)
			want := []string{fmt.Sprintf(testCase.want, filepath.Join(userDir, "skills"))}
			if !slices.Equal(report.Warnings, want) {
				t.Errorf("Warnings = %q, want %q", report.Warnings, want)
			}
		})
	}
}

func TestWarnShadowedSkills_Silent(t *testing.T) {
	t.Parallel()

	ours := "<!-- " + ownedfile.Marker + " -->\n"
	testCases := []struct {
		name        string
		projectIDs  []string
		userSkills  map[string]string
		global      bool
		sameUserDir bool
	}{
		{
			name:       "a global gopls-navigation without a project copy is not shadowing",
			projectIDs: []string{"go-style"},
			userSkills: map[string]string{"gopls-navigation": ours},
		},
		{
			name:       "a hand-written global skill is not ours to report",
			projectIDs: []string{"errors-logging"},
			userSkills: map[string]string{"errors-logging": "by hand\n"},
		},
		{
			name:       "a global install has nothing to be shadowed by",
			projectIDs: []string{"go-style"},
			userSkills: map[string]string{"go-style": ours},
			global:     true,
		},
		{
			name:        "a project at the user dir is the global install",
			projectIDs:  []string{"go-style"},
			userSkills:  map[string]string{"go-style": ours},
			sameUserDir: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userDir, claudeDir := seedShadowTree(t, testCase.projectIDs, testCase.userSkills)
			if testCase.sameUserDir {
				claudeDir = userDir
			}
			ctx := Context{
				UserClaudeDir: userDir,
				IDs:           testCase.projectIDs,
				Global:        testCase.global,
			}

			var report Report
			warnShadowedSkills(ctx, claudeDir, &report)
			if len(report.Warnings) != 0 {
				t.Errorf("Warnings = %q, want none", report.Warnings)
			}
		})
	}
}

func TestWarnShadowedSkills_UnknownUserDir(t *testing.T) {
	t.Parallel()

	claudeDir := filepath.Join(t.TempDir(), ".claude")
	seedSkill(t, claudeDir, "go-style", "mine\n")

	var report Report
	warnShadowedSkills(Context{IDs: []string{"go-style"}}, claudeDir, &report)
	if len(report.Warnings) != 0 {
		t.Errorf("Warnings = %q, want none", report.Warnings)
	}
}

func TestWarnShadowedSkills_SymlinkedUserDirIsTheSameDir(t *testing.T) {
	t.Parallel()

	realHome := t.TempDir()
	linkHome := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, linkHome); err != nil {
		t.Skipf("symlinks are unavailable: %v", err)
	}
	seedSkill(t, filepath.Join(realHome, ".claude"), "go-style", "<!-- "+ownedfile.Marker+" -->\n")

	var report Report
	ctx := Context{UserClaudeDir: filepath.Join(realHome, ".claude"), IDs: []string{"go-style"}}
	warnShadowedSkills(ctx, filepath.Join(linkHome, ".claude"), &report)
	if len(report.Warnings) != 0 {
		t.Errorf(
			"Warnings = %q, want none for the same directory seen through a link",
			report.Warnings,
		)
	}
}

func TestWarnShadowedSkills_CapsTheList(t *testing.T) {
	t.Parallel()

	userDir, claudeDir := t.TempDir(), filepath.Join(t.TempDir(), ".claude")
	ids := make([]string, 0, 13)
	for idx := range 13 {
		id := fmt.Sprintf("skill-%02d", idx)
		ids = append(ids, id)
		seedSkill(t, claudeDir, id, "mine\n")
		if idx < 12 {
			seedSkill(t, userDir, id, "<!-- "+ownedfile.Marker+" -->\n")
		}
	}

	var report Report
	warnShadowedSkills(Context{UserClaudeDir: userDir, IDs: ids}, claudeDir, &report)
	if len(report.Warnings) != 1 {
		t.Fatalf("Warnings = %q, want one", report.Warnings)
	}
	warning := report.Warnings[0]
	for _, want := range []string{"12 of 13 installed skills", "skill-09", " +2 more"} {
		if !strings.Contains(warning, want) {
			t.Errorf("warning %q lacks %q", warning, want)
		}
	}
	if strings.Contains(warning, "skill-10") {
		t.Errorf("warning %q lists past the cap", warning)
	}
}

func TestWarnShadowedSkills_NeverFailsOnOddEntries(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		corrupt func(tb testing.TB, userSkills string)
	}{
		{
			name: "a plain file where the skill directory belongs",
			corrupt: func(tb testing.TB, userSkills string) {
				tb.Helper()
				seedFile(tb, filepath.Join(userSkills, "go-style"), "not a directory")
			},
		},
		{
			name: "an unreadable SKILL.md",
			corrupt: func(tb testing.TB, userSkills string) {
				tb.Helper()
				if runtime.GOOS == "windows" || os.Geteuid() == 0 {
					tb.Skip("file modes do not deny this reader")
				}
				path := filepath.Join(userSkills, "go-style", "SKILL.md")
				seedSkill(tb, filepath.Dir(userSkills), "go-style", "x")
				if err := os.Chmod(path, 0); err != nil {
					tb.Fatalf("chmod: %v", err)
				}
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			userDir, claudeDir := t.TempDir(), filepath.Join(t.TempDir(), ".claude")
			seedSkill(t, claudeDir, "go-style", "mine\n")
			userSkills := filepath.Join(userDir, "skills")
			if err := os.MkdirAll(userSkills, 0o750); err != nil {
				t.Fatalf("mkdir: %v", err)
			}
			testCase.corrupt(t, userSkills)

			var report Report
			warnShadowedSkills(
				Context{UserClaudeDir: userDir, IDs: []string{"go-style"}},
				claudeDir,
				&report,
			)
			want := fmt.Sprintf("could not check 1 skills in %s for shadowing; skipped", userSkills)
			if !slices.Equal(report.Warnings, []string{want}) {
				t.Errorf("Warnings = %q, want [%q]", report.Warnings, want)
			}
		})
	}
}
