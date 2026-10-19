package language

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

type Language struct {
	ID       ID
	Comments lexer.CommentSyntax

	extensions    []string
	isConfig      bool
	strings       lexer.StringSyntax
	astral        string
	packageClause func(line string) bool
}

func ForExtension(ext string) Language {
	ext = strings.ToLower(ext)
	for _, lang := range languages() {
		if slices.Contains(lang.extensions, ext) {
			return lang
		}
	}
	return Language{}
}

func Extensions(ids ...ID) []string {
	var exts []string
	for _, lang := range languages() {
		if slices.Contains(ids, lang.ID) {
			exts = append(exts, lang.extensions...)
		}
	}
	return exts
}

func SourceExtensions() []string {
	var exts []string
	for _, lang := range languages() {
		if !lang.isConfig {
			exts = append(exts, lang.extensions...)
		}
	}
	return exts
}

func (lang Language) CommentPrefix() string {
	if len(lang.Comments.Line) > 0 {
		return lang.Comments.Line[0]
	}
	return lang.Comments.Blocks[0].Open
}

func (lang Language) IsPackageClause(line string) bool {
	return lang.packageClause != nil && lang.packageClause(line)
}
