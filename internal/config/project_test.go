package config

import (
	"reflect"
	"strings"
	"testing"
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

func decodeFile(content []byte) (*ProjectFile, error) {
	parsed, err := decode(ProjectFileName, content)
	if err != nil {
		return nil, err
	}
	if err = parsed.problems.err(); err != nil {
		return nil, err
	}
	return parsed.file, nil
}

func TestDecodeFile(t *testing.T) {
	t.Parallel()

	disabled := false
	included := true
	want := &ProjectFile{
		Path:             ProjectFileName,
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
			Analyzers: map[string]OverrideRules{
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

func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{"empty file", "", ".pulsarules.yaml: version: required"},
		{"comment only", "# nothing\n", ".pulsarules.yaml: version: required"},
		{
			"not yaml",
			"version: [",
			".pulsarules.yaml: line 1: did not find expected node content",
		},
		{
			"bad indent",
			"version: 1\n  bad: indent\n",
			".pulsarules.yaml: line 2: mapping values are not allowed in this context",
		},
		{
			"second document",
			"version: 1\n---\nversion: 1\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
		{
			"second document with a bare null",
			"version: 1\n---\nnull\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
		{
			"second document that is not yaml",
			"version: 1\n---\n  : bad: [\n",
			".pulsarules.yaml: only one YAML document is allowed",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			_, err := decodeFile([]byte(testCase.content))
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("decodeFile() error = %v, want %q", err, testCase.want)
			}
			assertPlainMessage(t, err)
			if strings.Contains(err.Error(), ".yaml: yaml:") {
				t.Errorf("decodeFile() error %q keeps the yaml: prefix", err)
			}
		})
	}
}

func TestDecodeTrailingEmptyDocuments(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		content string
	}{
		{"separator", "version: 1\n---\n"},
		{"separator and comment", "version: 1\n---\n# trailing\n"},
		{"two separators", "version: 1\n---\n---\n"},
		{"document end", "version: 1\n...\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil || file.Version != 1 {
				t.Fatalf("decodeFile() = %+v, %v, want version 1", file, err)
			}
		})
	}
}

type parseCase struct {
	name    string
	content string
	want    []string
}
