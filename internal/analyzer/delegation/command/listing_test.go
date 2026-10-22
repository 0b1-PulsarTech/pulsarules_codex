//go:build unix

package command

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

//nolint:ireturn // test helper hands back the interface the analyzer consumes
func listedProvider(t *testing.T, paths []string) source.Provider {
	t.Helper()
	skips := source.NewSkipSet(nil)
	return source.NewListedProvider(source.NewProvider(t.TempDir(), skips), paths, skips)
}

func TestListingOf(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		paths []string
		none  bool
		want  listing
	}{
		{"nil provider", nil, true, nil},
		{
			"listed paths",
			[]string{"lib/a.dart", "web/b.dart"},
			false,
			listing{"lib/a.dart": {}, "web/b.dart": {}},
		},
		{"empty tree stays a non-nil listing", nil, false, listing{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var provider source.Provider
			if !testCase.none {
				provider = listedProvider(t, testCase.paths)
			}

			got := listingOf(provider)

			if (got == nil) != (testCase.want == nil) || !maps.Equal(got, testCase.want) {
				t.Errorf("listing = %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestListingJudge(t *testing.T) {
	t.Parallel()

	root := aliasedTree(t)
	listed := listing{
		"web/src/a.dart": {}, "link.dart": {}, "dangling.dart": {}, "gone.dart": {},
		"toignored.dart": {}, "toouter.dart": {}, "lib/real.dart": {},
		"lib/sub/alias.dart": {}, "devnull.dart": {}, "inlink": {},
	}

	testCases := []struct {
		name  string
		files listing
		rel   string
		want  verdict
	}{
		{"listed regular file", listed, "web/src/a.dart", kept},
		{"unlisted file that exists", listed, "build/out.dart", excluded},
		{"unlisted file that is missing", listed, "build/gone.dart", dropped},
		{"listed file missing from disk", listed, "gone.dart", dropped},
		{"empty listing excludes a real file", listing{}, "web/src/a.dart", excluded},
		{"empty listing drops a missing file", listing{}, "web/none.dart", dropped},
		{"no listing keeps a regular file", nil, "web/src/a.dart", kept},
		{"no listing keeps a missing file", nil, "web/none.dart", kept},
		{"unlisted under an unlisted dir link", listed, "node_modules/web/src/a.dart", excluded},
		{"file under a listed dir link", listed, "inlink/src/a.dart", kept},
		{"missing file under a listed dir link", listed, "inlink/none.dart", dropped},
		{"listed symlink to a listed file", listed, "link.dart", kept},
		{"nested link to a listed file", listed, "lib/sub/alias.dart", kept},
		{"listed link to a device", listed, "devnull.dart", dropped},
		{"listed symlink to an ignored file", listed, "toignored.dart", kept},
		{"listed symlink to an outside file", listed, "toouter.dart", kept},
		{"listed dangling symlink", listed, "dangling.dart", dropped},
		{"directory", listed, "web", dropped},
		{"project root", listed, ".", dropped},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := testCase.files.judge(root, root, testCase.rel)

			if got != testCase.want {
				t.Errorf("judge(%q) = %v, want %v", testCase.rel, got, testCase.want)
			}
		})
	}
}

func TestListingHolds(t *testing.T) {
	t.Parallel()

	const root = "/project"
	listed := listing{"web/src/a.dart": {}, "web2/b.dart": {}, "pkg/x.dart": {}}

	testCases := []struct {
		name  string
		files listing
		dir   string
		want  bool
	}{
		{"no listing holds everything", nil, root + "/anything", true},
		{"root with listed files", listed, root, true},
		{"root of an empty tree", listing{}, root, false},
		{"dir with a listed file", listed, root + "/pkg", true},
		{"dir with a deeper listed file", listed, root + "/web", true},
		{"dir with no listed file", listed, root + "/build", false},
		{"dir whose name only prefixes a listed dir", listed, root + "/we", false},
		{"namesake nested deeper", listing{"lib/build/z.dart": {}}, root + "/build", false},
		{"nested dir with no listed file", listed, root + "/web/other", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.files.holds(root, testCase.dir); got != testCase.want {
				t.Errorf("holds(%q) = %v, want %v", testCase.dir, got, testCase.want)
			}
		})
	}
}

func noFilesMessage(dir string) string {
	return fmt.Sprintf(
		"dir %q holds no file governance analyses (ignored, skipped dir, or submodule)", dir,
	)
}

type underDirCase struct {
	name  string
	dir   string
	files listing
	want  string
}

func underDirCases() []underDirCase {
	listed := listing{"web/src/a.dart": {}}

	return []underDirCase{
		{"dir with a listed file runs", "web", listed, ""},
		{"dir through a link inside runs", "inlink", listed, ""},
		{"root with listed files runs", "", listed, ""},
		{"no listing runs", "build", nil, ""},
		{"dir with no listed file", "build", listed, noFilesMessage("build")},
		{"root of an empty tree", "", listing{}, noFilesMessage("")},
		{"sibling name prefix", "web", listing{"web2/a.dart": {}}, noFilesMessage("web")},
	}
}

func TestRunEntryNeedsListedFilesUnderDir(t *testing.T) {
	t.Parallel()

	for _, testCase := range underDirCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := aliasedTree(t)
			marker := filepath.Join(t.TempDir(), "ran")
			body := fmt.Sprintf("echo ran > %q\necho '{\"version\":1}'\n", marker)
			entry := pass.DelegatedCommand{
				ID: entryID, Format: "dart-analyze-json", Dir: testCase.dir,
				Run: writeScript(t, body),
			}

			findings := NewAnalyzer(nil).runEntry(root, testCase.files, entry)

			_, statErr := os.Stat(marker)
			if ran := statErr == nil; ran != (testCase.want == "") {
				t.Errorf("command ran = %v, want %v", ran, testCase.want == "")
			}
			messages := make([]string, 0, len(findings))
			for _, hit := range findings {
				messages = append(messages, hit.AnalyzerID+": "+hit.Message)
			}
			want := []string{entryID + "/config: " + testCase.want}
			if testCase.want == "" {
				want = nil
			}
			if !slices.Equal(messages, want) {
				t.Errorf("findings = %q, want %q", messages, want)
			}
		})
	}
}

