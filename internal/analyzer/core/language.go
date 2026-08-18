package core

// Language is the consumer-declared port every language handler implements.
// A language handler knows how to parse a source file and extract language-
// specific constructs (comments, declarations, imports) so that generic
// analyzers can run language-agnostically. The handler is a Strategy selected
// by file extension at pipeline build time.
type Language interface {
	// ID returns the language identifier (e.g. "go", "sql", "proto").
	ID() string
	// Extensions returns the file extensions this handler covers (e.g. ".go").
	Extensions() []string
	// IsCommentLine reports whether a line is a comment line in this language.
	IsCommentLine(line string) bool
	// IsBlankLine reports whether a line is blank (whitespace only).
	IsBlankLine(line string) bool
	// CommentPrefix returns the comment prefix for this language (e.g. "//").
	CommentPrefix() string
	// IsPackageDeclaration reports whether a line is the package/declaration
	// line in this language. For Go this is `package foo`; for SQL, `CREATE
	// SCHEMA`; etc. Used by the top-of-file comment analyzer.
	IsPackageDeclaration(line string) bool
}

// LanguageRegistry maps file extensions to their language handlers. Build it
// once at boot and inject it into analyzers that need language detection.
type LanguageRegistry struct {
	byExt map[string]Language
}

func NewLanguageRegistry() *LanguageRegistry {
	return &LanguageRegistry{byExt: map[string]Language{}}
}

// Register adds a language handler to the registry, keyed by each of its
// extensions.
func (r *LanguageRegistry) Register(lang Language) {
	for _, ext := range lang.Extensions() {
		r.byExt[ext] = lang
	}
}

//nolint:ireturn // registry lookup returns interface by design
func (r *LanguageRegistry) Lookup(ext string) Language {
	return r.byExt[ext]
}
