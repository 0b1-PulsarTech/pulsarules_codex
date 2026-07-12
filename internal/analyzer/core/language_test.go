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
	mock := mockLang{exts: []string{".go"}, id: "go"}
	reg.Register(mock)

	got := reg.Lookup(".go")
	if got == nil {
		t.Fatal("expected non-nil for .go")
	}
	if got.ID() != "go" {
		t.Errorf("expected id go, got %q", got.ID())
	}
}

type mockLang struct {
	exts []string
	id   string
}

func (lang mockLang) ID() string                       { return lang.id }
func (lang mockLang) Extensions() []string             { return lang.exts }
func (lang mockLang) IsCommentLine(string) bool        { return false }
func (lang mockLang) IsPackageDeclaration(string) bool { return false }