type reportItem struct {
	file     string
	line     int
	code     string
	severity string
}

func reportedAt(file string, line int, severity string) reportItem {
	return reportItem{file: file, line: line, code: "c", severity: severity}
}

func reportOn(t *testing.T, root string, files ...string) []string {
	t.Helper()
	items := make([]reportItem, len(files))
	for idx, file := range files {
		items[idx] = reportItem{file: file, code: "c"}
	}
	return reportWith(t, root, items)
}

func reportWith(t *testing.T, root string, items []reportItem) []string {
	t.Helper()
	rows := make([]string, len(items))
	for idx, item := range items {
		severity := cmp.Or(item.severity, "INFO")
		rows[idx] = fmt.Sprintf(
			`{"code":%q,"severity":%q,"location":{"file":%q,"range":{"start":{"line":%d}}}}`,
			item.code, severity, filepath.Join(root, item.file), item.line,
		)
	}
	payload := `{"version":1,"diagnostics":[` + strings.Join(rows, ",") + `]}`
	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatalf("write payload: %v", err)
	}
	return writeScript(t, fmt.Sprintf("cat %q\nexit 3\n", path))
}

func aliasedTree(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	for _, dir := range []string{"web/src", "node_modules", "build", "lib/sub"} {
		if mkErr := os.MkdirAll(filepath.Join(root, dir), 0o700); mkErr != nil {
			t.Fatalf("mkdir: %v", mkErr)
		}
	}
	for _, file := range []string{"web/src/a.dart", "build/out.dart", "lib/real.dart"} {
		if wrErr := os.WriteFile(filepath.Join(root, file), nil, 0o600); wrErr != nil {
			t.Fatalf("write: %v", wrErr)
		}
	}
	outside := filepath.Join(t.TempDir(), "x.dart")
	if wrErr := os.WriteFile(outside, nil, 0o600); wrErr != nil {
		t.Fatalf("write outside: %v", wrErr)
	}
	links := map[string]string{
		"node_modules/web": filepath.Join(root, "web"), "inlink": "web",
		"link.dart":     "web/src/a.dart",
		"dangling.dart": "nowhere.dart", "toignored.dart": "build/out.dart",
		"toouter.dart": outside, "lib/sub/alias.dart": "../real.dart",
		"devnull.dart": "/dev/null",
	}
	for name, target := range links {
		if lnErr := os.Symlink(target, filepath.Join(root, name)); lnErr != nil {
			t.Fatalf("symlink %s: %v", name, lnErr)
		}
	}
	return root
}

type listingCase struct {
	name      string
	files     []string
	wantFiles []string
	wantID    string
}

func listingCases() []listingCase {
	const aliasedFile = "node_modules/web/src/a.dart"

	return []listingCase{
		{
			"ignored and aliased files are dropped, the real one kept",
			[]string{"build/out.dart", aliasedFile, "web/src/a.dart"},
			[]string{"web/src/a.dart"},
			entryID + "/c",
		},
		{"only an ignored file is a quiet pass", []string{"build/out.dart"}, nil, ""},
		{
			"only a missing file is a failure",
			[]string{"build/gone.dart"},
			nil, entryID + "/failed",
		},
		{
			"ignored beside a missing file is reported as dropped",
			[]string{"build/out.dart", "build/gone.dart"},
			nil, entryID + "/dropped",
		},
		{"only an aliased file is a quiet pass", []string{aliasedFile}, nil, ""},
	}
}

