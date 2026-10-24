package projectfile

import (
	"fmt"
	"strings"
	"testing"
)

type validateCase struct {
	name    string
	content string
	want    string
}

func testRules() Rules {
	return Rules{
		AnalyzerIDs: []string{"file-size", "big-comment", "golangci-lint"},
		ReservedIDs: []string{"file-size", "big-comment", "golangci-lint", "output"},
		Params: map[string][]string{
			"file-size":   {"max_lines", "error_lines"},
			"big-comment": {"allowed_numbers"},
		},
		DelegatedFormats: []string{"dart-analyze-json"},
	}
}

func runValidateCases(t *testing.T, testCases []validateCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil {
				t.Fatalf("decodeFile() error = %v", err)
			}
			err = file.Validate(testRules())
			assertPlainMessage(t, err)
			if testCase.want == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, testCase.want)
			}
			if !strings.HasPrefix(err.Error(), Name+": ") {
				t.Fatalf("Validate() error %q lacks the file path", err)
			}
		})
	}
}

func TestValidateNamesThePathOfTheFile(t *testing.T) {
	t.Parallel()

	file := &File{Path: "custom/place.yaml", Version: 2}
	err := file.Validate(testRules())
	if err == nil || err.Error() != "custom/place.yaml: version: unsupported 2 (want 1)" {
		t.Fatalf("Validate() error = %v, want the file's own path", err)
	}
}

func TestValidateHeader(t *testing.T) {
	t.Parallel()

	runValidateCases(t, []validateCase{
		{"valid", "version: 1\nanalyzers: {file-size: {params: {max_lines: 9}}}\n", ""},
		{"missing version", "preset: strict\n", "version: required"},
		{"bad version", "version: 2\n", "version: unsupported 2"},
		{
			"bad preset", "version: 1\npreset: wild\n",
			`preset: unknown preset "wild" (want recommended|strict|minimal)`,
		},
		{"bad skip dir", "version: 1\nskip_dirs: [a/b]\n", "skip_dirs: \"a/b\""},
		{"bad tests glob", "version: 1\ntests: [\"a//b\"]\n", "tests[0]:"},
	})
}

func TestValidateAnalyzers(t *testing.T) {
	t.Parallel()

	runValidateCases(t, []validateCase{
		{
			"unknown analyzer",
			"version: 1\nanalyzers: {nope: {}}\n",
			"analyzers.nope: unknown analyzer",
		},
		{
			"bad severity",
			"version: 1\nanalyzers: {file-size: {severity: loud}}\n",
			"analyzers.file-size.severity: invalid severity",
		},
		{
			"unknown param",
			"version: 1\nanalyzers: {file-size: {params: {max_line: 1}}}\n",
			"analyzers.file-size.params.max_line: unknown param",
		},
		{
			"param on analyzer without params",
			"version: 1\nanalyzers: {big-comment: {params: {max_lines: 1}}}\n",
			"analyzers.big-comment.params.max_lines: unknown param",
		},
	})
}

func TestValidateDelegated(t *testing.T) {
	t.Parallel()

	const (
		head = "version: 1\ndelegated:\n"
		good = "  - {id: a, run: [x], format: dart-analyze-json}\n"
	)
	runValidateCases(t, []validateCase{
		{"valid", head + good, ""},
		{
			"no id",
			head + "  - {run: [x], format: dart-analyze-json}\n",
			"delegated[0].id: required",
		},
		{"duplicate id", head + good + good, "delegated[1].id: duplicate"},
		{
			"no run",
			head + "  - {id: a, format: dart-analyze-json}\n",
			"delegated[0].run: required",
		},
		{
			"bad format",
			head + "  - {id: a, run: [x], format: nope}\n",
			"delegated[0].format: unknown format",
		},
		{
			"registry id",
			head + "  - {id: file-size, run: [x], format: dart-analyze-json}\n",
			"delegated[0].id: \"file-size\" is a built-in analyzer id",
		},
		{
			"internal registry id",
			head + "  - {id: output, run: [x], format: dart-analyze-json}\n",
			"delegated[0].id: \"output\" is a built-in analyzer id",
		},
		{
			"id with slash",
			head + "  - {id: a/b, run: [x], format: dart-analyze-json}\n",
			"delegated[0].id: \"a/b\" is not a valid id (want [a-z0-9][a-z0-9_-]*, at most 64",
		},
	})
}

