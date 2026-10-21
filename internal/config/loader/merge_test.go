package loader

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

func parseFile(t *testing.T, name, content string) *projectfile.File {
	t.Helper()
	file, err := projectfile.Parse(name, []byte(content), testLoader().rules)
	if err != nil {
		t.Fatalf("Parse(%s) error = %v", name, err)
	}
	return file
}

type mergeCase struct {
	name, base, private string
	want                projectfile.File
}

func mergeCases() []mergeCase {
	return slices.Concat(
		mergeScalarCases(), mergeListCases(), mergeEntryCases(), mergeAnalyzerCases(),
	)
}

func mergeScalarCases() []mergeCase {
	return []mergeCase{
		{
			name:    "preset and include_generated: private wins when set",
			base:    "version: 1\npreset: strict\ninclude_generated: true\n",
			private: "version: 1\npreset: minimal\ninclude_generated: false\n",
			want:    projectfile.File{Preset: "minimal", IncludeGenerated: new(false)},
		},
		{
			name:    "preset and include_generated: base stays when private is silent",
			base:    "version: 1\npreset: strict\ninclude_generated: true\n",
			private: "version: 1\n",
			want:    projectfile.File{Preset: "strict", IncludeGenerated: new(true)},
		},
	}
}

func mergeListCases() []mergeCase {
	return []mergeCase{
		{
			name:    "skip_dirs and tests: union, base first, duplicates dropped",
			base:    "version: 1\nskip_dirs: [b, a]\ntests: [x/**, y/**]\n",
			private: "version: 1\nskip_dirs: [c, a, b]\ntests: [y/**, z/**]\n",
			want: projectfile.File{
				SkipDirs: []string{"b", "a", "c"}, Tests: []string{"x/**", "y/**", "z/**"},
			},
		},
		{
			name:    "overrides: base entries first, then private",
			base:    "version: 1\n" + overrideOf("a", "severity: info"),
			private: "version: 1\n" + overrideOf("b", "enabled: false"),
			want: projectfile.File{Overrides: []projectfile.OverrideEntry{
				{Files: []string{"a"}, Analyzers: map[string]projectfile.AnalyzerEntry{
					"file-size": {Severity: "info"},
				}},
				{Files: []string{"b"}, Analyzers: map[string]projectfile.AnalyzerEntry{
					"file-size": {Enabled: new(false)},
				}},
			}},
		},
		{
			name: "delegated: same id replaced whole, new ids appended",
			base: doc("delegated:", "  - {id: one, run: [a], format: dart-analyze-json, dir: x}",
				"  - {id: two, run: [b], format: dart-analyze-json}"),
			private: doc("delegated:", "  - {id: three, run: [d], format: dart-analyze-json}",
				"  - {id: one, run: [c], format: dart-analyze-json}"),
			want: projectfile.File{Delegated: []projectfile.DelegatedEntry{
				{ID: "one", Run: []string{"c"}, Format: "dart-analyze-json"},
				{ID: "two", Run: []string{"b"}, Format: "dart-analyze-json"},
				{ID: "three", Run: []string{"d"}, Format: "dart-analyze-json"},
			}},
		},
	}
}

func mergeEntryCases() []mergeCase {
	return []mergeCase{
		{
			name:    "overrides: base entries first, then private",
			base:    "version: 1\n" + overrideOf("a", "severity: info"),
			private: "version: 1\n" + overrideOf("b", "enabled: false"),
			want: projectfile.File{Overrides: []projectfile.OverrideEntry{
				{Files: []string{"a"}, Analyzers: map[string]projectfile.AnalyzerEntry{
					"file-size": {Severity: "info"},
				}},
				{Files: []string{"b"}, Analyzers: map[string]projectfile.AnalyzerEntry{
					"file-size": {Enabled: new(false)},
				}},
			}},
		},
		{
			name: "delegated: same id replaced whole, new ids appended",
			base: doc("delegated:", "  - {id: one, run: [a], format: dart-analyze-json, dir: x}",
				"  - {id: two, run: [b], format: dart-analyze-json}"),
			private: doc("delegated:", "  - {id: three, run: [d], format: dart-analyze-json}",
				"  - {id: one, run: [c], format: dart-analyze-json}"),
			want: projectfile.File{Delegated: []projectfile.DelegatedEntry{
				{ID: "one", Run: []string{"c"}, Format: "dart-analyze-json"},
				{ID: "two", Run: []string{"b"}, Format: "dart-analyze-json"},
				{ID: "three", Run: []string{"d"}, Format: "dart-analyze-json"},
			}},
		},
	}
}

