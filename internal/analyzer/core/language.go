package core

type Language interface {
	Extensions() []string
	CommentPrefix() string
	CommentSyntax() CommentSyntax
	IsPackageDeclaration(line string) bool
}

type LanguageRegistry struct {
	byExt map[string]Language
}

func NewLanguageRegistry() *LanguageRegistry {
	return &LanguageRegistry{byExt: map[string]Language{}}
}

func (registry *LanguageRegistry) Register(lang Language) {
	for _, ext := range lang.Extensions() {
		registry.byExt[ext] = lang
	}
}

//nolint:ireturn // registry lookup returns interface by design
func (registry *LanguageRegistry) Lookup(ext string) Language {
	return registry.byExt[ext]
}