func TestValidateDelegatedIDShape(t *testing.T) {
	t.Parallel()

	entry := func(id string) string {
		return "version: 1\ndelegated:\n  - {id: \"" + id +
			"\", run: [x], format: dart-analyze-json}\n"
	}
	bad := func(quoted string) string {
		return "delegated[0].id: " + quoted + " is not a valid id"
	}
	runValidateCases(t, []validateCase{
		{"hyphenated", entry("dart-analyze"), ""},
		{"underscored", entry("svelte_check"), ""},
		{"64 bytes", entry(strings.Repeat("a", 64)), ""},
		{"65 bytes", entry(strings.Repeat("a", 65)), bad(`"` + strings.Repeat("a", 65) + `"`)},
		{"uppercase", entry("Dart"), bad(`"Dart"`)},
		{"space", entry("a b"), bad(`"a b"`)},
		{"newline", entry(`a\nb`), bad(`"a\nb"`)},
		{"control byte", entry(`a\x01b`), bad(`"a\x01b"`)},
		{"leading hyphen", entry("-a"), bad(`"-a"`)},
		{"leading underscore", entry("_a"), bad(`"_a"`)},
		{"leading digit", entry("9x"), ""},
		{"inner uppercase", entry("dArt"), bad(`"dArt"`)},
		{"non ascii", entry("caf\u00e9"), bad("\"caf\u00e9\"")},
	})
}

func TestValidateDelegatedDir(t *testing.T) {
	t.Parallel()

	dir := func(path string) string {
		return "version: 1\ndelegated:\n  - {id: a, dir: " + path +
			", run: [x], format: dart-analyze-json}\n"
	}
	const fault = "must be a relative path inside the project"
	runValidateCases(t, []validateCase{
		{"dir inside", dir("web/app"), ""},
		{"dir escapes", dir("../out"), "delegated[0].dir: \"../out\" " + fault},
		{"dir absolute", dir("/etc"), "delegated[0].dir: \"/etc\" " + fault},
	})
}

func TestValidateDelegatedIDsAsKeys(t *testing.T) {
	t.Parallel()

	const (
		delegated = "delegated: [{id: dart-analyze, run: [x], format: dart-analyze-json}]\n"
		override  = "version: 1\noverrides: [{files: [a], analyzers: {"
	)
	const analyzers = "version: 1\nanalyzers: {dart-analyze: "
	runValidateCases(t, []validateCase{
		{"analyzers key", analyzers + "{enabled: false}}\n" + delegated, ""},
		{"override key", override + "dart-analyze: {enabled: false}}}]\n" + delegated, ""},
		{"override sub id", override + "dart-analyze/x: {enabled: false}}}]\n" + delegated, ""},
		{
			"param is unknown", analyzers + "{params: {x: 1}}}\n" + delegated,
			"analyzers.dart-analyze.params.x: unknown param",
		},
		{
			"undeclared id stays unknown", analyzers + "{enabled: false}}\n",
			"analyzers.dart-analyze: unknown analyzer",
		},
	})
}

type sizeShape struct {
	name    string
	content string
	total   int
	fill    string
}

func aliasedItems(head, anchored, copies string, count int) string {
	return head + anchored + strings.Repeat(copies, count-1)
}

func sizeShapes() []sizeShape {
	longX, longS := strings.Repeat("x", 1000), strings.Repeat("S", 1000)
	longK := strings.Repeat("K", 1000)
	return []sizeShape{
		{
			"skip_dirs fan-out",
			aliasedItems("version: 1\nskip_dirs:\n", `  - &s "a/`+longX+"\"\n", "  - *s\n", 31),
			31, "x",
		},
		{
			"severity fan-out",
			aliasedItems(
				"version: 1\noverrides:\n",
				`  - &o {files: [a], analyzers: {file-size: {severity: "`+longS+"\"}}}\n",
				"  - *o\n", 30,
			),
			30, "S",
		},
		{
			"glob fan-out",
			aliasedItems(
				"version: 1\ntests:\n", `  - &g "a//`+longX[:175]+"\"\n", "  - *g\n", 22,
			),
			22, "x",
		},
		{
			"long key and long value together",
			aliasedItems(
				"version: 1\noverrides:\n",
				`  - &o {files: [a], analyzers: {? "`+longK+`" : {severity: "`+longS+"\"}}}\n",
				"  - *o\n", 30,
			),
			60, "S",
		},
	}
}

