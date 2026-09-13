package language

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestLookup_UnknownNameReturnsNil(t *testing.T) {
	t.Parallel()

	if got := Lookup("cobol"); got != nil {
		t.Errorf("Lookup(%q) = %v, want nil", "cobol", got)
	}
}

func TestLookup_Grammar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		languageName string
		wantExts     []string
		wantPrefix   string
	}{
		{"go", "go", []string{".go"}, "//"},
		{"clike", "clike", []string{".dart", ".ts", ".java", ".rs"}, "//"},
		{"hash", "hash", []string{".py", ".rb", ".sh", ".yaml", ".toml"}, "#"},
		{"sql", "sql", []string{".sql"}, "--"},
		{"lua", "lua", []string{".lua"}, "--"},
		{"markup", "markup", []string{".html"}, "<!--"},
		{"sfc", "sfc", []string{".svelte", ".vue"}, "//"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			language := Lookup(testCase.languageName)
			if language == nil {
				t.Fatalf("Lookup(%q) = nil, want a handler", testCase.languageName)
			}
			if got := language.CommentPrefix(); got != testCase.wantPrefix {
				t.Errorf("CommentPrefix() = %q, want %q", got, testCase.wantPrefix)
			}
			exts := language.Extensions()
			for _, ext := range testCase.wantExts {
				if !slices.Contains(exts, ext) {
					t.Errorf("Extensions() = %v, want it to contain %q", exts, ext)
				}
			}
		})
	}
}

func TestLookup_MarkupExcludesXML(t *testing.T) {
	t.Parallel()

	language := Lookup("markup")
	if slices.Contains(language.Extensions(), ".xml") {
		t.Error(
			"markup registers .xml, but core.DefaultSourceExtensions excludes it; " +
				"no consumer could ever reach a .xml registration",
		)
	}
	if slices.Contains(pass.DefaultSourceExtensions(), ".xml") {
		t.Error(
			"core.DefaultSourceExtensions started including .xml; keep markup and it consistent",
		)
	}
}

func TestRegisterAll_RegistersEveryExtension(t *testing.T) {
	t.Parallel()

	reg := pass.NewLanguageRegistry()
	RegisterAll(reg)

	testCases := []string{
		".go",
		".dart", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
		".java", ".kt", ".kts", ".scala", ".swift",
		".rs", ".c", ".h", ".cpp", ".hpp", ".cs", ".php", ".proto",
		".py", ".rb", ".sh", ".bash", ".zsh", ".yaml", ".yml", ".toml",
		".sql", ".lua", ".html", ".svelte", ".vue",
	}
	for _, testCase := range testCases {
		if lang := reg.Lookup(testCase); lang == nil {
			t.Errorf("registry lookup for %q returned nil after RegisterAll", testCase)
		}
	}
}
