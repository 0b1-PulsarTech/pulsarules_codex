package banner

import (
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/lexer"
)

func annotationBannerLine(lines []string) (int, bool) {
	annotated := false
	for pos := 0; pos < len(lines); {
		trimmed := strings.TrimSpace(lines[pos])
		span := 0
		if annotated {
			span = lexer.ExemptHeaderSpan(lines[pos:])
		}
		switch {
		case span > 0:
			pos += span
		case trimmed == "":
			pos++
		case strings.HasPrefix(trimmed, "/**"):
			next, rest, ok := annotationBlock(lines, pos)
			if !ok {
				return pos, false
			}
			annotated = true
			if rest != "" {
				return next - 1, isCode(rest)
			}
			pos = next
		default:
			return pos, annotated && isCode(trimmed)
		}
	}
	return 0, false
}

func annotationBlock(lines []string, start int) (int, string, bool) {
	end, closed := lexer.BlockEnd(lines[start:], lexer.BlockDelim{Open: "/**", Close: "*/"})
	if !closed {
		return 0, "", false
	}
	contents, rest := blockContents(lines[start : start+end+1])
	return start + end + 1, rest, onlyAnnotationTags(contents)
}

func blockContents(block []string) (contents []string, rest string) {
	for idx, line := range block {
		content := strings.TrimSpace(line)
		if idx == 0 {
			content = strings.TrimPrefix(content, "/**")
		}
		if idx == len(block)-1 {
			content, rest, _ = strings.Cut(content, "*/")
			rest = strings.TrimSpace(rest)
		}
		content = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(content), "*"))
		if content != "" {
			contents = append(contents, content)
		}
	}
	return contents, rest
}

func onlyAnnotationTags(contents []string) bool {
	var groups []string
	for _, content := range contents {
		if strings.HasPrefix(content, "@") || len(groups) == 0 {
			groups = append(groups, content)
			continue
		}
		groups[len(groups)-1] += " " + content
	}
	return len(groups) > 0 && !slices.ContainsFunc(groups, func(group string) bool {
		return !annotationOnly(group)
	})
}

func isCode(trimmed string) bool {
	return !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*")
}
