package loader

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/vcs"
)

const (
	baseStrict     = "version: 1\npreset: strict\n"
	bogusKey       = "version: 1\nbogus: 1\n"
	private        = ".git/pulsarules.yaml: "
	privateMinimal = "version: 1\npreset: minimal\n"
)

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	identity := []string{"-C", dir, "-c", "user.name=probe", "-c", "user.email=probe@example.com"}
	command := exec.CommandContext(t.Context(), "git", slices.Concat(identity, args)...)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

type layout struct {
	base, private string
	git, worktree bool
	privateKind   string
	brokenGit     bool
	deep          string
	baseLoop      bool
	unreadable    bool
	privateLen    int
}

func (spec layout) build(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if spec.privateLen > 0 {
		spec.deep = exactDeep(spec.privateLen - len(root) - len("/main/.git/"+privateFileName))
	}
	if spec.deep != "" && !spec.worktree {
		root = filepath.Join(root, spec.deep)
		if err := os.MkdirAll(root, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	mainDir := root
	if spec.worktree {
		mainDir = filepath.Join(root, "main", spec.deep)
		root = filepath.Join(root, "linked")
		if err := os.MkdirAll(mainDir, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if spec.git || spec.worktree {
		gitRun(t, mainDir, "init", "-q")
	}
	if spec.worktree {
		gitRun(t, mainDir, "commit", "-q", "--allow-empty", "-m", "init")
		gitRun(t, mainDir, "worktree", "add", "-q", root, "-b", "probe")
	}
	spec.placeBase(t, root)
	spec.placePrivate(t, filepath.Join(mainDir, ".git", privateFileName))
	return root
}

func (spec layout) placeBase(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, projectfile.Name)
	if spec.base != "" {
		write(t, path, spec.base)
	}
	if spec.baseLoop {
		if err := os.Symlink(projectfile.Name, path); err != nil {
			t.Fatalf("symlink: %v", err)
		}
	}
	if spec.unreadable {
		write(t, path, bogusKey)
		if err := os.Chmod(path, 0); err != nil {
			t.Fatalf("chmod: %v", err)
		}
	}
	if spec.brokenGit {
		write(t, filepath.Join(root, ".git"), "gitdir: "+filepath.Join(root, "nowhere")+"\n")
	}
}

func (spec layout) placePrivate(t *testing.T, path string) {
	t.Helper()
	switch {
	case spec.privateKind == "dir":
		if err := os.Mkdir(path, 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	case spec.privateKind == "fifo":
		makeFifo(t, path)
	case spec.private != "":
		write(t, path, spec.private)
	}
}

type lookupCase struct {
	name       string
	spec       layout
	explicit   string
	wantPreset string
	wantNil    bool
	wantErr    []string
}

func lookupCases() []lookupCase {
	return slices.Concat(lookupLayoutCases(), lookupFaultCases())
}

func lookupLayoutCases() []lookupCase {
	return []lookupCase{
		{name: "only base", spec: layout{base: baseStrict, git: true}, wantPreset: "strict"},
		{
			name:       "only private",
			spec:       layout{private: privateMinimal, git: true},
			wantPreset: "minimal",
		},
		{
			name:       "both, private wins",
			spec:       layout{base: baseStrict, private: privateMinimal, git: true},
			wantPreset: "minimal",
		},
		{name: "neither", spec: layout{git: true}, wantNil: true},
		{name: "no git, only base", spec: layout{base: baseStrict}, wantPreset: "strict"},
		{name: "no git, nothing", spec: layout{}, wantNil: true},
		{
			name:       "explicit ignores both",
			spec:       layout{base: bogusKey, private: bogusKey, git: true},
			explicit:   "version: 1\npreset: recommended\n",
			wantPreset: "recommended",
		},
		{
			name:       "linked worktree reads the common dir",
			spec:       layout{base: baseStrict, private: privateMinimal, worktree: true},
			wantPreset: "minimal",
		},
	}
}

func lookupFaultCases() []lookupCase {
	huge := "version: 1\n#" + strings.Repeat("x", projectfile.MaxFileBytes)
	return []lookupCase{
		{
			name:    "linked worktree names the private file by its absolute path",
			spec:    layout{private: bogusKey, worktree: true},
			wantErr: []string{"main/.git/pulsarules.yaml: line 2: bogus: unknown key"},
		},
		{
			name:    "invalid private alone",
			spec:    layout{base: baseStrict, private: bogusKey, git: true},
			wantErr: []string{private + "line 2: bogus: unknown key"},
		},
		{
			name:    "invalid base alone",
			spec:    layout{base: bogusKey, private: privateMinimal, git: true},
			wantErr: []string{".pulsarules.yaml: line 2: bogus: unknown key"},
		},
		{
			name: "both invalid, both named",
			spec: layout{base: bogusKey, private: bogusKey, git: true},
			wantErr: []string{
				".pulsarules.yaml: line 2: bogus", "; " + private + "line 2: bogus",
			},
		},
		{
			name:    "private version unsupported",
			spec:    layout{base: baseStrict, private: "version: 2\n", git: true},
			wantErr: []string{private + "version: unsupported 2"},
		},
		{
			name:    "private over the size cap",
			spec:    layout{private: huge, git: true},
			wantErr: []string{private + "config is larger than 256 KiB"},
		},
	}
}

func TestLoadLayers(t *testing.T) {
	t.Parallel()

	for _, testCase := range lookupCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			root := testCase.spec.build(t)
			var explicit string
			if testCase.explicit != "" {
				explicit = filepath.Join(t.TempDir(), "explicit.yaml")
				write(t, explicit, testCase.explicit)
			}
			result := loadWithin(t, root, explicit)
			if len(testCase.wantErr) > 0 {
				assertErrorHas(t, result.err, testCase.wantErr)
				return
			}
			if result.err != nil {
				t.Fatalf("Load() error = %v", result.err)
			}
			if (result.file == nil) != testCase.wantNil {
				t.Fatalf("Load() file = %+v, wantNil %v", result.file, testCase.wantNil)
			}
			if result.file != nil && result.file.Preset != testCase.wantPreset {
				t.Fatalf("Load() preset = %q, want %q", result.file.Preset, testCase.wantPreset)
			}
		})
	}
}

func assertErrorHas(t *testing.T, err error, parts []string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(fmt.Sprint(err), part) {
			t.Fatalf("error = %v, want containing %q", err, part)
		}
	}
}

