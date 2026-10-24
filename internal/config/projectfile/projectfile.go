package projectfile

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile/strictyaml"

const (
	Name             = ".pulsarules.yaml"
	SupportedVersion = 1
)

type Rules struct {
	AnalyzerIDs      []string
	ReservedIDs      []string
	Params           map[string][]string
	DelegatedFormats []string
	AnyAnalyzer      bool
}

type File struct {
	Path    string   `yaml:"-"`
	Sources []string `yaml:"-"`
	// PersonalOff lists analyzers the user layer switches off everywhere an override does not
	// enable them again; set by the loader, never read from a file.
	PersonalOff      []string                 `yaml:"-"`
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
	Analyzers map[string]AnalyzerEntry `yaml:"analyzers"`
}

type DelegatedEntry struct {
	ID     string   `yaml:"id"`
	Dir    string   `yaml:"dir"`
	Format string   `yaml:"format"`
	Rule   string   `yaml:"rule"`
	Run    []string `yaml:"run"`
}

func Parse(path string, content []byte, rules Rules) (*File, error) {
	doc, err := strictyaml.Decode[File](path, content)
	if err != nil {
		return nil, err
	}
	if doc.Value != nil {
		doc.Value.Path = path
		doc.Value.check(rules, doc.Root, doc.Problems)
	}
	if err = doc.Problems.Err(); err != nil {
		return nil, err
	}
	return doc.Value, nil
}
