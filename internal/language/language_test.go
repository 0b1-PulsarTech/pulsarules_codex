package language

import (
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func TestLanguages_EveryExtensionHasOneOwner(t *testing.T) {
	t.Parallel()

	owner := map[string]ID{}
	ids := map[ID]bool{}
	for _, lang := range languages() {
		if ids[lang.ID] {
			t.Errorf("language %q is listed twice", lang.ID)
		}
		ids[lang.ID] = true
		if len(lang.extensions) == 0 {
			t.Errorf("language %q has no extension", lang.ID)
		}
		if len(lang.Comments.Line) == 0 && len(lang.Comments.Blocks) == 0 {
			t.Errorf("language %q has no comment grammar", lang.ID)
		}
		for _, ext := range lang.extensions {
			if previous, taken := owner[ext]; taken {
				t.Errorf("extension %q is listed for %q and %q", ext, previous, lang.ID)
			}
			owner[ext] = lang.ID
		}
	}
}

func TestForExtension(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext        string
		wantID     ID
		wantPrefix string
		wantNested bool
	}{
		{".go", Go, "//", false},
		{".GO", Go, "//", false},
		{".dart", Dart, "//", true},
		{".ts", TypeScript, "//", false},
		{".tsx", TypeScript, "//", false},
		{".mts", TypeScript, "//", false},
		{".cts", TypeScript, "//", false},
		{".js", JavaScript, "//", false},
		{".mjs", JavaScript, "//", false},
		{".cjs", JavaScript, "//", false},
		{".jsx", JavaScript, "//", false},
		{".svelte", Svelte, "//", false},
		{".vue", Vue, "//", false},
		{".rs", Rust, "//", false},
		{".java", Java, "//", false},
		{".py", Python, "#", false},
		{".sh", Shell, "#", false},
		{".yaml", YAML, "#", false},
		{".toml", TOML, "#", false},
		{".sql", SQL, "--", false},
		{".nix", Nix, "#", false},
		{".lua", Lua, "--", false},
		{".html", HTML, "<!--", false},
		{".xml", "", "", false},
		{"", "", "", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()
			lang := ForExtension(testCase.ext)
			if lang.ID != testCase.wantID {
				t.Fatalf(
					"ForExtension(%q).ID = %q, want %q",
					testCase.ext,
					lang.ID,
					testCase.wantID,
				)
			}
			if lang.ID == "" {
				return
			}
			isNested := slices.ContainsFunc(
				lang.Comments.Blocks,
				func(block lexer.BlockDelim) bool {
					return block.Nested
				},
			)
			if isNested != testCase.wantNested {
				t.Errorf("nested block comments = %v, want %v", isNested, testCase.wantNested)
			}
		})
	}
}

func TestSourceExtensions(t *testing.T) {
	t.Parallel()

	source := SourceExtensions()
	for _, want := range []string{".go", ".dart", ".ts", ".mts", ".cts", ".nix", ".sql", ".svelte"} {
		if !slices.Contains(source, want) {
			t.Errorf("SourceExtensions() lacks %q", want)
		}
	}
	for _, unwanted := range []string{".yaml", ".yml", ".toml", ".xml"} {
		if slices.Contains(source, unwanted) {
			t.Errorf("SourceExtensions() holds %q, config is not source", unwanted)
		}
	}
	source[0] = "mutated"
	if SourceExtensions()[0] == "mutated" {
		t.Fatal("SourceExtensions must build a fresh slice on each call")
	}
}

func TestExtensions(t *testing.T) {
	t.Parallel()

	got := Extensions(TypeScript, Proto)
	want := []string{".ts", ".tsx", ".mts", ".cts", ".proto"}
	if !slices.Equal(got, want) {
		t.Fatalf("Extensions(TypeScript, Proto) = %v, want %v", got, want)
	}
	if none := Extensions(); len(none) != 0 {
		t.Fatalf("Extensions() = %v, want none", none)
	}
}

func TestLanguage_IsPackageClause(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		line string
		want bool
	}{
		{"go package", ".go", "package foo", true},
		{"go indented", ".go", "  package foo", true},
		{"go glued word", ".go", "packagefoobar", false},
		{"go comment", ".go", "// package foo", false},
		{"sql schema", ".sql", "create schema app;", true},
		{"sql select", ".sql", "SELECT 1;", false},
		{"dart has none", ".dart", "package foo", false},
		{"python has none", ".py", "package foo", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := ForExtension(
				testCase.ext,
			).IsPackageClause(testCase.line); got != testCase.want {
				t.Errorf("IsPackageClause(%q) = %v, want %v", testCase.line, got, testCase.want)
			}
		})
	}
}
