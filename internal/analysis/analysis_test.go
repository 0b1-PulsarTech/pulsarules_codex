package analysis

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func gitInitRepo(t *testing.T, dir string) {
	t.Helper()
	ctx := context.Background()
	if out, err := exec.CommandContext(ctx, "git", "-C", dir, "init").
		CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.email", "test@test.com").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
	if err := exec.CommandContext(ctx, "git", "-C", dir, "config", "user.name", "Test").
		Run(); err != nil {
		t.Fatalf("git config: %v", err)
	}
}

func commitAllowEmpty(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.CommandContext(
		t.Context(),
		"git",
		"-C",
		dir,
		"commit",
		"-q",
		"--allow-empty",
		"-m",
		msg,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func commitAllowEmptyAt(t *testing.T, dir, msg, authorDate string) {
	t.Helper()
	cmd := exec.CommandContext(
		t.Context(),
		"git",
		"-C",
		dir,
		"commit",
		"-q",
		"--allow-empty",
		"-m",
		msg,
	)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_DATE="+authorDate, "GIT_COMMITTER_DATE="+authorDate)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

type noListRepo struct{ vcs.Repository }

type failingListRepo struct{ *goldenRepo }

func (failingListRepo) ListFiles() ([]string, error) { return nil, errors.New("git unavailable") }

type listedRepo struct {
	*goldenRepo
	paths []string
}

func (repo listedRepo) ListFiles() ([]string, error) { return repo.paths, nil }

func skipFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	body := "package demo\n\n// bad note \u2014 trips typographic-markers\n"
	for _, name := range []string{
		"main.go", "target/x.go", "build/y.go", "thirdparty/z.go", "vendor/v.go", "ignored.go",
	} {
		writeFileAtDir(t, dir, name, body)
	}
	return dir
}

func reportedFiles(session *Session) map[string]bool {
	seen := map[string]bool{}
	for _, finding := range session.Analyze(ScopeChanged, nil, FileSetAll).Findings {
		if finding.AnalyzerID == "typographic-markers" {
			seen[finding.File] = true
		}
	}
	return seen
}

type skipDirsCase struct {
	name string
	repo func(root string) vcs.Repository
	want map[string]bool
}

func skipDirsGitModeCases() []skipDirsCase {
	return []skipDirsCase{
		{
			name: "listed repo sees target and build but honours the configured skip",
			repo: func(root string) vcs.Repository { return &goldenRepo{root: root} },
			want: map[string]bool{
				"main.go": true, "target/x.go": true, "build/y.go": true,
				"thirdparty/z.go": false, "vendor/v.go": false, "ignored.go": true,
			},
		},
		{
			name: "listed repo drops what git ignores",
			repo: func(root string) vcs.Repository {
				paths := []string{"main.go", "target/x.go"}
				return listedRepo{goldenRepo: &goldenRepo{root: root}, paths: paths}
			},
			want: map[string]bool{"main.go": true, "target/x.go": true, "ignored.go": false},
		},
		{
			name: "repo without ListFiles falls back to the full skip list",
			repo: func(root string) vcs.Repository { return noListRepo{&goldenRepo{root: root}} },
			want: map[string]bool{
				"main.go": true, "target/x.go": false, "build/y.go": false,
				"thirdparty/z.go": false, "vendor/v.go": false, "ignored.go": true,
			},
		},
		{
			name: "failing ListFiles falls back to the full skip list",
			repo: func(root string) vcs.Repository {
				return failingListRepo{&goldenRepo{root: root}}
			},
			want: map[string]bool{
				"main.go": true, "target/x.go": false, "build/y.go": false,
				"thirdparty/z.go": false, "vendor/v.go": false, "ignored.go": true,
			},
		},
	}
}

func TestSession_SkipDirs_GitModeKeepsGenericNames(t *testing.T) {
	t.Parallel()

	testCases := skipDirsGitModeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			cfg := config.Defaults()
			cfg.ApplyPreset()
			cfg.SkipDirs = []string{"thirdparty"}

			seen := reportedFiles(NewSession(testCase.repo(skipFixture(t)), "", nil, cfg))
			for file, want := range testCase.want {
				if seen[file] != want {
					t.Errorf("%s reported = %v, want %v (seen %v)", file, seen[file], want, seen)
				}
			}
		})
	}
}

