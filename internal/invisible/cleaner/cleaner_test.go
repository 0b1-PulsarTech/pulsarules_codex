package cleaner

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func seed(tb testing.TB, root, rel, body string) string {
	tb.Helper()
	path := filepath.Join(root, rel)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil { //nolint:gosec // fixture.
		tb.Fatalf("seed %q: %v", rel, err)
	}
	return path
}

func TestCleanFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		body        string
		want        string
		wantChanged bool
		wantRemains int
	}{
		{"zero width space goes", "a\u200Bb\n", "ab\n", true, 0},
		{"nbsp folds to a space", "a\u00A0b\n", "a b\n", true, 0},
		{"bidi override goes", "x\u202Ey\n", "xy\n", true, 0},
		{"nothing to do", "clean\n", "clean\n", false, 0},
		{"em dash stays and is reported", "a\u2014b\n", "a\u2014b\n", false, 1},
		{
			"emoji glue stays",
			"\U0001F468\u200D\U0001F469\n",
			"\U0001F468\u200D\U0001F469\n",
			false,
			1,
		},
		{"carrier removed, dash reported", "a\u200Bb\u2014c\n", "ab\u2014c\n", true, 1},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "f.md", testCase.body)

			report, err := New(root).CleanFile(path)
			if err != nil {
				t.Fatalf("CleanFile: %v", err)
			}
			if report.Changed != testCase.wantChanged {
				t.Errorf("Changed = %v, want %v", report.Changed, testCase.wantChanged)
			}
			if len(report.Remaining) != testCase.wantRemains {
				t.Errorf("Remaining = %+v, want %d", report.Remaining, testCase.wantRemains)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != testCase.want {
				t.Errorf("file = %q, want %q", body, testCase.want)
			}
		})
	}
}

type goCase struct {
	name        string
	body        string
	want        string
	wantChanged bool
	wantRemains int
}

func runGoCases(t *testing.T, testCases []goCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "f.go", testCase.body)

			report, err := New(root).CleanFile(path)
			if err != nil {
				t.Fatalf("CleanFile: %v", err)
			}
			if report.Changed != testCase.wantChanged {
				t.Errorf("Changed = %v, want %v", report.Changed, testCase.wantChanged)
			}
			if len(report.Remaining) != testCase.wantRemains {
				t.Errorf("Remaining = %+v, want %d", report.Remaining, testCase.wantRemains)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != testCase.want {
				t.Errorf("file = %q, want %q", body, testCase.want)
			}
		})
	}
}

func keep(name, body string) goCase {
	return goCase{name: name, body: body, want: body, wantRemains: 1}
}

func TestCleanFile_KeepsGoStringLiterals(t *testing.T) {
	t.Parallel()

	testCases := []goCase{
		keep("interpreted string", "var a = \"R$\u00A010\"\n"),
		keep("raw string", "var a = `R$\u00A010`\n"),
		keep("rune", "var a = '\u00A0'\n"),
		keep("zero width space in a string", "var a = \"x\u200By\"\n"),
		keep("raw string with quote and comment openers", "var a = `say \"x\u00A0\" // y /* z`\n"),
		keep("raw string holding a rune quote", "var a = `say 'x\u00A0' // y`\n"),
		keep("raw string holding an escaped quote", "var a = `say \\\"x\u00A0\\\"`\n"),
		keep("multiline raw string", "var a = `line\nR$\u00A010\n`\n"),
	}
	runGoCases(t, testCases)
}

func TestCleanFile_CleansGoCommentsAroundLiterals(t *testing.T) {
	t.Parallel()

	testCases := []goCase{
		{"comment is still cleaned", "// a\u00A0b\n", "// a b\n", true, 0},
		{
			"block comment after a quote comment",
			"/* ' */ // a\u00A0b\n",
			"/* ' */ // a b\n",
			true,
			0,
		},
		{
			"literal kept and comment cleaned",
			"var a = \"x\u00A0y\" // p\u00A0q\n",
			"var a = \"x\u00A0y\" // p q\n",
			true,
			1,
		},
		{
			"comment after a raw string is cleaned",
			"var a = `x\u00A0y` // p\u00A0q\n",
			"var a = `x\u00A0y` // p q\n",
			true,
			1,
		},
		{
			"comment marks a quote that is no literal",
			"// \"\nvar a = 1 // \u00A0\n",
			"// \"\nvar a = 1 //  \n",
			true,
			0,
		},
		{
			"cleaning that opens a comment is no false survival",
			"/\u200B* \"a\u00A0b\"\n",
			"/* \"a\u00A0b\"\n",
			true,
			1,
		},
	}
	runGoCases(t, testCases)
}

