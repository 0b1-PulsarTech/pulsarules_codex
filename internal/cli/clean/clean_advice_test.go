package clean

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
)

func TestAdvice(t *testing.T) {
	t.Parallel()

	const (
		phrase    = "write it as the escape \\u2014"
		caveat    = "inside a string literal it is data: " + phrase
		removable = "removable outside a string literal; inside one, " + phrase
	)
	testCases := []struct {
		name  string
		ext   string
		class invisible.Class
		want  string
	}{
		{
			"typographic",
			".go",
			invisible.ClassTypographic,
			"replace with its ASCII form by hand; " + caveat,
		},
		{"strip", ".go", invisible.ClassStrip, removable},
		{"space", ".go", invisible.ClassSpace, removable},
		{
			"contextual",
			".go",
			invisible.ClassContextual,
			"may be load-bearing; judge it before removing",
		},
		{"unknown", ".go", invisible.Class(250), "unclassified"},
		{
			"prose typographic",
			".md",
			invisible.ClassTypographic,
			"replace with its ASCII form by hand",
		},
		{"prose strip", ".md", invisible.ClassStrip, "removable outside possible code"},
		{"prose space", ".md", invisible.ClassSpace, "removable outside possible code"},
		{
			"prose contextual",
			".md",
			invisible.ClassContextual,
			"may be load-bearing; judge it before removing",
		},
		{"prose unknown", ".md", invisible.Class(250), "unclassified"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := advice(testCase.class, testCase.ext, phrase); got != testCase.want {
				t.Errorf("advice(%v) = %q, want %q", testCase.class, got, testCase.want)
			}
		})
	}
}

func TestSweep_EscapePhraseIsTheSharedOne(t *testing.T) {
	t.Parallel()

	const phrase = "write it as the escape \\u2014"
	testCases := []struct {
		name string
		file string
		body string
		want string
	}{
		{
			"held in a string literal",
			"m.go",
			"package m\n\nvar c = \"x\u200By\"\n",
			"(in a string literal, write it as the escape \\u200B; clean leaves it)",
		},
		{
			"not held, outside any literal",
			"m.go",
			"package m\n\n// x\u2014y\n",
			"(replace with its ASCII form by hand; inside a string literal it is data: " +
				phrase + ")",
		},
		{
			"held in a raw string",
			"m.go",
			"package m\n\nvar c = `x\u200By`\n",
			"(a raw string cannot hold an escape: move it into an " +
				"interpreted string as \\u200B; clean leaves it)",
		},
		{
			"markdown prose has no string literal",
			"m.md",
			"a\u2014b\n",
			"(replace with its ASCII form by hand)",
		},
		{
			"markdown prose astral mark",
			"m.md",
			"a\U0001D173b\n",
			"(removable outside possible code)",
		},
		{
			"markdown fence holds an astral mark",
			"m.md",
			"```\nx\U0001D173y\n```\n",
			"(in possible code: judge it by hand; clean leaves it)",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root := t.TempDir()
			if err := os.WriteFile(
				filepath.Join(root, testCase.file), []byte(testCase.body), disk.ModeFile,
			); err != nil {
				t.Fatalf("write: %v", err)
			}
			var out strings.Builder
			if err := sweep(&out, root, inspecting(), nil); err != nil {
				t.Fatalf("sweep: %v", err)
			}
			if !strings.Contains(out.String(), testCase.want) {
				t.Errorf("output lacks %q:\n%s", testCase.want, out.String())
			}
			if strings.Contains(out.String(), "%!") {
				t.Errorf("output carries a fmt error:\n%s", out.String())
			}
		})
	}
}
