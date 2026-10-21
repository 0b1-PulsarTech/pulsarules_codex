package jsondoc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/disk"
)

var ErrUnparseable = errors.New("existing config file is not valid JSON")

type Document struct {
	path   string
	scope  string
	exists bool
	fields map[string]json.RawMessage
}

func Load(path string) (*Document, error) {
	raw, err := disk.ReadOptional(path)
	if err != nil {
		return nil, err
	}
	doc := &Document{path: path, exists: raw != nil, fields: map[string]json.RawMessage{}}
	if len(bytes.TrimSpace(raw)) == 0 {
		return doc, nil
	}
	if err = json.Unmarshal(raw, &doc.fields); err != nil {
		return nil, fmt.Errorf("%w: %q: %w", ErrUnparseable, path, err)
	}
	if doc.fields == nil {
		// A bare `null` decodes to a nil map, which cannot take a section.
		doc.fields = map[string]json.RawMessage{}
	}
	return doc, nil
}

func (doc *Document) Exists() bool { return doc.exists }

func (doc *Document) Has(key string) bool {
	_, ok := doc.fields[key]
	return ok
}

func (doc *Document) Len() int { return len(doc.fields) }

func (doc *Document) Delete(key string) { delete(doc.fields, key) }

func (doc *Document) label(key string) string {
	if doc.scope == "" {
		return key
	}
	return doc.scope + "." + key
}

func Section[Value any](doc *Document, key string) (Value, error) {
	var value Value
	raw, ok := doc.fields[key]
	if !ok {
		return value, nil
	}
	if err := json.Unmarshal(raw, &value); err != nil {
		var zero Value
		return zero, fmt.Errorf("%w: %q %s: %w", ErrUnparseable, doc.path, doc.label(key), err)
	}
	return value, nil
}

func SetSection[Value any](doc *Document, key string, value Value) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", doc.label(key), err)
	}
	doc.fields[key] = raw
	return nil
}

func (doc *Document) Child(key string) (*Document, error) {
	fields, err := Section[map[string]json.RawMessage](doc, key)
	if err != nil {
		return nil, err
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return &Document{path: doc.path, scope: doc.label(key), fields: fields}, nil
}

func (doc *Document) PutChild(key string, child *Document) error {
	if child.Len() == 0 {
		doc.Delete(key)
		return nil
	}
	return SetSection(doc, key, child.fields)
}
