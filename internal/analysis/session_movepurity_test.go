package analysis

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func writeCommittedFile(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFileAt(t, dir, name, content)
	runGitAt(t, dir, "add", name)
	commitAllowEmpty(t, dir, ":seedling: feat: Seed "+name)
}

func stageRename(t *testing.T, dir, newContent string) {
	t.Helper()
	if err := os.Remove(filepath.Join(dir, "old.go")); err != nil {
		t.Fatalf("remove old.go: %v", err)
	}
	runGitAt(t, dir, "add", "old.go")
	writeFileAt(t, dir, "new.go", newContent)
	runGitAt(t, dir, "add", "new.go")
}

func stageModify(t *testing.T, dir, name, content string) {
	t.Helper()
	writeFileAt(t, dir, name, content)
	runGitAt(t, dir, "add", name)
}

func writeFileAt(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func runGitAt(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

type movePurityFixture struct {
	name        string
	setup       func(t *testing.T, dir string)
	wantFinding bool
	wantSubstr  string
}

func movePurityCases() []movePurityFixture {
	return []movePurityFixture{
		{
			name: "pure rename alone",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				body := "line one\nline two\nline three\nline four\n"
				writeCommittedFile(t, dir, "old.go", body)
				stageRename(t, dir, body)
			},
			wantFinding: false,
		},
		{
			name: "partial rename below the default threshold",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeCommittedFile(t, dir, "old.go", "a\nb\nc\nd\n")
				stageRename(t, dir, "a\nb\nc\nz\n")
			},
			wantFinding: true,
			wantSubstr:  "not a pure move",
		},
		{
			name: "pure rename staged alongside an unrelated edit",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				body := "line one\nline two\nline three\nline four\n"
				writeCommittedFile(t, dir, "old.go", body)
				writeCommittedFile(t, dir, "other.go", "package other\n")
				stageRename(t, dir, body)
				stageModify(t, dir, "other.go", "package other\n\nvar x = 1\n")
			},
			wantFinding: true,
			wantSubstr:  "mixes",
		},
		{
			name: "no staged renames",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				writeCommittedFile(t, dir, "plain.go", "package plain\n")
			},
			wantFinding: false,
		},
		{
			name: "pure rename staged alongside a pure import-path edit",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				body := "line one\nline two\nline three\nline four\n"
				writeCommittedFile(t, dir, "old.go", body)
				writeCommittedFile(
					t,
					dir,
					"importer.go",
					"package importer\n\nimport (\n\t\"repo/old/thing\"\n)\n\n"+
						"func Use() { thing.Do() }\n",
				)
				stageRename(t, dir, body)
				stageModify(
					t,
					dir,
					"importer.go",
					"package importer\n\nimport (\n\t\"repo/new/thing\"\n)\n\n"+
						"func Use() { thing.Do() }\n",
				)
			},
			wantFinding: false,
		},
		{
			name: "rename whose own file also gained a statement",
			setup: func(t *testing.T, dir string) {
				t.Helper()
				original := strings.Repeat("same line\n", 20)
				writeCommittedFile(t, dir, "old.go", original)
				stageRename(t, dir, original+"var extra = 1\n")
			},
			wantFinding: true,
			wantSubstr:  "carries an edit beyond the move",
		},
	}
}

func TestSession_MovePurity_StagedRename(t *testing.T) {
	t.Parallel()

	cfg := config.Defaults()
	cfg.ApplyPreset()

	for _, testCase := range movePurityCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			gitInitRepo(t, dir)
			testCase.setup(t, dir)

			repo, err := vcs.Open(dir)
			if err != nil {
				t.Fatalf("vcs.Open: %v", err)
			}

			sess := NewSession(repo, "", nil, cfg)
			findings := sess.Analyze(ScopeCommit, nil, FileSetChanged).Findings

			found := false
			for _, finding := range findings {
				if finding.AnalyzerID != "commit-move-purity" {
					continue
				}
				found = true
				if testCase.wantSubstr != "" &&
					!strings.Contains(finding.Message, testCase.wantSubstr) {
					t.Errorf(
						"message = %q, want substring %q",
						finding.Message,
						testCase.wantSubstr,
					)
				}
			}
			if found != testCase.wantFinding {
				t.Fatalf(
					"commit-move-purity finding present = %v, want %v; findings = %+v",
					found, testCase.wantFinding, findings,
				)
			}
		})
	}
}
