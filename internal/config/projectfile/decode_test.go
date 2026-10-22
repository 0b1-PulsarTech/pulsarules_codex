package projectfile

import (
	"strings"
	"testing"
)

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
			_, err := decodeFixture([]byte(testCase.content))
			if err == nil || err.Error() != testCase.want {
				t.Fatalf("decodeFixture() error = %v, want %q", err, testCase.want)
			}
			assertPlainFixture(t, err)
			if strings.Contains(err.Error(), ".yaml: yaml:") {
				t.Errorf("decodeFixture() error %q keeps the yaml: prefix", err)
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
			file, err := decodeFixture([]byte(testCase.content))
			if err != nil || file.Version != 1 {
				t.Fatalf("decodeFixture() = %+v, %v, want version 1", file, err)
			}
		})
	}
}
