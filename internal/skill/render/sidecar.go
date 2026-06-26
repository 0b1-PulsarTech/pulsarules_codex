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