func linkCases() []listingCase {
	return []listingCase{
		{
			"a link to a listed file reported alone is loud at the link",
			[]string{"link.dart"},
			[]string{"link.dart"},
			entryID + "/c",
		},
		{
			"a link to an ignored file is loud",
			[]string{"toignored.dart"},
			[]string{"toignored.dart"},
			entryID + "/c",
		},
		{
			"a link to an outside file is loud",
			[]string{"toouter.dart"},
			[]string{"toouter.dart"},
			entryID + "/c",
		},
		{"a link to a device is a failure", []string{"devnull.dart"}, nil, entryID + "/failed"},
		{"a dangling link is a failure", []string{"dangling.dart"}, nil, entryID + "/failed"},
		{"a dir is a failure", []string{"web"}, nil, entryID + "/failed"},
		{"the root is a failure", []string{"."}, nil, entryID + "/failed"},
		{
			"ignored beside an outside file is reported as dropped",
			[]string{"build/out.dart", "../elsewhere/x.dart"},
			nil, entryID + "/dropped",
		},
		{
			"only an outside file is still a failure",
			[]string{"../elsewhere/x.dart"},
			nil, entryID + "/failed",
		},
	}
}

func TestRunEntryListing(t *testing.T) {
	t.Parallel()

	listed := listing{
		"web/src/a.dart": {}, "link.dart": {}, "toignored.dart": {}, "toouter.dart": {},
		"dangling.dart": {}, "lib/real.dart": {}, "lib/sub/alias.dart": {}, "devnull.dart": {},
	}
	for _, testCase := range slices.Concat(listingCases(), linkCases()) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := aliasedTree(t)
			entry := pass.DelegatedCommand{
				ID: entryID, Format: "dart-analyze-json",
				Run: reportOn(t, root, testCase.files...),
			}

			findings := NewAnalyzer(nil).runEntry(root, listed, entry)

			files := make([]string, 0, len(findings))
			ids := make([]string, 0, len(findings))
			for _, hit := range findings {
				files, ids = append(files, hit.File), append(ids, hit.AnalyzerID)
			}
			wantIDs := []string{}
			if testCase.wantID != "" {
				wantIDs = []string{testCase.wantID}
			}
			if !slices.Equal(ids, wantIDs) {
				t.Fatalf("ids = %v, want %v", ids, wantIDs)
			}
			if testCase.wantFiles != nil && !slices.Equal(files, testCase.wantFiles) {
				t.Errorf("files = %v, want %v", files, testCase.wantFiles)
			}
		})
	}
}

type reportedFile struct {
	severity finding.Severity
	file     string
}

type linkReportCase struct {
	name  string
	items []reportItem
	want  []reportedFile
}

func linkReportCases() []linkReportCase {
	const (
		realFile = "web/src/a.dart"
		linkFile = "link.dart"
		nested   = "lib/sub/alias.dart"
	)
	failure, hint := finding.SeverityError, finding.SeverityInfo

	return []linkReportCase{
		{"link and target both reported", []reportItem{
			reportedAt(realFile, 3, "ERROR"), reportedAt(linkFile, 3, "ERROR"),
		}, []reportedFile{{failure, realFile}, {failure, linkFile}}},
		{"link before target keeps the report order", []reportItem{
			reportedAt(linkFile, 3, "ERROR"), reportedAt(realFile, 3, "ERROR"),
		}, []reportedFile{{failure, linkFile}, {failure, realFile}}},
		{"error link over an info target", []reportItem{
			reportedAt(realFile, 3, "INFO"), reportedAt(linkFile, 3, "ERROR"),
		}, []reportedFile{{hint, realFile}, {failure, linkFile}}},
		{"error link alone", []reportItem{
			reportedAt(linkFile, 3, "ERROR"),
		}, []reportedFile{{failure, linkFile}}},
		{"nested link beside its target", []reportItem{
			reportedAt("lib/real.dart", 1, "ERROR"), reportedAt(nested, 1, "ERROR"),
		}, []reportedFile{{failure, "lib/real.dart"}, {failure, nested}}},
	}
}

func TestRunEntryReportsEveryLink(t *testing.T) {
	t.Parallel()

	listed := listing{
		"web/src/a.dart": {}, "link.dart": {}, "lib/real.dart": {}, "lib/sub/alias.dart": {},
	}
	for _, testCase := range linkReportCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := aliasedTree(t)
			entry := pass.DelegatedCommand{
				ID: entryID, Format: "dart-analyze-json",
				Run: reportWith(t, root, testCase.items),
			}

			findings := NewAnalyzer(nil).runEntry(root, listed, entry)

			got := make([]reportedFile, 0, len(findings))
			for _, hit := range findings {
				got = append(got, reportedFile{hit.Severity, hit.File})
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("findings = %v, want %v", got, testCase.want)
			}
		})
	}
}
