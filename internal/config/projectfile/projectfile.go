package projectfile

const Name = ".pulsarules.yaml"

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
