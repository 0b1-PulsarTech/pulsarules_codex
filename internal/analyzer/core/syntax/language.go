package syntax

import "strings"

func LanguageForExtension(ext string) Language {
	switch strings.ToLower(ext) {
	case ".go":
		return LangGo
	case ".dart":
		return LangDart
	case ".ts", ".tsx":
		return LangTypeScript
	case ".js", ".mjs", ".cjs", ".jsx":
		return LangJavaScript
	case ".svelte":
		return LangSvelte
	}
	return ""
}