func mergeAnalyzerCases() []mergeCase {
	return []mergeCase{
		{
			name: "analyzers: field by field, params per key",
			base: doc("analyzers:",
				"  file-size: {enabled: false, severity: error,",
				"    params: {max_lines: 1, extensions: [go]}}",
				"  doc-comment: {severity: info}"),
			private: doc("analyzers:", "  file-size: {enabled: true, params: {max_lines: 9}}",
				"  big-comment: {severity: warning}"),
			want: projectfile.File{Analyzers: map[string]projectfile.AnalyzerEntry{
				"file-size": {
					Enabled: new(true), Severity: "error",
					Params: map[string]any{"max_lines": 9, "extensions": []any{"go"}},
				},
				"doc-comment": {Severity: "info"},
				"big-comment": {Severity: "warning"},
			}},
		},
		{
			name:    "analyzers: both files set severity, private wins",
			base:    doc("analyzers: {file-size: {severity: error}}"),
			private: doc("analyzers: {file-size: {severity: info}}"),
			want: projectfile.File{Analyzers: map[string]projectfile.AnalyzerEntry{
				"file-size": {Severity: "info"},
			}},
		},
	}
}

func TestMerge(t *testing.T) {
	t.Parallel()

	for _, testCase := range mergeCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := parseFile(t, ".pulsarules.yaml", testCase.base)
			private := parseFile(t, ".git/pulsarules.yaml", testCase.private)
			got, err := layers{base: base, private: private, rules: testLoader().rules}.merge()
			if err != nil {
				t.Fatalf("merge() error = %v", err)
			}
			want := testCase.want
			want.Path = ".pulsarules.yaml + .git/pulsarules.yaml (merged)"
			want.Sources = []string{".pulsarules.yaml", ".git/pulsarules.yaml"}
			want.Version = 1
			if !reflect.DeepEqual(*got, want) {
				t.Fatalf("merge() = %+v, want %+v", *got, want)
			}
		})
	}
}

func doc(lines ...string) string {
	return "version: 1\n" + strings.Join(lines, "\n") + "\n"
}

func names(key string, count int) string {
	return strings.ReplaceAll(flowList(key, count), "/**", "")
}

func flowList(key string, count int) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = fmt.Sprintf("%s%04d/**", key, idx)
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func TestMergeLimits(t *testing.T) {
	t.Parallel()

	both := ".pulsarules.yaml + .git/pulsarules.yaml (merged): "
	testCases := []struct {
		name, base, private, want string
	}{
		{
			name:    "globs over the total",
			base:    doc("tests: " + flowList("a", 150)),
			private: doc("tests: " + flowList("b", 150)),
			want:    both + "overrides and tests: more than 256 globs in total",
		},
		{
			name:    "sequence over its entry cap",
			base:    doc("skip_dirs: " + names("a", 600)),
			private: doc("skip_dirs: " + names("b", 600)),
			want:    both + "skip_dirs: more than 1024 entries",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := parseFile(t, ".pulsarules.yaml", testCase.base)
			private := parseFile(t, ".git/pulsarules.yaml", testCase.private)
			_, err := layers{base: base, private: private, rules: testLoader().rules}.merge()
			assertErrorHas(t, err, []string{testCase.want})
		})
	}
}

func bombDoc() string {
	lines := []string{
		"analyzers:", "  big-comment:", "    params:", "      max_lines:",
		"        - &l0 [" + strings.Repeat("x", 1000) + "]",
		"        - &l1 [" + strings.TrimSuffix(strings.Repeat("*l0, ", 9), ", ") + "]",
		"        - &l2 [" + strings.TrimSuffix(strings.Repeat("*l1, ", 9), ", ") + "]",
	}
	return doc(lines...)
}

func denseDoc() string {
	inner := "[" + strings.TrimSuffix(strings.Repeat("ab,", 987), ",") + "]"
	outer := strings.TrimSuffix(strings.Repeat(inner+",", 81), ",")
	return doc("analyzers: {big-comment: {params: {max_lines: [" + outer + "]}}}")
}

type singleFileCase struct {
	name, base, private, single string
}

