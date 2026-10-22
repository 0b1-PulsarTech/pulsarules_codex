package projectfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"

	"gopkg.in/yaml.v3"
)

type Document[Target any] struct {
	Value    *Target
	Problems *Problems
	Root     *Field
}

func Decode[Target any](path string, content []byte) (Document[Target], error) {
	if len(content) > MaxFileBytes {
		return Document[Target]{}, fmt.Errorf(
			"%s: config is larger than %d KiB", path, MaxFileBytes>>10,
		)
	}
	if err := rejectBadText(path, content); err != nil {
		return Document[Target]{}, err
	}
	root, err := readDocument(path, content)
	if err != nil {
		return Document[Target]{}, err
	}
	if err = checkExpansion(path, root); err != nil {
		return Document[Target]{}, err
	}
	value, top := root.Content[0], Root()
	parsed := Document[Target]{Problems: NewProblems(path), Root: top}
	walkSchema(parsed.Problems, value, top, reflect.TypeFor[Target]())
	if value = deref(value); value.Kind != yaml.MappingNode && !isNull(value) {
		return parsed, nil
	}
	target := new(Target)
	err = root.Decode(target)
	if err == nil {
		parsed.Value = target
		return parsed, nil
	}
	if parsed.Problems.total == 0 {
		return Document[Target]{}, leftover[Target](path, root, err)
	}
	// The walk already reported the same mismatches in terms of the file, not Go types.
	if _, isType := errors.AsType[*yaml.TypeError](err); isType {
		parsed.Value = target
	}
	return parsed, nil
}

func readDocument(path string, content []byte) (*yaml.Node, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	var root yaml.Node
	err := decoder.Decode(&root)
	if errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: version: required", path)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %s", path, clipParseText(plainText(err)))
	}
	for {
		var extra yaml.Node
		err = decoder.Decode(&extra)
		if errors.Is(err, io.EOF) {
			return &root, nil
		}
		if err != nil || !emptyDocument(&extra) {
			return nil, fmt.Errorf("%s: only one YAML document is allowed", path)
		}
	}
}

func deref(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

func emptyDocument(node *yaml.Node) bool {
	return node.Kind == yaml.DocumentNode && len(node.Content) == 1 &&
		isNull(node.Content[0]) && node.Content[0].Value == ""
}
