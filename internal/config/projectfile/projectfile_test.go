package projectfile

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"
)

const fullProjectYAML = `version: 1
preset: strict
include_generated: true
skip_dirs: [fixtures]
tests: ["web/test/**"]
analyzers:
  big-comment: {params: {max_lines: 1}}
  doc-comment: {enabled: false, severity: error}
overrides:
  - files: ["web/test/**"]
    analyzers: {single-letter-name: {enabled: false}, file-size: {severity: warning}}
delegated:
  - {id: dart-analyze, dir: ., run: [dart, analyze], format: dart-analyze-json, rule: dart-style}
`

func decodeFile(content []byte) (*File, error) {
	doc, err := strictyaml.Decode[File](Name, content)
	if err != nil {
		return nil, err
	}
	if err = doc.Problems.Err(); err != nil {
		return nil, err
	}
	doc.Value.Path = Name
	return doc.Value, nil
}

func TestDecodeFile(t *testing.T) {
	t.Parallel()

	disabled := false
	included := true
	want := &File{
		Path:             Name,
		Version:          1,
		Preset:           "strict",
		IncludeGenerated: &included,
		SkipDirs:         []string{"fixtures"},
		Tests:            []string{"web/test/**"},
		Analyzers: map[string]AnalyzerEntry{
			"big-comment": {Params: map[string]any{"max_lines": 1}},
			"doc-comment": {Enabled: &disabled, Severity: "error"},
		},
		Overrides: []OverrideEntry{{
			Files: []string{"web/test/**"},
			Analyzers: map[string]AnalyzerEntry{
				"single-letter-name": {Enabled: &disabled},
				"file-size":          {Severity: "warning"},
			},
		}},
		Delegated: []DelegatedEntry{{
			ID: "dart-analyze", Dir: ".", Run: []string{"dart", "analyze"},
			Format: "dart-analyze-json", Rule: "dart-style",
		}},
	}

	got, err := decodeFile([]byte(fullProjectYAML))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeFile() = %+v, want %+v", got, want)
	}
}

type parseCase struct {
	name    string
	content string
	want    []string
}

func parseCases() []parseCase {
	const head = ".pulsarules.yaml: "
	return []parseCase{
		{"valid", "version: 1\n", nil},
		{
			"unknown key and a kind error",
			"version: 1\nbogus: 1\nanalyzers:\n  file-size: 5\n",
			[]string{
				head + "line 2: bogus: unknown key",
				head + "line 4: analyzers.file-size: want a mapping",
			},
		},
		{
			"unknown analyzer and unknown key",
			"version: 1\nanalyzers: {x: {bogus: 1}}\n",
			[]string{
				head + "line 2: analyzers.x.bogus: unknown key",
				head + "analyzers.x: unknown analyzer",
			},
		},
		{
			"a mistyped list item and a bad preset",
			"version: 1\ntests: [[a]]\npreset: nope\n",
			[]string{
				head + "line 2: tests[0]: want a string",
				head + "preset: unknown preset \"nope\" (want recommended|strict|minimal)",
			},
		},
		{
			"structural and semantic problems together",
			"version: 1\ntests: [\"a//b\"]\nbogus: 1\n",
			[]string{
				head + "line 3: bogus: unknown key",
				head + "tests[0]: compile glob \"a//b\": empty path segment",
			},
		},
	}
}

func aliasKeyCases() []parseCase {
	return []parseCase{
		{
			"an aliased key names a known field",
			"version: 1\nskip_dirs: [&k preset]\n*k : recommended\n",
			nil,
		},
		{
			"an aliased key repeats an earlier key",
			"&k version: 1\n*k : 2\n",
			[]string{".pulsarules.yaml: line 2: version: duplicate key, first at line 1"},
		},
	}
}

func mergePrecedenceCases() []parseCase {
	const head = ".pulsarules.yaml: line "
	return []parseCase{
		{
			"an explicit key hides the merged one",
			"version: 1\nanalyzers:\n  big-comment: &b {enabled: notabool}\n" +
				"  file-size: {<<: *b, enabled: true}\n",
			[]string{head + "3: analyzers.big-comment.enabled: want a boolean"},
		},
		{
			"an earlier merge source hides a later one",
			"version: 1\nanalyzers:\n  big-comment: &one {enabled: true}\n" +
				"  file-size: &two {enabled: notabool}\n  golangci-lint: {<<: [*one, *two]}\n",
			[]string{head + "4: analyzers.file-size.enabled: want a boolean"},
		},
	}
}

func dedupeCases() []parseCase {
	const head = ".pulsarules.yaml: "
	return []parseCase{
		{
			"a mistyped version is not also missing",
			"version: abc\n",
			[]string{head + "line 1: version: want an integer"},
		},
		{
			"a mistyped run is not also required",
			"version: 1\ndelegated:\n  - {id: x, run: a, format: dart-analyze-json}\n",
			[]string{head + "line 3: delegated[0].run: want a list of strings"},
		},
		{
			"a mistyped glob is not also a missing list",
			"version: 1\noverrides:\n  - files: [[a]]\n",
			[]string{head + "line 3: overrides[0].files[0]: want a string"},
		},
		{
			"a mistyped run word is not also a missing list",
			"version: 1\ndelegated:\n  - {id: x, run: [[a]], format: dart-analyze-json}\n",
			[]string{head + "line 3: delegated[0].run[0]: want a string"},
		},
		{
			"a null glob is not also a missing list",
			"version: 1\noverrides:\n  - files: [~]\n",
			[]string{head + "line 3: overrides[0].files[0]: want a string"},
		},
		{
			"a scalar root is not also a missing version",
			"5\n",
			[]string{head + "line 1: want a mapping"},
		},
		{
			"a list root is not also a missing version",
			"[a]\n",
			[]string{head + "line 1: want a mapping"},
		},
		{
			"a complex analyzer key is not also unknown",
			"version: 1\nanalyzers:\n  ? [a]\n  : {}\n",
			[]string{head + "line 3: analyzers.<complex key>: want a string key"},
		},
		{
			"an empty analyzer key",
			"version: 1\nanalyzers: {\"\": {}}\n",
			[]string{head + "analyzers.<empty key>: unknown analyzer"},
		},
	}
}

