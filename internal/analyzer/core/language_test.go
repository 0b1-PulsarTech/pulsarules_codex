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

func (m mockLang) ID() string                       { return m.id }
func (m mockLang) Extensions() []string             { return m.exts }
func (m mockLang) IsCommentLine(string) bool        { return false }
func (m mockLang) IsBlankLine(string) bool          { return false }
func (m mockLang) CommentPrefix() string            { return "//" }
func (m mockLang) IsPackageDeclaration(string) bool { return false }
