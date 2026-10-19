package render

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

type bodyParser struct {
	mu    sync.Mutex
	funcs template.FuncMap
	cache map[string]*template.Template
}

func newBodyParser() *bodyParser {
	return &bodyParser{funcs: funcs(), cache: map[string]*template.Template{}}
}

func (parser *bodyParser) parse(body string) (*template.Template, error) {
	parser.mu.Lock()
	defer parser.mu.Unlock()
	if cached, ok := parser.cache[body]; ok {
		return cached, nil
	}
	// Each body parses in its own namespace so bare {{define "must"}} names do not collide.
	tmpl, err := template.New("body").Funcs(parser.funcs).Parse(body)
	if err != nil {
		return nil, fmt.Errorf("parse body: %w", err)
	}
	parser.cache[body] = tmpl
	return tmpl, nil
}

func (parser *bodyParser) sections(body string) (map[string]string, error) {
	parsed, err := parser.parse(body)
	if err != nil {
		return nil, err
	}
	return renderSections(parsed)
}

func renderSections(tmpl *template.Template) (map[string]string, error) {
	byKey := map[string]string{}
	for _, canonical := range canonicalSections() {
		if tmpl.Lookup(canonical.Key) == nil {
			continue
		}
		var buf bytes.Buffer
		if execErr := tmpl.ExecuteTemplate(&buf, canonical.Key, nil); execErr != nil {
			return nil, fmt.Errorf("render section %q: %w", canonical.Key, execErr)
		}
		byKey[canonical.Key] = strings.TrimSpace(buf.String())
	}
	return byKey, nil
}

func (parser *bodyParser) whole(body string) (string, error) {
	parsed, err := parser.parse(body)
	if err != nil {
		return "", err
	}
	var preamble bytes.Buffer
	if execErr := parsed.Execute(&preamble, nil); execErr != nil {
		return "", fmt.Errorf("render preamble: %w", execErr)
	}
	sections, err := renderSections(parsed)
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

func (parser *bodyParser) sidecar(body string) (string, error) {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return "", nil
	}
	parsed, err := template.New("sidecar").Funcs(parser.funcs).Parse(trimmed)
	if err != nil {
		return "", fmt.Errorf("parse sidecar: %w", err)
	}
	var buf bytes.Buffer
	if execErr := parsed.Execute(&buf, nil); execErr != nil {
		return "", fmt.Errorf("render sidecar: %w", execErr)
	}
	return buf.String(), nil
}

func UnknownSections(body string) ([]string, error) {
	parsed, err := newBodyParser().parse(body)
	if err != nil {
		return nil, err
	}
	canonical := map[string]bool{}
	for _, section := range canonicalSections() {
		canonical[section.Key] = true
	}
	var names []string
	for _, assoc := range parsed.Templates() {
		if name := assoc.Name(); name != "body" && !canonical[name] {
			names = append(names, name)
		}
	}
	return names, nil
}
