package strictyaml

const fixtureName = ".pulsarules.yaml"

type fixtureFile struct {
	Version          int                        `yaml:"version"`
	Preset           string                     `yaml:"preset"`
	IncludeGenerated *bool                      `yaml:"include_generated"`
	SkipDirs         []string                   `yaml:"skip_dirs"`
	Tests            []string                   `yaml:"tests"`
	Analyzers        map[string]fixtureAnalyzer `yaml:"analyzers"`
	Overrides        []fixtureOverride          `yaml:"overrides"`
	Delegated        []fixtureDelegated         `yaml:"delegated"`
}

type fixtureAnalyzer struct {
	Enabled  *bool          `yaml:"enabled"`
	Severity string         `yaml:"severity"`
	Params   map[string]any `yaml:"params"`
}

type fixtureOverride struct {
	Files     []string                   `yaml:"files"`
	Analyzers map[string]fixtureAnalyzer `yaml:"analyzers"`
}

type fixtureDelegated struct {
	ID     string   `yaml:"id"`
	Dir    string   `yaml:"dir"`
	Format string   `yaml:"format"`
	Rule   string   `yaml:"rule"`
	Run    []string `yaml:"run"`
}

type fixtureCase struct {
	name    string
	content string
	want    []string
}

func decodeFixture(content []byte) (*fixtureFile, error) {
	doc, err := Decode[fixtureFile](fixtureName, content)
	if err != nil {
		return nil, err
	}
	if err = doc.Problems.Err(); err != nil {
		return nil, err
	}
	return doc.Value, nil
}