func TestParse(t *testing.T) {
	t.Parallel()

	for _, testCase := range slices.Concat(
		parseCases(), dedupeCases(), aliasKeyCases(), mergePrecedenceCases(), pathKeyCases(),
	) {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := Parse(Name, []byte(testCase.content), testRules())
			assertPlainMessage(t, err)
			if len(testCase.want) == 0 {
				if err != nil || file == nil || file.Path != Name {
					t.Fatalf("Parse() = %+v, %v, want the file", file, err)
				}
				return
			}
			want := strings.Join(testCase.want, "; ")
			if err == nil || err.Error() != want {
				t.Fatalf("Parse() error = %v, want %q", err, want)
			}
		})
	}
}

func problemAmplifier(width int) string {
	var content strings.Builder
	content.WriteString("version: 1\nskip_dirs: &o [" + strings.Repeat("{}, ", width) + "]\n")
	content.WriteString("overrides: [" + strings.Repeat("{files: *o}, ", width) + "]\n")
	return content.String()
}

func repeatedBadGlobs(count int) string {
	return "version: 1\ntests: [" + strings.Repeat("a//b, ", count) + "]\n"
}

func hugeAliasKey(width, overrides int) string {
	var content strings.Builder
	content.WriteString("version: 1\nanalyzers:\n  ? &k " + strings.Repeat("K", width))
	content.WriteString("\n  : {}\noverrides:\n")
	content.WriteString(strings.Repeat("  - {files: [a], analyzers: {? *k : {}}}\n", overrides))
	return content.String()
}

func TestParseCapsReportedProblems(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		content   string
		wantTail  string
		wantParts int
	}{
		{"alias fan-out", problemAmplifier(300), "... and 90280 more problems", 21},
		{"exactly the cap", repeatedBadGlobs(20), "", 20},
		{"one over the cap", repeatedBadGlobs(21), "... and 1 more problem", 21},
		{"huge alias key", hugeAliasKey(1000, 30), "... and 11 more problems", 21},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse(Name, []byte(testCase.content), testRules())
			if err == nil {
				t.Fatal("Parse() error = nil, want problems")
			}
			message := err.Error()
			if len(message) > 4096 {
				t.Errorf("message is %d bytes, want under 4096", len(message))
			}
			parts := strings.Split(message, "; ")
			if len(parts) != testCase.wantParts {
				t.Errorf("message has %d parts, want %d", len(parts), testCase.wantParts)
			}
			if testCase.wantTail != "" && parts[len(parts)-1] != testCase.wantTail {
				t.Errorf("last part = %q, want %q", parts[len(parts)-1], testCase.wantTail)
			}
		})
	}
}

func pathKeyCases() []parseCase {
	const head = ".pulsarules.yaml: analyzers.file-size.params."
	params := func(body string) string {
		return "version: 1\nanalyzers:\n  file-size:\n    params: {" + body + "}\n"
	}
	return []parseCase{
		{
			"a params key made of brackets",
			params(`"[[[": 1`),
			[]string{head + "[[[: unknown param"},
		},
		{
			"a dotted params key",
			params(`"a.b": 1`),
			[]string{head + "a.b: unknown param"},
		},
		{
			"a dotted key beside a skipped parent",
			"version: 1\nanalyzers:\n  file-size: [x]\n  a.b: {}\n  file-size.params: {}\n",
			[]string{
				".pulsarules.yaml: line 3: analyzers.file-size: want a mapping",
				".pulsarules.yaml: analyzers.a.b: unknown analyzer",
				".pulsarules.yaml: analyzers.file-size.params: unknown analyzer",
			},
		},
		{
			"both odd keys beside each other",
			params(`"[": 1, "a.b": 2, "a[0]": 3`),
			[]string{
				head + "[: unknown param",
				head + "a.b: unknown param",
				head + "a[0]: unknown param",
			},
		},
	}
}

func assertPlainMessage(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	pkg, _, _ := strings.Cut(reflect.TypeFor[File]().String(), ".")
	for _, goText := range []string{pkg + ".", "interface", "map["} {
		if strings.Contains(err.Error(), goText) {
			t.Errorf("error %q names a Go type (%q)", err, goText)
		}
	}
}

func schemaStructs(typ reflect.Type, seen map[reflect.Type]bool) {
	for typ.Kind() == reflect.Pointer || typ.Kind() == reflect.Slice || typ.Kind() == reflect.Map {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || seen[typ] {
		return
	}
	seen[typ] = true
	for field := range typ.Fields() {
		schemaStructs(field.Type, seen)
	}
}

func TestSchemaFieldsHaveExplicitTags(t *testing.T) {
	t.Parallel()

	seen := map[reflect.Type]bool{}
	schemaStructs(reflect.TypeFor[File](), seen)
	if len(seen) < 4 {
		t.Fatalf("visited %d struct types, want the whole schema", len(seen))
	}
	for typ := range seen {
		for field := range typ.Fields() {
			tag, hasTag := field.Tag.Lookup("yaml")
			name, options, _ := strings.Cut(tag, ",")
			if !hasTag || name == "" || strings.Contains(options, "inline") {
				t.Errorf("%s.%s needs an explicit, non-inline yaml tag", typ.Name(), field.Name)
			}
		}
	}
}
