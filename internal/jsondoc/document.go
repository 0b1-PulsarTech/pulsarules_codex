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
	path    string
	scope   string
	exists  bool
	removal bool
	fields  map[string]json.RawMessage
}

func Load(path string) (*Document, error) {
	doc, raw, err := read(path)
	if err != nil {
		return nil, err
	}
	// A file about to be written treats a blank one as an empty document, and a parse
	// failure carries only its cause so the caller can phrase it.
	if len(bytes.TrimSpace(raw)) == 0 {
		return doc, nil
	}
	return doc.decode(raw)
}

func LoadForRemoval(path string) (*Document, error) {
	doc, raw, err := read(path)
	if err != nil {
		return nil, err
	}
	// A file about to be stripped has no blank case: a blank one does not parse, and
	// every parse failure reads as the warning uninstall prints.
	doc.removal = true
	if raw == nil {
		return doc, nil
	}
	return doc.decode(raw)
}

func read(path string) (*Document, []byte, error) {
	raw, err := disk.ReadOptional(path)
	if err != nil {
		return nil, nil, err
	}
	doc := &Document{path: path, exists: raw != nil, fields: map[string]json.RawMessage{}}
	return doc, raw, nil
}

func (doc *Document) decode(raw []byte) (*Document, error) {
	if err := json.Unmarshal(raw, &doc.fields); err != nil {
		return nil, doc.parseFailure("", err)
	}
	if doc.fields == nil {
		// A bare `null` decodes to a nil map, which cannot take a section.
		doc.fields = map[string]json.RawMessage{}
	}
	return doc, nil
}

func (doc *Document) parseFailure(label string, cause error) error {
	if !doc.removal {
		return &parseError{cause: cause}
	}
	where := fmt.Sprintf("%q", doc.path)
	if label != "" {
		where += " " + label
	}
	return fmt.Errorf("%w: %s: %w", ErrUnparseable, where, cause)
}

func (doc *Document) Raw(key string) (json.RawMessage, bool) {
	raw, ok := doc.fields[key]
	return raw, ok
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
		return zero, doc.parseFailure(doc.label(key), err)
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
	child := &Document{path: doc.path, scope: doc.label(key), removal: doc.removal, fields: fields}
	return child, nil
}

func (doc *Document) PutChild(key string, child *Document) error {
	if child.Len() == 0 {
		doc.Delete(key)
		return nil
	}
	return SetSection(doc, key, child.fields)
}