func TestDisplayName(t *testing.T) {
	t.Parallel()

	sep := string(filepath.Separator)
	repo := sep + "repo"
	testCases := []struct {
		name       string
		root, path string
		want       string
	}{
		{"inside the root", repo, repo + "/.git/x.yaml", ".git/x.yaml"},
		{"outside the root", repo, sep + "main/.git/x.yaml", sep + "main/.git/x.yaml"},
		{"sibling sharing a prefix", repo, repo + "2/x.yaml", repo + "2/x.yaml"},
		{"relative path against an absolute root", repo, "x.yaml", "x.yaml"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := displayName(testCase.root, testCase.path); got != testCase.want {
				t.Fatalf("displayName() = %q, want %q", got, testCase.want)
			}
		})
	}
}

type forRootCase struct {
	name         string
	spec         layout
	wantFile     string
	wantMessage  string
	wantNoSubstr string
}

func forRootCases() []forRootCase {
	return slices.Concat(forRootFileCases(), forRootBuildCases(), forRootNumberingCases())
}

func forRootFileCases() []forRootCase {
	const privateName = ".git/pulsarules.yaml"
	return []forRootCase{
		{
			name:        "invalid private is one config finding",
			spec:        layout{base: baseStrict, private: bogusKey, git: true},
			wantFile:    privateName,
			wantMessage: privateName + ": line 2: bogus: unknown key",
		},
		{
			name:        "invalid base names the base",
			spec:        layout{base: bogusKey, private: baseStrict, git: true},
			wantFile:    projectfile.Name,
			wantMessage: ".pulsarules.yaml: line 2: bogus: unknown key",
		},
		{
			name:        "both invalid blames the base first",
			spec:        layout{base: bogusKey, private: bogusKey, git: true},
			wantFile:    projectfile.Name,
			wantMessage: "; " + privateName + ": line 2: bogus",
		},
	}
}

func forRootBuildCases() []forRootCase {
	const (
		privateName = ".git/pulsarules.yaml"
		narrowOff   = "analyzers: {file-size: {enabled: false}}\n"
	)
	widen := overrideOf("a", "enabled: true")
	return []forRootCase{
		{
			name: "base alone widening names the base, not the merge",
			spec: layout{
				base: "version: 1\n" + narrowOff + widen, private: baseStrict, git: true,
			},
			wantFile:     projectfile.Name,
			wantMessage:  ".pulsarules.yaml: overrides[0].analyzers.file-size.enabled",
			wantNoSubstr: "(merged)",
		},
		{
			name: "private alone widening names the private file",
			spec: layout{
				base: baseStrict, private: "version: 1\n" + narrowOff + widen, git: true,
			},
			wantFile:     privateName,
			wantMessage:  privateName + ": overrides[0].analyzers.file-size.enabled",
			wantNoSubstr: "(merged)",
		},
	}
}

