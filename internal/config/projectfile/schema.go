package projectfile

import (
	"reflect"

	"gopkg.in/yaml.v3"
)

type pair struct {
	key       *yaml.Node
	value     *yaml.Node
	inherited bool
}

type schemaWalker struct {
	problems *collector
}

func walkSchema(problems *collector, root *yaml.Node, top *fieldPath) {
	walker := schemaWalker{problems: problems}
	if !isNull(deref(root)) {
		walker.walk(root, reflect.TypeFor[File](), top)
	}
}

func (walker *schemaWalker) report(line int, at *fieldPath, format string, args ...any) {
	at.mark()
	walker.problems.add(line, at, format, args...)
}

func (walker *schemaWalker) walk(node *yaml.Node, typ reflect.Type, at *fieldPath) {
	line := node.Line
	node = deref(node)
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() == reflect.Interface {
		if node.Decode(new(any)) != nil {
			walker.report(line, at, "invalid value")
		}
		return
	}
	kind, want := shape(typ)
	if isNull(node) || node.Kind != kind || taggedNull(node) ||
		!walker.walkValue(node, typ, at) {
		walker.report(line, at, "want %s", want)
	}
}

func (walker *schemaWalker) walkValue(node *yaml.Node, typ reflect.Type, at *fieldPath) bool {
	switch typ.Kind() {
	case reflect.Struct:
		walker.walkStruct(node, typ, at)
	case reflect.Map:
		walker.walkMap(node, typ, at)
	case reflect.Slice:
		for idx, item := range node.Content {
			walker.walk(item, typ.Elem(), at.item(idx))
		}
	default:
		if typ.Kind() == reflect.Int && node.ShortTag() != "!!int" {
			return false
		}
		return node.Decode(reflect.New(typ).Interface()) == nil
	}
	return true
}

func (walker *schemaWalker) walkMap(node *yaml.Node, typ reflect.Type, at *fieldPath) {
	for _, entry := range walker.pairs(node, at) {
		field := at.key(keyName(entry.key))
		key := deref(entry.key)
		if key.Kind != yaml.ScalarNode || isNull(key) {
			walker.report(key.Line, field, "want a string key")
			continue
		}
		if undecodableKey(key) {
			walker.report(key.Line, field, "invalid key")
			continue
		}
		walker.walk(entry.value, typ.Elem(), field)
	}
}

func (walker *schemaWalker) walkStruct(node *yaml.Node, typ reflect.Type, at *fieldPath) {
	fields := yamlFields(typ)
	for _, entry := range walker.pairs(node, at) {
		field := at.key(keyName(entry.key))
		if undecodableKey(deref(entry.key)) {
			walker.report(entry.key.Line, field, "invalid key")
			continue
		}
		fieldType, known := fields[deref(entry.key).Value]
		if !known {
			walker.report(entry.key.Line, field, "unknown key")
			continue
		}
		if !isNull(deref(entry.value)) {
			walker.walk(entry.value, fieldType, field)
		}
	}
}

func (walker *schemaWalker) pairs(node *yaml.Node, at *fieldPath) []pair {
	return walker.collect(node, at, map[*yaml.Node]bool{node: true})
}

func (walker *schemaWalker) collect(
	node *yaml.Node, at *fieldPath, merged map[*yaml.Node]bool,
) []pair {
	var entries []pair
	firstLine := map[string]int{}
	for idx := 0; idx+1 < len(node.Content); idx += 2 {
		key, value := node.Content[idx], node.Content[idx+1]
		if isMerge(key) {
			entries = append(entries, walker.inherit(key, value, at, merged)...)
			continue
		}
		if name := deref(key); name.Kind == yaml.ScalarNode {
			if first, dup := firstLine[name.Value]; dup {
				walker.report(
					key.Line, at.key(name.Value), "duplicate key, first at line %d", first,
				)
				continue
			}
			firstLine[name.Value] = key.Line
		}
		entries = append(entries, pair{key: key, value: value})
	}
	return dropShadowed(entries)
}

func (walker *schemaWalker) inherit(
	key, value *yaml.Node, at *fieldPath, merged map[*yaml.Node]bool,
) []pair {
	targets, ok := mergeTargets(value)
	if !ok {
		walker.report(key.Line, at.key("<<"), "want a mapping or a list of mappings")
		return nil
	}
	var entries []pair
	for _, target := range targets {
		if !merged[target] {
			merged[target] = true
			for _, entry := range walker.collect(target, at, merged) {
				entry.inherited = true
				entries = append(entries, entry)
			}
		}
	}
	return entries
}
