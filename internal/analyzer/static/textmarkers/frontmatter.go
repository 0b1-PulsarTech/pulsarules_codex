package textmarkers

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

var frontmatterReporter = core.NewReporter("ai-frontmatter", core.SeverityError, core.CatSyntax)

// provenanceKeys are the frontmatter keys that name a generating machine.
//
// why: "model" and "llm" are deliberately absent from the upstream list - a
// legitimate frontmatter of ours could use either word about its own subject.
var provenanceKeys = map[string]bool{
	"generator": true, "ai": true, "ai_generated": true, "ai-generated": true,
	"claude": true, "anthropic": true, "openai": true, "gemini": true,
	"synthid": true, "c2pa": true, "content_credentials": true,
	"provenance": true, "digital_source_type": true, "created_with": true,
}

// frontmatterFindings reports a provenance key in a leading YAML block.
//
// why: key position in the frontmatter only, never body prose. A document that
// merely discusses Claude is not machine-generated, and scanning the body is
// the false positive this check exists to avoid.
func frontmatterFindings(fc core.FileChange, src string) []core.Finding {
	if strings.ToLower(fc.Extension) != ".md" {
		return nil
	}
	var findings []core.Finding
	for offset, line := range frontmatterLines(src) {
		key, _, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if !provenanceKeys[key] {
			continue
		}
		findings = append(findings, frontmatterReporter.At(
			fc.Path, offset+2,
			fmt.Sprintf("frontmatter key %q names a generating machine", key),
			"delete the key; provenance metadata does not belong in a source document",
		))
	}
	return findings
}

// frontmatterLines returns the top-level lines of a leading --- block, or nil.
func frontmatterLines(src string) []string {
	rest, found := strings.CutPrefix(src, "---\n")
	if !found {
		return nil
	}
	body, _, found := strings.Cut(rest, "\n---")
	if !found {
		return nil
	}
	var top []string
	for line := range strings.SplitSeq(body, "\n") {
		// why: an indented line is nested under its parent key, and a nested
		// "claude:" is a value's field, not the document's own provenance.
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '-' {
			top = append(top, "")
			continue
		}
		top = append(top, line)
	}
	return top
}
