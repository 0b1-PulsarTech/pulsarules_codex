package projectfile

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

const (
	Name             = ".pulsarules.yaml"
	SupportedVersion = 1
)

const (
	MaxFileBytes      = 256 << 10
	maxScalarBytes    = 1024
	maxDepth          = 64
	maxExpandedNodes  = 100_000
	MaxMapKeys        = 256
	MaxSeqItems       = 1024
	MaxGlobBytes      = 256
	MaxGlobStars      = 4
	MaxTotalGlobs     = 256
	MaxTotalGlobBytes = 4096
	MaxWildcardWork   = 256
	MaxStarTextBytes  = 2048
	maxReported       = 20
	maxReportedBytes  = 3072
	maxClipBytes      = 80
)

type Rules struct {
	AnalyzerIDs      []string
	ReservedIDs      []string
	Params           map[string][]string
	DelegatedFormats []string
	AnyAnalyzer      bool
}

type File struct {
	Path             string                   `yaml:"-"`
	Version          int                      `yaml:"version"`
	Preset           string                   `yaml:"preset"`
	IncludeGenerated *bool                    `yaml:"include_generated"`
	SkipDirs         []string                 `yaml:"skip_dirs"`
	Tests            []string                 `yaml:"tests"`
	Analyzers        map[string]AnalyzerEntry `yaml:"analyzers"`
	Overrides        []OverrideEntry          `yaml:"overrides"`
	Delegated        []DelegatedEntry         `yaml:"delegated"`
}

type AnalyzerEntry struct {
	Enabled  *bool          `yaml:"enabled"`
	Severity string         `yaml:"severity"`
	Params   map[string]any `yaml:"params"`
}

type OverrideEntry struct {
	Files     []string                 `yaml:"files"`
	Analyzers map[string]OverrideRules `yaml:"analyzers"`
}

type OverrideRules struct {
	Enabled  *bool          `yaml:"enabled"`
	Severity string         `yaml:"severity"`
	Params   map[string]any `yaml:"params"`
}

type DelegatedEntry struct {
	ID     string   `yaml:"id"`
	Dir    string   `yaml:"dir"`
	Format string   `yaml:"format"`
	Rule   string   `yaml:"rule"`
	Run    []string `yaml:"run"`
}

func Parse(path string, content []byte, rules Rules) (*File, error) {
	parsed, err := Decode(path, content)
	if err != nil {
		return nil, err
	}
	if parsed.file != nil {
		parsed.file.check(rules, parsed.top, parsed.problems)
	}
	return parsed.result()
}

func (parsed Document) result() (*File, error) {
	if err := parsed.problems.err(); err != nil {
		return nil, err
	}
	return parsed.file, nil
}

type Document struct {
	file     *File
	problems *Problems
	top      *Field
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
	if err = checkExpansion(path, root); err != nil {
		return Document{}, err
	}
	value, top := root.Content[0], newRoot()
	parsed := Document{problems: &Problems{path: path}, top: top}
	walkSchema(parsed.problems, value, top)
	if value = deref(value); value.Kind != yaml.MappingNode && !isNull(value) {
		return parsed, nil
	}
	file := &File{Path: path}
	err = root.Decode(file)
	if err == nil {
		parsed.file = file
		return parsed, nil
	}
	if parsed.problems.total == 0 {
		return Document{}, leftover(path, root, err)
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
