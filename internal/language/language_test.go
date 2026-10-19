package language

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type extensionCase struct {
	ext      string
	wantID   ID
	wantSyn  lexer.CommentSyntax
	isListed bool
}

func extensionCases() []extensionCase {
	slash := lexer.CommentSyntax{
		Line:   []string{"//"},
		Blocks: []lexer.BlockDelim{{Open: "/*", Close: "*/"}},
	}
	dart := lexer.CommentSyntax{
		Line:   []string{"//"},
		Blocks: []lexer.BlockDelim{{Open: "/*", Close: "*/", Nested: true}},
	}
	component := lexer.CommentSyntax{
		Line: []string{"//"},
		Blocks: []lexer.BlockDelim{
			{Open: "/*", Close: "*/"}, {Open: "<!--", Close: "-->"},
		},
	}
	hash := lexer.CommentSyntax{Line: []string{"#"}}
	nix := lexer.CommentSyntax{Line: []string{"#"}, Blocks: slash.Blocks}
	sql := lexer.CommentSyntax{Line: []string{"--"}, Blocks: slash.Blocks}
	lua := lexer.CommentSyntax{
		Line:   []string{"--"},
		Blocks: []lexer.BlockDelim{{Open: "--[[", Close: "]]"}},
	}
	markup := lexer.CommentSyntax{Blocks: []lexer.BlockDelim{{Open: "<!--", Close: "-->"}}}
	return []extensionCase{
		{".go", Go, slash, true},
		{".dart", Dart, dart, true},
		{".ts", TypeScript, slash, true},
		{".tsx", TypeScript, slash, true},
		{".mts", TypeScript, slash, true},
		{".cts", TypeScript, slash, true},
		{".js", JavaScript, slash, true},
		{".jsx", JavaScript, slash, true},
		{".mjs", JavaScript, slash, true},
		{".cjs", JavaScript, slash, true},
		{".svelte", Svelte, component, true},
		{".vue", Vue, component, true},
		{".java", Java, slash, true},
		{".kt", Kotlin, slash, true},
		{".kts", Kotlin, slash, true},
		{".scala", Scala, slash, true},
		{".swift", Swift, slash, true},
		{".rs", Rust, slash, true},
		{".c", CLang, slash, true},
		{".h", CLang, slash, true},
		{".cpp", CPP, slash, true},
		{".hpp", CPP, slash, true},
		{".cs", CSharp, slash, true},
		{".py", Python, hash, true},
		{".rb", Ruby, hash, true},
		{".php", PHP, slash, true},
		{".sh", Shell, hash, true},
		{".bash", Shell, hash, true},
		{".zsh", Shell, hash, true},
		{".sql", SQL, sql, true},
		{".proto", Proto, slash, true},
		{".lua", Lua, lua, true},
		{".html", HTML, markup, true},
		{".nix", Nix, nix, true},
		{".yaml", YAML, hash, true},
		{".yml", YAML, hash, true},
		{".toml", TOML, hash, true},
		{".GO", Go, slash, false},
		{".Dart", Dart, dart, false},
		{".xml", "", lexer.CommentSyntax{}, false},
		{"", "", lexer.CommentSyntax{}, false},
	}
}

func TestForExtension(t *testing.T) {
	t.Parallel()

	testCases := extensionCases()
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
			if !reflect.DeepEqual(lang.Comments, testCase.wantSyn) {
				t.Fatalf("ForExtension(%q).Comments = %+v, want %+v",
					testCase.ext, lang.Comments, testCase.wantSyn)
			}
		})
	}
	listed := map[string]bool{}
	for _, testCase := range testCases {
		listed[testCase.ext] = testCase.isListed
	}
	for _, lang := range languages() {
		for _, ext := range lang.extensions {
			if !listed[ext] {
				t.Errorf("extension %q of %q has no row in this table", ext, lang.ID)
			}
		}
	}
}

func TestSourceExtensions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ext  string
		want bool
	}{
		{"go", ".go", true},
		{"dart", ".dart", true},
		{"mts", ".mts", true},
		{"cts", ".cts", true},
		{"nix", ".nix", true},
		{"sql", ".sql", true},
		{"svelte", ".svelte", true},
		{"yaml is config", ".yaml", false},
		{"yml is config", ".yml", false},
		{"toml is config", ".toml", false},
		{"xml is unknown", ".xml", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := slices.Contains(SourceExtensions(), testCase.ext); got != testCase.want {
				t.Fatalf(
					"SourceExtensions() contains %q = %v, want %v",
					testCase.ext,
					got,
					testCase.want,
				)
			}
		})
	}
}

func TestSourceExtensions_FreshSlice(t *testing.T) {
	t.Parallel()

	source := SourceExtensions()
	source[0] = "mutated"
	if SourceExtensions()[0] == "mutated" {
		t.Fatal("SourceExtensions must build a fresh slice on each call")
	}
}

func TestExtensions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		ids  []ID
		want []string
	}{
		{"table order", []ID{TypeScript, Proto}, []string{".ts", ".tsx", ".mts", ".cts", ".proto"}},
		{"one id", []ID{Go}, []string{".go"}},
		{"unknown id", []ID{"cobol"}, nil},
		{"no ids", nil, nil},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := Extensions(testCase.ids...); !slices.Equal(got, testCase.want) {
				t.Fatalf("Extensions(%v) = %v, want %v", testCase.ids, got, testCase.want)
			}
		})
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