func TestParseBoundsMessageSize(t *testing.T) {
	t.Parallel()

	for _, testCase := range sizeShapes() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(Name, []byte(testCase.content), testRules())
			if err == nil {
				t.Fatal("Parse() error = nil, want problems")
			}
			message := err.Error()
			if len(message) >= 4096 {
				t.Errorf("message is %d bytes, want under 4096", len(message))
			}
			if strings.Contains(message, strings.Repeat(testCase.fill, 100)) {
				t.Errorf("message carries a value longer than the clip: %.200s", message)
			}
			parts := strings.Split(message, "; ")
			want := fmt.Sprintf("... and %d more problems", testCase.total-(len(parts)-1))
			if parts[len(parts)-1] != want {
				t.Errorf("last part = %q, want %q", parts[len(parts)-1], want)
			}
		})
	}
}

type quotedCase struct {
	name    string
	content string
	rules   Rules
}

func quotedCases() []quotedCase {
	long := strings.Repeat("x", 200)
	reserved := testRules()
	reserved.ReservedIDs = append(reserved.ReservedIDs, long)
	delegated := func(id, format string) string {
		return "  - {id: " + id + ", run: [a], format: " + format + "}\n"
	}
	const head = "version: 1\ndelegated:\n"
	const known = "dart-analyze-json"
	rules := testRules()
	return []quotedCase{
		{"skip_dirs", "version: 1\nskip_dirs: [a/" + long + "]\n", testRules()},
		{"preset", "version: 1\npreset: " + long + "\n", testRules()},
		{"glob", "version: 1\ntests: [\"[" + long + "\"]\n", testRules()},
		{"severity", "version: 1\nanalyzers: {file-size: {severity: " + long + "}}\n", rules},
		{"format", head + delegated("a", long), testRules()},
		{"id with a slash", head + delegated("a/"+long, known), testRules()},
		{"built-in id", head + delegated(long, known), reserved},
		{"duplicate id", head + delegated(long, known) + delegated(long, known), testRules()},
	}
}

func TestValidateClipsQuotedValues(t *testing.T) {
	t.Parallel()

	for _, testCase := range quotedCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(Name, []byte(testCase.content), testCase.rules)
			if err == nil {
				t.Fatal("Parse() error = nil, want a problem")
			}
			message := err.Error()
			if strings.Contains(message, strings.Repeat("x", 81)) {
				t.Errorf("message holds more than the 80 character clip: %q", message)
			}
			if !strings.Contains(message, strings.Repeat("x", 78)) {
				t.Errorf("message lost the start of the value: %q", message)
			}
		})
	}
}

func overlongProblems(prefix string, count int) string {
	problems := make([]string, count)
	for idx := range problems {
		problems[idx] = fmt.Sprintf("%stests[%d]: glob longer than 256 bytes", prefix, idx)
	}
	return strings.Join(problems, "; ")
}

func TestValidateAnyAnalyzer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name, content string
		anyAnalyzer   bool
		want          string
	}{
		{"unknown id refused", "version: 1\nanalyzers: {nope: {}}\n", false, "unknown analyzer"},
		{"unknown id accepted", "version: 1\nanalyzers: {nope: {}}\n", true, ""},
		{
			"unknown id in an override accepted",
			"version: 1\noverrides: [{files: [a], analyzers: {nope/sub: {enabled: false}}}]\n",
			true, "",
		},
		{
			"severity still checked",
			"version: 1\nanalyzers: {nope: {severity: loud}}\n", true, "invalid severity",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil {
				t.Fatalf("decodeFile() error = %v", err)
			}
			rules := testRules()
			rules.AnyAnalyzer = testCase.anyAnalyzer
			err = file.Validate(rules)
			if testCase.want == "" {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, testCase.want)
			}
		})
	}
}