func TestSession_Discover_ListingWarning(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		repo     func(root string) vcs.Repository
		wantWarn int
	}{
		{"listing works", func(root string) vcs.Repository { return &goldenRepo{root: root} }, 0},
		{
			"repo cannot list",
			func(root string) vcs.Repository { return noListRepo{&goldenRepo{root: root}} },
			1,
		},
		{
			"listing fails",
			func(root string) vcs.Repository { return failingListRepo{&goldenRepo{root: root}} },
			1,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var logs bytes.Buffer
			session := NewSession(testCase.repo(skipFixture(t)), "", nil, nil)
			session.logger = slog.New(slog.NewTextHandler(&logs, nil))
			session.Discover(ScopeChanged, nil, FileSetAll)

			if got := bytes.Count(logs.Bytes(), []byte("level=WARN")); got != testCase.wantWarn {
				t.Errorf("warn lines = %d, want %d: %s", got, testCase.wantWarn, logs.String())
			}
		})
	}
}

func TestSession_SkipDirs_GitMode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)
	body := "package demo\n\n// bad note \u2014 trips typographic-markers\n"
	writeFileAtDir(t, dir, "target/x.go", body)
	writeFileAtDir(t, dir, "vendor/v.go", body)
	runGitAt(t, dir, "add", "-A")

	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}
	cfg := config.Defaults()
	cfg.ApplyPreset()

	findings := NewSession(repo, "", nil, cfg).Analyze(ScopeChanged, nil, FileSetChanged).Findings
	seen := map[string]bool{}
	for _, finding := range findings {
		seen[finding.File] = true
	}
	if !seen["target/x.go"] {
		t.Errorf("expected target/x.go to be analyzed in git mode, findings=%+v", findings)
	}
	if seen["vendor/v.go"] {
		t.Error("expected vendor/v.go to stay skipped in git mode")
	}
}

const emDashBody = "package demo\n\n// bad note \u2014 trips typographic-markers\n"

type countingRepo struct {
	*goldenRepo
	listCalls *int
}

func (repo countingRepo) ListFiles() ([]string, error) {
	*repo.listCalls++
	return repo.goldenRepo.ListFiles()
}

func typographicFiles(findings []core.Finding) map[string]bool {
	seen := map[string]bool{}
	for _, finding := range findings {
		if finding.AnalyzerID == "typographic-markers" {
			seen[finding.File] = true
		}
	}
	return seen
}

func TestSession_ScopeCommit_FileList(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	gitInitRepo(t, dir)
	writeFileAtDir(t, dir, "target/x.go", emDashBody)
	runGitAt(t, dir, "add", "-A")
	repo, err := vcs.Open(dir)
	if err != nil {
		t.Fatalf("vcs.Open: %v", err)
	}

	findings := NewSession(repo, "", nil, nil).Analyze(ScopeCommit, nil, FileSetChanged).Findings
	if !typographicFiles(findings)["target/x.go"] {
		t.Fatalf("expected the staged target/x.go to be analyzed, got %+v", findings)
	}
}

func TestSession_ScopeCommit_OnlyStagedChangesGate(t *testing.T) {
	t.Parallel()

	status := &vcs.Status{Changes: []vcs.Change{
		{Path: "a.go", Extension: ".go", Staged: true},
		{Path: "scratch/x.go", Extension: ".go"},
	}}
	testCases := []struct {
		name        string
		scope       Scope
		wantScratch bool
	}{
		{"commit ignores an untracked file", ScopeCommit, false},
		{"changed still reports it", ScopeChanged, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFileAtDir(t, dir, "a.go", "package demo\n")
			writeFileAtDir(t, dir, "scratch/x.go", emDashBody)

			session := NewSession(&goldenRepo{root: dir}, "", nil, nil)
			findings := session.Analyze(testCase.scope, status, FileSetChanged).Findings
			if got := typographicFiles(findings)["scratch/x.go"]; got != testCase.wantScratch {
				t.Fatalf("scratch/x.go reported = %v, want %v", got, testCase.wantScratch)
			}
		})
	}
}

