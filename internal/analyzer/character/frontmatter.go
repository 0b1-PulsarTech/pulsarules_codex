package character

import (
	"fmt"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/markdown"
)

func isProvenanceKey(key string) bool {
	switch key {
	// "model" and "llm" are absent: a legitimate frontmatter of ours could
	// use either word about its own subject.
	case "generator", "ai", "ai_generated", "ai-generated",
		"claude", "anthropic", "openai", "gemini",
		"synthid", "c2pa", "content_credentials",
		"provenance", "digital_source_type", "created_with":
		return true
	}
	return false
}

func frontmatterFindings(
	fc pass.FileChange,
	src string,
	reporter finding.Reporter,
) []finding.Finding {
	if !markdown.IsMarkdown(fc.Extension) {
		return nil
	}
	var findings []finding.Finding
	// Key position only, never body prose: a document that merely discusses a vendor
	// is not machine-generated.
	for offset, line := range frontmatterLines(src) {
		key, _, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if !isProvenanceKey(key) {
			continue
		}
		findings = append(findings, reporter.At(
			fc.Path, offset+2,
			fmt.Sprintf("frontmatter key %q names a generating machine", key),
			"delete the key; provenance metadata does not belong in a source document",
		))
	}
	return findings
}

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
		// An indented line is nested under its parent key, and a nested "claude:" is a
		// value's field, not the document's own provenance.
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '-' {
			top = append(top, "")
			continue
		}
		top = append(top, line)
	}
	return top
}
