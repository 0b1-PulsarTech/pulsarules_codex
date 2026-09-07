package core

import "strings"

const ParamExtensions = "extensions"

func ExtensionFilter(
	ctx *AnalysisContext, analyzerID string, fallback []string,
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
