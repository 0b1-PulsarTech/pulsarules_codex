package knowledge

type Rule struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Linters      []string `yaml:"linters,omitempty"`
	References   []string `yaml:"references,omitempty"`
}

type Pattern struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Composes     []string `yaml:"composes,omitempty"`
	References   []string `yaml:"references,omitempty"`
}

type Workflow struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Tags             []string `yaml:"tags,omitempty"`
	Steps            []string `yaml:"steps,omitempty"`
	ComposesRules    []string `yaml:"composes_rules,omitempty"`
	ComposesPatterns []string `yaml:"composes_patterns,omitempty"`
}

type Skill struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Triggers         []string `yaml:"triggers,omitempty"`
	AlwaysLoad       bool     `yaml:"always_load"`
	Order            int      `yaml:"order"`
	ComposeRules     []string `yaml:"compose_rules,omitempty"`
	ComposePatterns  []string `yaml:"compose_patterns,omitempty"`
	ComposeWorkflows []string `yaml:"compose_workflows,omitempty"`
	ComposeSkills    []string `yaml:"compose_skills,omitempty"`
}

type skillsFile struct {
	Skills []Skill `yaml:"skills"`
}
