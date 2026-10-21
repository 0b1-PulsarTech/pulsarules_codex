package cleaner

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/invisible"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language"
)

type adviceCase struct {
	name string
	ext  string
	body string
	want string
}

func runAdviceCases(t *testing.T, testCases []adviceCase) {
	t.Helper()

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			found := invisible.Scan(testCase.body)
			if len(found) != 1 {
				t.Fatalf("Scan found %d markers, want 1", len(found))
			}
			literals := language.ScanLiterals(testCase.ext, testCase.body)
			got := EscapeAdvice(literals, found[0])
			if got != testCase.want {
				t.Errorf("EscapeAdvice = %q, want %q", got, testCase.want)
			}
			if strings.Contains(got, "%!") {
				t.Errorf("EscapeAdvice carries a fmt error: %q", got)
			}
		})
	}
}

func TestEscapeAdvice(t *testing.T) {
	t.Parallel()

	runAdviceCases(t, []adviceCase{
		{"interpreted string", ".go", "var a = \"x\u200By\"\n", "write it as the escape \\u200B"},
		{
			"raw string",
			".go",
			"var a = `x\u200By`\n",
			"a raw string cannot hold an escape: move it into an interpreted string as \\u200B",
		},
		{
			"astral regex needs the u flag",
			".mjs",
			"const re = /a\U000E0001/;\n",
			"write it as the escape \\u{E0001} and add the u flag",
		},
		{"bmp regex", ".mjs", "const re = /a\u200B/;\n", "write it as the escape \\u200B"},
		{"outside a literal", ".go", "var a = 1 // x\u200By\n", "write it as the escape \\u200B"},
		{
			"go astral",
			".go",
			"var a = \"x\U0001D173y\"\n",
			"write it as the escape \\U0001D173",
		},
	})
}

func TestEscapeAdvice_UnlexedFilesNameNoSyntax(t *testing.T) {
	t.Parallel()

	const (
		ordinary = "write the language's escape for U+200B instead"
		astral   = "write the language's escape for U+1D173 instead"
	)
	runAdviceCases(t, []adviceCase{
		{"rust", ".rs", "let a = \"x\u200By\";\n", ordinary},
		{"rust astral", ".rs", "let a = \"x\U0001D173y\";\n", astral},
		{"rust raw string", ".rs", "let a = r\"x\U0001D173y\";\n", astral},
		{"python", ".py", "a = 'x\u200By'\n", ordinary},
		{"python astral", ".py", "a = 'x\U0001D173y'\n", astral},
		{"shell", ".sh", "echo \"x\u200By\"\n", ordinary},
		{"shell astral", ".sh", "echo \"x\U0001D173y\"\n", astral},
		{"markdown", ".md", "x\u200By\n", ordinary},
		{"markdown astral", ".md", "x\U0001D173y\n", astral},
		{"text", ".txt", "x\u200By\n", ordinary},
		{"text astral", ".txt", "x\U0001D173y\n", astral},
	})
}