func TestVerify(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		onDisk  string
		want    string
		missing bool
		wantErr error
	}{
		{name: "equal", onDisk: "ab\n", want: "ab\n"},
		{name: "short write", onDisk: "a", want: "ab\n", wantErr: errWriteMismatch},
		{name: "altered write", onDisk: "a\u200Bb\n", want: "ab\n", wantErr: errWriteMismatch},
		{name: "empty on disk", onDisk: "", want: "ab\n", wantErr: errWriteMismatch},
		{name: "extra bytes", onDisk: "ab\nc", want: "ab\n", wantErr: errWriteMismatch},
		{name: "missing file", want: "ab\n", missing: true, wantErr: os.ErrNotExist},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "f.go")
			if !testCase.missing {
				seed(t, filepath.Dir(path), "f.go", testCase.onDisk)
			}
			err := verify(path, testCase.want)
			if !errors.Is(err, testCase.wantErr) {
				t.Errorf("verify() = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestCleanFile_PreservesMode(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := seed(t, root, "f.go", "package a // a\u200Bb\n")

	if _, err := New(root).CleanFile(path); err != nil {
		t.Fatalf("CleanFile: %v", err)
	}
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if stat.Mode().Perm() != 0o644 {
		t.Errorf("mode = %v, want 0644", stat.Mode().Perm())
	}
}

func TestCleanFile_IsIdempotent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	path := seed(t, root, "f.md", "a\u200Bb\u00A0c\n")
	scrub := New(root)

	first, err := scrub.CleanFile(path)
	if err != nil || !first.Changed {
		t.Fatalf("first pass: changed=%v err=%v", first.Changed, err)
	}
	second, err := scrub.CleanFile(path)
	if err != nil {
		t.Fatalf("second pass: %v", err)
	}
	if second.Changed {
		t.Error("second pass rewrote an already-clean file")
	}
}

func TestInspect_NeverWrites(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	const body = "a\u200Bb\u2014c\n"
	path := seed(t, root, "f.md", body)

	report, err := New(root).Inspect(path)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(report.Remaining) != 2 {
		t.Errorf("Remaining = %+v, want 2", report.Remaining)
	}
	if report.Changed {
		t.Error("Inspect reported a change")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatalf("read: %v", readErr)
	}
	if string(got) != body {
		t.Errorf("Inspect rewrote the file to %q", got)
	}
}

type unlexableCase struct {
	name string
	body string
}

func unlexableCases() []unlexableCase {
	return []unlexableCase{
		{"unterminated block comment", "package m\n\n/* todo\n\nconst a = \"R$\u00A010\"\n"},
		{
			"odd backtick",
			"package m\n\nvar u = `half\n\nconst a = \"R$\u00A010\"\n\nvar r = `R$\u00A010`\n",
		},
		{"cut-off raw string", "package m\n\nvar a = `R$\u00A010\n"},
		{
			"double quote cut at the newline",
			"package m\n\nvar a = \"x + `L1\nL2`\n\nvar c = \"` R$\u00A010\"\n",
		},
		{
			"rune cut at the newline",
			"package m\n\nvar r = '` + 1\nL2`\n\nvar c = \"` R$\u00A010\"\n",
		},
		{
			"double quote cut at CRLF",
			"package m\r\n\r\nvar a = \"x + `L1\r\nL2`\r\n\r\nvar c = \"` R$\u00A010\"\r\n",
		},
		{
			"backslash at line end is no continuation in Go",
			"package m\n\nvar p = \"C:\\\nvar c = \"R$\u00A010\"\n",
		},
	}
}

