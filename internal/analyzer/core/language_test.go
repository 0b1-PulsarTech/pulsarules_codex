package core

import "testing"

func TestLanguageRegistry(t *testing.T) {
	t.Parallel()

	reg := NewLanguageRegistry()
	if reg.Lookup(".go") != nil {
		t.Error("expected nil for unregistered language")
	}
}

func TestLanguageRegistry_RegisterAndLookup(t *testing.T) {
	t.Parallel()

	reg := NewLanguageRegistry()
	mock := mockLang{exts: []string{".go"}, prefix: "//"}
	reg.Register(mock)

	got := reg.Lookup(".go")
	if got == nil {
		t.Fatal("expected non-nil for .go")
	}
	if got.CommentPrefix() != "//" {
		t.Errorf("expected comment prefix //, got %q", got.CommentPrefix())
	}
}

type mockLang struct {
	exts   []string
	prefix string
}

func (lang mockLang) Extensions() []string  { return lang.exts }
func (lang mockLang) CommentPrefix() string { return lang.prefix }

func (lang mockLang) CommentSyntax() CommentSyntax {
	return CommentSyntax{Line: []string{lang.prefix}}
}
func (lang mockLang) IsPackageDeclaration(string) bool { return false }