func forRootNumberingCases() []forRootCase {
	const privateName = ".git/pulsarules.yaml"
	baseOverrides := overridesOf(
		"{files: [x], analyzers: {file-size: {severity: info}}}",
		"{files: [y], analyzers: {file-size: {severity: error}}}",
	)
	return []forRootCase{
		{
			name: "private override widening a base disable names the private file",
			spec: layout{
				base:    doc("analyzers: {big-comment: {enabled: false}}", baseOverrides),
				private: doc(enableDoc("a", "big-comment")), git: true,
			},
			wantFile:     privateName,
			wantMessage:  privateName + ": overrides[0].analyzers.big-comment.enabled: big-comm",
			wantNoSubstr: "(merged)",
		},
		{
			name: "private override index counts the private file only",
			spec: layout{
				base:    doc("analyzers: {big-comment: {enabled: false}}", baseOverrides),
				private: doc(overridesOf(enable("a", "file-size"), enable("b", "big-comment"))),
				git:     true,
			},
			wantFile:     privateName,
			wantMessage:  privateName + ": overrides[1].analyzers.big-comment.enabled",
			wantNoSubstr: "(merged)",
		},
	}
}

func TestForRootLayered(t *testing.T) {
	t.Parallel()

	for _, testCase := range forRootCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, findings := forRoot(testCase.spec.build(t))
			assertConfigFinding(t, findings, testCase.wantFile, testCase.wantMessage)
			if testCase.wantNoSubstr != "" &&
				strings.Contains(findings[0].Message, testCase.wantNoSubstr) {
				t.Fatalf("message = %q, must not contain %q",
					findings[0].Message, testCase.wantNoSubstr)
			}
		})
	}
}

func failing(err error) commonDirResolver {
	return func(string) (string, error) { return "", err }
}

type failClosedCase struct {
	name     string
	resolve  commonDirResolver
	withBase bool
	wantNil  bool
	wantErr  string
}

func failClosedCases() []failClosedCase {
	return []failClosedCase{
		{
			name:    "not a repository means no private layer",
			resolve: failing(fmt.Errorf("read git common dir: %w", vcs.ErrNoRepository)),
			wantNil: true,
		},
		{
			name:    "git missing means no private layer",
			resolve: failing(fmt.Errorf("read git common dir: %w", vcs.ErrGitNotFound)),
			wantNil: true,
		},
		{
			name:    "timeout is a config error",
			resolve: failing(context.DeadlineExceeded),
			wantErr: "locate private config: context deadline exceeded",
		},
		{
			name:    "dubious ownership is a config error",
			resolve: failing(errors.New("fatal: detected dubious ownership")),
			wantErr: "locate private config: fatal: detected dubious ownership",
		},
		{
			name:     "a valid base file is not blamed when git fails",
			resolve:  failing(errors.New("fatal: simulated failure")),
			withBase: true,
			wantErr:  "locate private config: fatal: simulated failure",
		},
	}
}

func TestLoadPrivateFailsClosed(t *testing.T) {
	t.Parallel()

	for _, testCase := range failClosedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			spec := layout{}
			if testCase.withBase {
				spec.base = baseStrict
			}
			file, err := loadLayers(spec.build(t), testCase.resolve, "")
			if testCase.wantErr != "" {
				assertErrorHas(t, err, []string{testCase.wantErr})
				if got := faultFile(err, nil); got != "<git-common-dir>/pulsarules.yaml" {
					t.Fatalf("faultFile() = %q, want the expected private file", got)
				}
				return
			}
			if err != nil || (file == nil) != testCase.wantNil {
				t.Fatalf("loadPrivate() = %+v, %v", file, err)
			}
		})
	}
}

func TestLoadBrokenWorktreeFailsClosed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		spec layout
	}{
		{"with a base file", layout{base: baseStrict, brokenGit: true}},
		{"without a base file", layout{brokenGit: true}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			result := loadWithin(t, testCase.spec.build(t), "")
			assertErrorHas(t, result.err, []string{"locate private config: read git common dir"})
			if errors.Is(result.err, vcs.ErrNoRepository) {
				t.Fatalf("Load() error = %v, must not read as no repository", result.err)
			}
		})
	}
}
