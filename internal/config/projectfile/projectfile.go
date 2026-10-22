package projectfile

const (
	Name             = ".pulsarules.yaml"
	SupportedVersion = 1
)

const (
	MaxFileBytes      = 256 << 10
	MaxScalarBytes    = 1024
	MaxDepth          = 64
	MaxExpandedNodes  = 100_000
	MaxMapKeys        = 256
	MaxSeqItems       = 1024
	MaxGlobBytes      = 256
	MaxGlobStars      = 4
	MaxTotalGlobs     = 256
	MaxTotalGlobBytes = 4096
	MaxWildcardWork   = 256
	MaxStarTextBytes  = 2048
	MaxReported       = 20
	MaxReportedBytes  = 3072
	MaxClipBytes      = 80
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
	Sources          []string                 `yaml:"-"`
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
	doc, err := Decode[File](path, content)
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
