package render

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"text/template"
)

type parsedBody struct{ tmpl *template.Template }

type bodyParser struct {
	mu    sync.Mutex
	funcs template.FuncMap
	cache map[string]parsedBody
}

func newBodyParser() *bodyParser {
	return &bodyParser{funcs: funcs(), cache: map[string]parsedBody{}}
}

func (parser *bodyParser) parse(body string) (parsedBody, error) {
	parser.mu.Lock()
	defer parser.mu.Unlock()
	if cached, ok := parser.cache[body]; ok {
		return cached, nil
	}
	// Each body parses in its own namespace so bare {{define "must"}} names do not collide.
	tmpl, err := template.New("body").Funcs(parser.funcs).Parse(body)
	if err != nil {
		return parsedBody{}, fmt.Errorf("parse body: %w", err)
	}
	parsed := parsedBody{tmpl: tmpl}
	parser.cache[body] = parsed
	return parsed, nil
}

func (parser *bodyParser) sections(body string) (map[string]string, error) {
	parsed, err := parser.parse(body)
	if err != nil {
		return nil, err
	}
	return parsed.sections()
}

func (parsed parsedBody) sections() (map[string]string, error) {
	byKey := map[string]string{}
	for _, canonical := range canonicalSections() {
		if parsed.tmpl.Lookup(canonical.Key) == nil {
			continue
		}
		var buf bytes.Buffer
		if execErr := parsed.tmpl.ExecuteTemplate(&buf, canonical.Key, nil); execErr != nil {
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
	if execErr := parsed.tmpl.Execute(&preamble, nil); execErr != nil {
		return "", fmt.Errorf("render preamble: %w", execErr)
	}
	sections, err := parsed.sections()
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
	for _, assoc := range parsed.tmpl.Templates() {
		if name := assoc.Name(); name != "body" && !canonical[name] {
			names = append(names, name)
		}
	}
	return names, nil
}
