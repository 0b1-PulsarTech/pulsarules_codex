package strictyaml

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
)

const (
	MaxFileBytes     = 256 << 10
	maxReported      = 20
	maxReportedBytes = 3072
)

type Document struct {
	file     *projectfile.File
	problems *Problems
}

func Decode(path string, content []byte) (Document, error) {
	if len(content) > MaxFileBytes {
		return Document{}, fmt.Errorf("%s: config is larger than %d KiB", path, MaxFileBytes>>10)
	}
	if err := rejectBadText(path, content); err != nil {
		return Document{}, err
	}
	root, err := readDocument(path, content)
	if err != nil {
		return Document{}, err
	}
	value, top := root.Content[0], newRoot()
	parsed := Document{problems: &Problems{path: path}}
	walkSchema(parsed.problems, value, top)
	if value = deref(value); value.Kind != yaml.MappingNode && !isNull(value) {
		return parsed, nil
	}
	file := &projectfile.File{Path: path}
	err = root.Decode(file)
	if err == nil {
		parsed.file = file
		return parsed, nil
	}
	if parsed.problems.total == 0 {
		return Document{}, leftover(path, err)
	}
	// The walk already reported the same mismatches in terms of the file, not Go types.
	if _, isType := errors.AsType[*yaml.TypeError](err); isType {
		parsed.file = file
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