func singleFileCases() []singleFileCase {
	const tabbed = `["\tx\ny"]`
	return []singleFileCase{
		{
			name:    "tab-leading multi-line string in run",
			base:    doc("delegated: [{id: one, run: " + tabbed + ", format: dart-analyze-json}]"),
			private: doc("preset: minimal"),
			single: doc("preset: minimal",
				"delegated: [{id: one, run: "+tabbed+", format: dart-analyze-json}]"),
		},
		{
			name:    "tab-leading multi-line string in params",
			base:    doc("analyzers: {file-size: {params: {extensions: " + tabbed + "}}}"),
			private: doc("analyzers: {file-size: {severity: info}}"),
			single: doc("analyzers: {file-size: {severity: info,",
				"  params: {extensions: "+tabbed+"}}}"),
		},
		{
			name:    "leading newline",
			base:    doc("delegated: [{id: one, run: [\"\\nx\"], format: dart-analyze-json}]"),
			private: doc("preset: minimal"),
			single: doc("preset: minimal",
				"delegated: [{id: one, run: [\"\\nx\"], format: dart-analyze-json}]"),
		},
		{
			name:    "alias bomb in one layer",
			base:    bombDoc(),
			private: doc("preset: minimal"),
			single:  bombDoc() + "preset: minimal\n",
		},
		{
			name: "dense flow collections in both layers", base: denseDoc(),
			private: denseDoc(), single: denseDoc(),
		},
	}
}

func TestMergeMatchesSingleFile(t *testing.T) {
	t.Parallel()

	for _, testCase := range singleFileCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := parseFile(t, ".pulsarules.yaml", testCase.base)
			private := parseFile(t, ".git/pulsarules.yaml", testCase.private)
			got, err := layers{base: base, private: private, rules: testLoader().rules}.merge()
			if err != nil {
				t.Fatalf("merge() error = %v", err)
			}
			want := parseFile(t, "single.yaml", testCase.single)
			got.Path, got.Sources, want.Path = "", nil, ""
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("merge() = %+v, want %+v", *got, *want)
			}
		})
	}
}

func delegatedDoc(prefix string, count int, analyzed int) string {
	entries := make([]string, count)
	for idx := range entries {
		entries[idx] = fmt.Sprintf(
			"{id: %s%04d, run: [x], format: dart-analyze-json}", prefix, idx)
	}
	lines := []string{"delegated: [" + strings.Join(entries, ", ") + "]"}
	if analyzed > 0 {
		lines = append(lines, "analyzers: {"+tuned(prefix, 0, analyzed)+"}")
	}
	return doc(lines...)
}

func tuned(prefix string, from, count int) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = fmt.Sprintf("%s%04d: {severity: info}", prefix, from+idx)
	}
	return strings.Join(items, ", ")
}

type sizeCase struct {
	name, base, private, wantErr string
}

func mergeSizeCases() []sizeCase {
	const overCap = "(merged): "
	return []sizeCase{
		{
			name:    "analyzers 256 + 256 distinct",
			base:    delegatedDoc("d", 512, 256),
			private: doc("analyzers: {" + tuned("d", 256, 256) + "}"),
			wantErr: overCap + "analyzers: more than 256 entries",
		},
		{
			name:    "analyzers 257 distinct",
			base:    delegatedDoc("d", 257, 256),
			private: doc("analyzers: {" + tuned("d", 256, 1) + "}"),
			wantErr: overCap + "analyzers: more than 256 entries",
		},
		{
			name:    "analyzers 256 in total",
			base:    delegatedDoc("d", 256, 128),
			private: doc("analyzers: {" + tuned("d", 128, 128) + "}"),
		},
		{
			name:    "delegated 1000 + 1000",
			base:    delegatedDoc("a", 1000, 0),
			private: delegatedDoc("b", 1000, 0),
			wantErr: overCap + "delegated: more than 1024 entries",
		},
		{
			name: "delegated 1024 in total", base: delegatedDoc("a", 512, 0),
			private: delegatedDoc("b", 512, 0),
		},
		{
			name: "delegated 1025 in total", base: delegatedDoc("a", 512, 0),
			private: delegatedDoc("b", 513, 0),
			wantErr: overCap + "delegated: more than 1024 entries",
		},
		{
			name: "skip_dirs 1024 in total", base: doc("skip_dirs: " + names("a", 512)),
			private: doc("skip_dirs: " + names("b", 512)),
		},
		{
			name: "skip_dirs 1025 in total", base: doc("skip_dirs: " + names("a", 512)),
			private: doc("skip_dirs: " + names("b", 513)),
			wantErr: overCap + "skip_dirs: more than 1024 entries",
		},
	}
}

func TestMergeSizeBoundaries(t *testing.T) {
	t.Parallel()

	testCases := mergeSizeCases()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			base := parseFile(t, ".pulsarules.yaml", testCase.base)
			private, err := projectfile.Parse(
				".git/pulsarules.yaml",
				[]byte(testCase.private),
				privateRules(testLoader().rules, base),
			)
			if err != nil {
				t.Fatalf("Parse(private) error = %v", err)
			}
			_, err = layers{base: base, private: private, rules: testLoader().rules}.merge()
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("merge() error = %v", err)
				}
				return
			}
			assertErrorHas(t, err, []string{testCase.wantErr})
		})
	}
}