func TestCleanFile_SkipsWhatDoesNotLex(t *testing.T) {
	t.Parallel()

	for _, testCase := range unlexableCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "m.go", testCase.body)

			report, err := New(root).CleanFile(path)
			if err != nil {
				t.Fatalf("CleanFile: %v", err)
			}
			if !report.Skipped() || report.Changed || len(report.Acted) != 0 {
				t.Errorf("report = %+v, want a skip with nothing acted", report)
			}
			const reason = "does not lex: unterminated comment or literal"
			if report.SkipReason != reason {
				t.Errorf("SkipReason = %q, want %q", report.SkipReason, reason)
			}
			body, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatalf("read: %v", readErr)
			}
			if string(body) != testCase.body {
				t.Errorf("file changed: %q, want %q", body, testCase.body)
			}
		})
	}
}

type skipCase struct {
	name      string
	body      string
	note      string
	remaining int
}

func skipCases() []skipCase {
	const (
		reason  = "skipped (does not lex: unterminated comment or literal)"
		unlexed = "var d = \"half\n"
	)
	return []skipCase{
		{
			"markers are counted and their lines named",
			"package m\n\nvar c = \"R$\u00A010\"\n// tail\u00A0x\n" + unlexed,
			reason + ": markers on lines 3, 4", 2,
		},
		{
			"two markers on one line name it once",
			"// a\u00A0b\u00A0c\n" + unlexed,
			reason + ": markers on line 1",
			2,
		},
		{"no marker names no line", "package m\n" + unlexed, reason, 0},
		{
			"the lines named are bounded",
			"package m\n" + strings.Repeat("// x\u00A0y\n", 10) + unlexed,
			reason + ": markers on lines 2, 3, 4, 5, 6, 7, 8, 9, ...", 10,
		},
	}
}

func TestInspect_SkipsWhatDoesNotLex(t *testing.T) {
	t.Parallel()

	for _, testCase := range skipCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, "m.go", testCase.body)

			report, err := New(root).Inspect(path)
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if report.SkipNote() != testCase.note {
				t.Errorf("SkipNote = %q, want %q", report.SkipNote(), testCase.note)
			}
			if len(report.Remaining) != testCase.remaining {
				t.Errorf("Remaining = %+v, want %d markers", report.Remaining, testCase.remaining)
			}
		})
	}
}

type heldCase struct {
	name      string
	file      string
	body      string
	write     bool
	remaining int
	held      []int
	removable int
}

func TestReport_SeparatesWhatCleanWillNotEdit(t *testing.T) {
	t.Parallel()

	const (
		fence          = "```\nx\u200By\n```\n"
		twoFences      = fence + "a\u200Bb\n" + fence
		proseThenFence = "a\u200Bb\n" + fence
	)
	testCases := []heldCase{
		{"prose is removable", "a.md", "a\u200Bb\n", false, 1, nil, 1},
		{"a fence is held", "a.md", fence, false, 1, []int{2}, 0},
		{"prose and a fence", "a.md", proseThenFence, false, 2, []int{3}, 1},
		{"a removable between two held", "a.md", twoFences, false, 3, []int{2, 6}, 1},
		{"a typographic mark is neither", "a.md", "```\nwait\u2026\n```\n", false, 1, nil, 0},
		{"a go string is held", "a.go", "var a = \"x\u200By\"\n", false, 1, []int{1}, 0},
		{"a go comment is removable", "a.go", "var a = 1 // x\u200By\n", false, 1, nil, 1},
		{"a write holds what it left", "a.md", proseThenFence, true, 1, []int{3}, 0},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			path := seed(t, root, testCase.file, testCase.body)
			run := New(root).Inspect
			if testCase.write {
				run = New(root).CleanFile
			}
			report, err := run(path)
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			held := len(testCase.held)
			if len(report.Remaining) != testCase.remaining || len(report.Held) != held {
				t.Errorf("Remaining %d, Held %d, want %d and %d",
					len(report.Remaining), len(report.Held), testCase.remaining, held)
			}
			if report.Removable() != testCase.removable {
				t.Errorf("Removable = %d, want %d", report.Removable(), testCase.removable)
			}
			for _, found := range report.Remaining {
				want := slices.Contains(testCase.held, found.Line)
				if got := report.IsHeld(found); got != want {
					t.Errorf("IsHeld(line %d) = %v, want %v", found.Line, got, want)
				}
			}
		})
	}
}
