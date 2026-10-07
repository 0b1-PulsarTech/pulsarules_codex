package pass

import (
	"strings"
)

const ParamExtensions = "extensions"

func DefaultSourceExtensions() []string {
	// YAML, TOML and XML stay out: flagging config prose trains people to ignore the gate.
	return []string{
		".go",
		".dart",
		".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs", ".svelte", ".vue",
		".java", ".kt", ".kts", ".scala", ".swift",
		".rs", ".c", ".h", ".cpp", ".hpp", ".cs",
		".py", ".rb", ".php", ".sh", ".bash", ".zsh",
		".sql", ".proto", ".lua", ".html", ".nix",
	}
}

func ExtensionFilter(
	ctx *Context, analyzerID string, fallback []string,
) func(FileChange) bool {
	configured := ctx.Params(analyzerID).Strings(ParamExtensions, fallback)
	allowed := make(map[string]bool, len(configured))
	for _, ext := range configured {
		if normalized := normalizeExtension(ext); normalized != "" {
			allowed[normalized] = true
		}
	}
	return func(fc FileChange) bool { return allowed[strings.ToLower(fc.Extension)] }
}

func normalizeExtension(ext string) string {
	trimmed := strings.ToLower(strings.TrimSpace(ext))
	if trimmed == "" || trimmed == "." {
		return ""
	}
	if strings.HasPrefix(trimmed, ".") {
		return trimmed
	}
	return "." + trimmed
}
