package simplification

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestAnalyze(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	writeFixture := func(t *testing.T, name, content string) string {
		t.Helper()
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture: %v", err)
		}
		return name
	}

	sameLinePath := writeFixture(t, "same_line_upgrade_path.go", "package foo\n\n"+
		"// simplification: skip the rare case. Upgrade path: handle it once seen twice.\n"+
		"func f() {}\n")

	revisitWhen := writeFixture(t, "revisit_when.go", "package foo\n\n"+
		"// simplification: skip the rare case; revisit when it shows up twice.\n"+
		"func f() {}\n")

	noLabel := writeFixture(t, "no_label.go", "package foo\n\n"+
		"// simplification: skip the rare case because it never happens in practice.\n"+
		"func f() {}\n")

	laterLine := writeFixture(t, "later_line.go", "package foo\n\n"+
		"// simplification: skip the rare case, which is ample signal for now.\n"+
		"// This keeps the function small.\n"+
		"// Upgrade path: handle it once the rare case is seen twice.\n"+
		"func f() {}\n")

	noMarker := writeFixture(t, "no_marker.go", "package foo\n\n"+
		"// f does a thing.\n"+
		"func f() {}\n")

	prosemention := writeFixture(t, "prose_mention.go", "package foo\n\n"+
		"// f applies a simplification the caller already validated, nothing more.\n"+
		"func f() {}\n")

	srcs := pass.NewSourceProvider(dir)

	testCases := []struct {
		name    string
		path    string
		wantLen int
	}{
		{"upgrade path label on the marker line", sameLinePath, 0},
		{"revisit when label on the marker line", revisitWhen, 0},
		{"no label at all fires", noLabel, 1},
		{"upgrade path label on a later block line", laterLine, 0},
		{"no marker at all", noMarker, 0},
		{"word simplification in prose, not a marker", prosemention, 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			a := NewAnalyzer()
			got := a.Analyze(&pass.Context{
				ProjectDir: dir,
				Sources:    srcs,
				ChangedFiles: []pass.FileChange{
					{Path: testCase.path, Extension: ".go"},
				},
			})
			if len(got) != testCase.wantLen {
				t.Fatalf("got %d findings, want %d: %+v", len(got), testCase.wantLen, got)
			}
		})
	}
}

func TestAnalyze_NoSources(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	got := a.Analyze(
		&pass.Context{ChangedFiles: []pass.FileChange{{Path: "x.go", Extension: ".go"}}},
	)
	if got != nil {
		t.Fatalf("expected nil findings with no source provider, got %+v", got)
	}
}

func TestAnalyze_NonGoFileSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "notes.md")
	if err := os.WriteFile(path, []byte("// simplification: not go\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	a := NewAnalyzer()
	got := a.Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      pass.NewSourceProvider(dir),
		ChangedFiles: []pass.FileChange{{Path: "notes.md", Extension: ".md"}},
	})
	if got != nil {
		t.Fatalf("expected nil findings for a non-Go file, got %+v", got)
	}
}

func TestAnalyze_NonexistentFileSkipped(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := NewAnalyzer()
	got := a.Analyze(&pass.Context{
		ProjectDir:   dir,
		Sources:      pass.NewSourceProvider(dir),
		ChangedFiles: []pass.FileChange{{Path: "gone.go", Extension: ".go"}},
	})
	if got != nil {
		t.Fatalf("expected nil findings for a nonexistent file, got %+v", got)
	}
}

func TestBlockNamesUpgradePath_WrappedLabel(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"label on one line", []string{"// simplification: x", "// Upgrade path: do y"}, true},
		{
			"label wrapped by golines",
			[]string{"// simplification: x. Upgrade", "// path: do y"},
			true,
		},
		{
			"revisit-when wrapped",
			[]string{"// simplification: x, revisit", "// when y ships"},
			true,
		},
		{"no label at all", []string{"// simplification: x", "// and nothing else"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := blockNamesUpgradePath(testCase.lines); got != testCase.want {
				t.Errorf(
					"blockNamesUpgradePath(%q) = %v, want %v",
					testCase.lines,
					got,
					testCase.want,
				)
			}
		})
	}
}
