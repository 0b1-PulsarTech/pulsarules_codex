package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

func renderSidecar(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := template.New("sidecar").Funcs(funcs()).Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse sidecar: %w", err)
	}
	var buf bytes.Buffer
	if execErr := parsed.Execute(&buf, nil); execErr != nil {
		return "", fmt.Errorf("render sidecar: %w", execErr)
	}
	return buf.String(), nil
}

func wholeBody(body string) (string, error) {
	parsed, err := template.New("body").Funcs(funcs()).Parse(body)
	if err != nil {
		return "", fmt.Errorf("parse body: %w", err)
	}
	var preamble bytes.Buffer
	if execErr := parsed.Execute(&preamble, nil); execErr != nil {
		return "", fmt.Errorf("render preamble: %w", execErr)
	}
	sections, err := bodySections(body)
	if err != nil {
		return "", err
	}
	var out strings.Builder
	out.WriteString(strings.TrimSpace(preamble.String()))
	for _, canonical := range canonicalSections() {
		if sectionBody := sections[canonical.Key]; sectionBody != "" {
			out.WriteString("\n\n## " + canonical.Heading + "\n\n" + sectionBody)
		}
	}
	return out.String(), nil
}
