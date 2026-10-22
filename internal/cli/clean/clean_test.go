package clean

import (
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

func openRepo(tb testing.TB, root string) *vcs.Repository {
	tb.Helper()
	repo, err := vcs.Open(root)
	if err != nil {
		tb.Fatalf("vcs.Open: %v", err)
	}
	return repo
}

func seedTree(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "testdata"), 0o750); err != nil {
		tb.Fatalf("mkdir: %v", err)
	}
	files := map[string]string{
		"notes.md":            "a\u200Bb and an em\u2014dash\n",
		"code.go":             "package a // zero\u200Bwidth\n",
		"held.go":             "package a\n\nvar s = \"x\u200By\"\n",
		"skip.txt":            "a\u200Bb\n",
		"testdata/fixture.md": "a\u200Bb\n",
	}
	for rel, body := range files {
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
			tb.Fatalf("seed %q: %v", rel, err)
		}
	}
	return root
}

func TestSweep_ReportOnlyWritesNothing(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	before, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var out strings.Builder
	if sweepErr := sweep(&out, root, inspecting(), nil); sweepErr != nil {
		t.Fatalf("sweep: %v", sweepErr)
	}
	after, err := os.ReadFile(filepath.Join(root, "notes.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(after) != string(before) {
		t.Errorf("report-only rewrote the file")
	}
	if !strings.Contains(out.String(), "re-run with --write") {
		t.Errorf("output = %q, want it to name the flag", out.String())
	}
}

func TestSweep_Write(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	var out strings.Builder
	if err := sweep(&out, root, rewriting(), nil); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	const held = "held.go:3 left zero width space " +
		"(in a string literal, write it as the escape \\u200B; clean leaves it)"
	if !strings.Contains(out.String(), held) {
		t.Errorf("output lacks %q:\n%s", held, out.String())
	}

	testCases := []struct {
		name      string
		rel       string
		wantClean bool
	}{
		{"markdown carrier removed", "notes.md", true},
		{"go carrier removed", "code.go", true},
		{"unlisted extension untouched", "skip.txt", false},
		{"fixture untouched", "testdata/fixture.md", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body, err := os.ReadFile(filepath.Join(root, testCase.rel))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			intact := !strings.ContainsRune(string(body), '\u200B')
			if intact != testCase.wantClean {
				t.Errorf("%q clean = %v, want %v", testCase.rel, intact, testCase.wantClean)
			}
		})
	}

	t.Run("em dash survives, reported not rewritten", func(t *testing.T) {
		t.Parallel()

		notes, err := os.ReadFile(filepath.Join(root, "notes.md"))
		if err != nil {
			t.Fatalf("read notes: %v", err)
		}
		if !strings.ContainsRune(string(notes), '\u2014') {
			t.Error("the em dash was rewritten; only carriers may be removed")
		}
	})
}

func TestSweep_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := seedTree(t)
	var first, second strings.Builder
	if err := sweep(&first, root, rewriting(), nil); err != nil {
		t.Fatalf("first sweep: %v", err)
	}
	if err := sweep(&second, root, rewriting(), nil); err != nil {
		t.Fatalf("second sweep: %v", err)
	}
	if !strings.Contains(second.String(), "removed 0 carrier(s)") {
		t.Errorf("second sweep acted again: %q", second.String())
	}
}

func TestSweep_QuotesForgedPath(t *testing.T) {
	t.Parallel()

	name := "a\n[ERROR] x\nb.md"
	testCases := []struct {
		name string
		how  sweepStrategy
	}{
		{"report", inspecting()},
		{"rewrite", rewriting()},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			body := []byte("a\u200Bb\n")
			if err := os.WriteFile(filepath.Join(root, name), body, 0o600); err != nil {
				t.Skipf("filesystem refuses the name: %v", err)
			}
			var out strings.Builder
			if err := sweep(&out, root, testCase.how, nil); err != nil {
				t.Fatalf("sweep: %v", err)
			}
			for line := range strings.SplitSeq(out.String(), "\n") {
				if strings.HasPrefix(line, "[ERROR]") {
					t.Errorf("forged line in output: %q", out.String())
				}
			}
			if !strings.Contains(out.String(), `\n[ERROR] x\nb.md"`) {
				t.Errorf("path not quoted: %q", out.String())
			}
		})
	}
}

func TestSweep_WalksTheGatesGitFileSet(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	files := map[string]string{
		".gitignore":         "scratch/\n",
		"scratch/notes.md":   "a\u200Bb\n",
		"build/tracked.md":   "a\u200Bb\n",
		"build/untracked.md": "a\u200Bb\n",
	}
	for rel, body := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatalf("seed %q: %v", rel, err)
		}
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"add", ".gitignore", "build/tracked.md"},
	} {
		cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", root}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Skipf("git %v: %v: %s", args, err, out)
		}
	}

	var out strings.Builder
	if sweepErr := sweep(&out, root, inspecting(), openRepo(t, root)); sweepErr != nil {
		t.Fatalf("sweep: %v", sweepErr)
	}
	testCases := []struct {
		name string
		path string
		want bool
	}{
		{"gitignored file is skipped", "scratch/notes.md", false},
		{"tracked build file is walked", "build/tracked.md", true},
		{"untracked build file is walked", "build/untracked.md", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := strings.Contains(out.String(), testCase.path); got != testCase.want {
				t.Errorf("reported = %v, want %v:\n%s", got, testCase.want, out.String())
			}
		})
	}
}