func TestSession_Discover_ListsOnDemand(t *testing.T) {
	t.Parallel()

	staged := &vcs.Status{Changes: []vcs.Change{{Path: "a.go", Extension: ".go", Staged: true}}}
	testCases := []struct {
		name      string
		scope     Scope
		status    *vcs.Status
		wantCalls int
	}{
		{"commit-msg hook with an empty status", ScopeCommit, &vcs.Status{}, 0},
		{"commit with only unstaged changes", ScopeCommit, &vcs.Status{
			Changes: []vcs.Change{{Path: "a.go", Extension: ".go"}},
		}, 0},
		{"commit with a staged change", ScopeCommit, staged, 1},
		{"changed with an empty status still feeds arch", ScopeChanged, &vcs.Status{}, 1},
		{"pre-commit with nothing staged", ScopeCommit, nil, 0},
		{"changed reading the worktree itself", ScopeChanged, nil, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			dir := t.TempDir()
			writeFileAtDir(t, dir, "a.go", "package demo\n")
			repo := countingRepo{goldenRepo: &goldenRepo{root: dir}, listCalls: &calls}

			session := NewSession(repo, "", nil, nil)
			session.Discover(testCase.scope, testCase.status, FileSetChanged)
			if calls != testCase.wantCalls {
				t.Fatalf("ListFiles calls = %d, want %d", calls, testCase.wantCalls)
			}
		})
	}
}

func TestSession_FileSetChanged_Siblings(t *testing.T) {
	t.Parallel()

	status := &vcs.Status{Changes: []vcs.Change{{Path: "pkg/f0.go", Extension: ".go"}}}
	listed := []string{"pkg/f0.go", "pkg/f1.go", "pkg/f2.go"}
	testCases := []struct {
		name      string
		repo      func(root string) vcs.Repository
		wantCount bool
	}{
		{
			"gitignored siblings stay out of the count",
			func(root string) vcs.Repository {
				return listedRepo{goldenRepo: &goldenRepo{root: root}, paths: listed}
			},
			false,
		},
		{
			"a walk of the disk counts them",
			func(root string) vcs.Repository { return noListRepo{&goldenRepo{root: root}} },
			true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for fileNum := range 12 {
				writeFileAtDir(t, dir, fmt.Sprintf("pkg/f%d.go", fileNum), "package pkg\n")
			}

			session := NewSession(testCase.repo(dir), "", nil, nil)
			found := false
			report := session.Analyze(ScopeChanged, status, FileSetChanged)
			for _, finding := range report.Findings {
				found = found || finding.AnalyzerID == "package-file-count"
			}
			if found != testCase.wantCount {
				t.Fatalf("package-file-count reported = %v, want %v", found, testCase.wantCount)
			}
		})
	}
}

type statusFailingRepo struct{ *goldenRepo }

func (statusFailingRepo) WorktreeStatus() (vcs.Status, error) {
	return vcs.Status{}, errors.New("status exploded")
}

func TestDiscover_LogsWorktreeStatusFailure(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	sess := NewSession(statusFailingRepo{&goldenRepo{root: t.TempDir()}}, "", nil, nil)
	sess.logger = slog.New(slog.NewTextHandler(&logs, nil))

	discovery := sess.Discover(ScopeFull, nil, FileSetChanged)
	if len(discovery.ChangedFiles) != 0 {
		t.Fatalf("ChangedFiles = %v, want none", discovery.ChangedFiles)
	}
	if !strings.Contains(logs.String(), "status exploded") {
		t.Fatalf("log output %q does not report the status failure", logs.String())
	}
}
