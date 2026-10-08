package config

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"
)

func mergeTargets(node *yaml.Node) ([]*yaml.Node, bool) {
	// yaml.v3 accepts an alias only when it names a mapping, never a sequence.
	if node.Kind == yaml.AliasNode {
		node = deref(node)
		return []*yaml.Node{node}, node.Kind == yaml.MappingNode
	}
	switch node.Kind {
	case yaml.MappingNode:
		return []*yaml.Node{node}, true
	case yaml.SequenceNode:
		targets := make([]*yaml.Node, 0, len(node.Content))
		for _, item := range node.Content {
			if item = deref(item); item.Kind != yaml.MappingNode {
				return nil, false
			}
			targets = append(targets, item)
		}
		return targets, true
	}
	return nil, false
}

func isMerge(key *yaml.Node) bool {
	return key.Kind == yaml.ScalarNode && key.Value == "<<" &&
		(key.Tag == "" || key.Tag == "!" || key.Tag == "!!merge")
}

func isNull(node *yaml.Node) bool {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!null" {
		return false
	}
	switch node.Value {
	case "", "~", "null", "Null", "NULL":
		return true
	}
	return false
}

func taggedNull(node *yaml.Node) bool {
	return node.Kind != yaml.ScalarNode && node.Tag == "!!null"
}

func undecodableKey(key *yaml.Node) bool {
	return key.Kind == yaml.ScalarNode && !isNull(key) && key.Decode(new(string)) != nil
}

func shape(typ reflect.Type) (yaml.Kind, string) {
	switch typ.Kind() {
	case reflect.Struct, reflect.Map:
		return yaml.MappingNode, "a mapping"
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.String {
			return yaml.SequenceNode, "a list of strings"
		}
		return yaml.SequenceNode, "a list"
	case reflect.Bool:
		return yaml.ScalarNode, "a boolean"
	case reflect.Int:
		return yaml.ScalarNode, "an integer"
	}
	return yaml.ScalarNode, "a string"
}

func yamlFields(typ reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type, typ.NumField())
	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("yaml"), ",")
		if name == "" || name == "-" {
			continue
		}
		fields[name] = field.Type
	}
	return fields
}

func keyName(key *yaml.Node) string {
	key = deref(key)
	if isNull(key) {
		return "<null key>"
	}
	if key.Kind != yaml.ScalarNode {
		return "<complex key>"
	}
	return key.Value
}

func dropShadowed(entries []pair) []pair {
	// An explicit key replaces a merged one, and an earlier merge source replaces a later one.
	taken := map[string]bool{}
	for _, entry := range entries {
		if key := deref(entry.key); !entry.inherited && key.Kind == yaml.ScalarNode {
			taken[key.Value] = true
		}
	}
	kept := entries[:0]
	for _, entry := range entries {
		key := deref(entry.key)
		if entry.inherited && key.Kind == yaml.ScalarNode {
			if taken[key.Value] {
				continue
			}
			taken[key.Value] = true
		}
		kept = append(kept, entry)
	}
	return kept
}

func rejectBadText(path string, content []byte) error {
	if err := rejectNonUTF8(path, content); err != nil {
		return err
	}
	return rejectDirectives(path, content)
}

func rejectNonUTF8(path string, content []byte) error {
	// A UTF-16 byte order mark starts with 0xff or 0xfe, which UTF-8 never contains.
	if bytes.IndexByte(content, 0) >= 0 || !utf8.Valid(content) {
		return fmt.Errorf("%s: the file must be UTF-8", path)
	}
	return nil
}

func rejectDirectives(path string, content []byte) error {
	body := bytes.TrimPrefix(content, []byte("\xef\xbb\xbf"))
	line, isStart := 1, true
	for idx := 0; idx < len(body); idx++ {
		if isStart && body[idx] == '%' {
			return fmt.Errorf("%s: line %d: YAML directives are not allowed", path, line)
		}
		width := breakWidth(body[idx:])
		isStart = width > 0
		if isStart {
			line++
			idx += width - 1
		}
	}
	return nil
}

func breakWidth(rest []byte) int {
	// Every break yaml.v3 recognises: LF, CR, CRLF, NEL, LS and PS.
	for _, text := range [...]string{"\r\n", "\n", "\r", "\u0085", "\u2028", "\u2029"} {
		if bytes.HasPrefix(rest, []byte(text)) {
			return len(text)
		}
	}
	return 0
}