func seedScopedRepo(tb testing.TB) string {
	tb.Helper()
	root := tb.TempDir()
	for _, rel := range []string{"sub/inner.md", "top.md", "build/tracked.md", "subway/x.md"} {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			tb.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(path, []byte("a\u200Bb\n"), 0o600); err != nil {
			tb.Fatalf("seed %q: %v", rel, err)
		}
	}
	cmd := exec.CommandContext(tb.Context(), "git", "-C", root, "init", "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		tb.Skipf("git init: %v: %s", err, out)
	}
	add := exec.CommandContext(tb.Context(), "git", "-C", root, "add", "build/tracked.md")
	if out, err := add.CombinedOutput(); err != nil {
		tb.Skipf("git add: %v: %s", err, out)
	}
	return root
}

func TestSweep_StaysInsideTheNamedDirectory(t *testing.T) {
	t.Parallel()

	all := []string{"sub/inner.md", "top.md", "build/tracked.md", "subway/x.md"}
	testCases := []struct {
		name     string
		project  string
		how      sweepStrategy
		writes   bool
		reported []string
	}{
		{"report in sub", "sub", inspecting(), false, []string{"sub/inner.md"}},
		{"write in sub", "sub", rewriting(), true, []string{"sub/inner.md"}},
		{"report in build", "build", inspecting(), false, []string{"build/tracked.md"}},
		{"report at the repo root", ".", inspecting(), false, all},
		{"write at the repo root", ".", rewriting(), true, all},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := seedScopedRepo(t)
			var out strings.Builder
			project := filepath.Join(root, testCase.project)
			if err := sweep(&out, project, testCase.how, openRepo(t, root)); err != nil {
				t.Fatalf("sweep: %v", err)
			}
			for _, rel := range all {
				want := slices.Contains(testCase.reported, rel)
				if got := strings.Contains(out.String(), rel); got != want {
					t.Errorf("%q reported = %v:\n%s", rel, got, out.String())
				}
				body, readErr := os.ReadFile(filepath.Join(root, rel))
				if readErr != nil {
					t.Fatalf("read: %v", readErr)
				}
				touched := !strings.ContainsRune(string(body), '\u200B')
				if wantTouched := testCase.writes && want; touched != wantTouched {
					t.Errorf("%q rewritten = %v, want %v", rel, touched, wantTouched)
				}
			}
		})
	}
}

func TestSweep_SymlinkOutsideNotFollowed(t *testing.T) {
	t.Parallel()

	root := seedScopedRepo(t)
	link := filepath.Join(root, "sub", "link.md")
	if err := os.Symlink(filepath.Join("..", "top.md"), link); err != nil {
		t.Skipf("symlink: %v", err)
	}
	var out strings.Builder
	if err := sweep(&out, filepath.Join(root, "sub"), rewriting(), openRepo(t, root)); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	top, err := os.ReadFile(filepath.Join(root, "top.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(top) != "a\u200Bb\n" {
		t.Errorf("top.md outside the named directory was rewritten: %q", top)
	}
	inner, err := os.ReadFile(filepath.Join(root, "sub", "inner.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.ContainsRune(string(inner), '\u200B') {
		t.Errorf("sub/inner.md inside the named directory was not cleaned: %q", inner)
	}
}

func TestSweep_DirOutsideRepo(t *testing.T) {
	t.Parallel()

	src := openRepo(t, seedScopedRepo(t))
	outside := t.TempDir()
	body := filepath.Join(outside, "a.md")
	if writeErr := os.WriteFile(body, []byte("a\u200Bb\n"), 0o600); writeErr != nil {
		t.Fatalf("seed: %v", writeErr)
	}
	if sweepErr := sweep(io.Discard, outside, rewriting(), src); sweepErr == nil {
		t.Fatal("sweep outside the repository returned nil")
	}
	got, readErr := os.ReadFile(body)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if !strings.ContainsRune(string(got), '\u200B') {
		t.Error("a file outside the repository was rewritten")
	}
}

func TestSweep_SaysWhySkippedFilesAreLeft(t *testing.T) {
	t.Parallel()

	const body = "package m\n\nvar c = \"R$\u00A010,00\"\n// tail\u00A0x\nvar d = \"half\n"
	const tail = "in files clean cannot read: fix their syntax first"
	testCases := []struct {
		name string
		how  sweepStrategy
		want string
	}{
		{"report mode", inspecting(), "0 marker(s) found; 2 more " + tail},
		{"write mode", rewriting(), "2 marker(s) left for a human; 2 of them " + tail},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := filepath.Join(root, "eof.go")
			if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			var out strings.Builder
			if err := sweep(&out, root, testCase.how, nil); err != nil {
				t.Fatalf("sweep: %v", err)
			}
			const note = "eof.go: skipped (does not lex: unterminated comment or literal): " +
				"markers on lines 3, 4"
			if !strings.Contains(out.String(), note) {
				t.Errorf("output lacks %q:\n%s", note, out.String())
			}
			if !strings.Contains(out.String(), testCase.want) {
				t.Errorf("output lacks %q:\n%s", testCase.want, out.String())
			}
			if got, _ := os.ReadFile(path); string(got) != body {
				t.Errorf("file changed: %q", got)
			}
		})
	}
}

type unlistableRepo struct{ *vcs.Repository }

func (unlistableRepo) ListFiles() ([]string, error) { return nil, errors.New("git unavailable") }

//nolint:paralleltest // Replaces the process-wide default logger.
func TestSweep_WarnsThroughTheDefaultLogger(t *testing.T) {
	var logs strings.Builder
	previous, output, flags := slog.Default(), log.Writer(), log.Flags()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous); log.SetOutput(output); log.SetFlags(flags) })
	src := unlistableRepo{openRepo(t, seedScopedRepo(t))}
	if err := sweep(io.Discard, src.Root(), inspecting(), src); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !strings.Contains(logs.String(), "walking the filesystem instead") {
		t.Errorf("logs = %q, want the fallback warning", logs.String())
	}
}
