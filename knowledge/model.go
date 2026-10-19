package knowledge

import "slices"

type Rule struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Linters      []string `yaml:"linters,omitempty"`
	Analyzers    []string `yaml:"analyzers,omitempty"`
	References   []string `yaml:"references,omitempty"`
}

func (rule Rule) FitsLanguages(known map[string]bool, languages []string) bool {
	if len(languages) == 0 {
		return true
	}
	tagged := false
	for _, tag := range rule.Tags {
		if !known[tag] {
			continue
		}
		tagged = true
		if slices.Contains(languages, tag) {
			return true
		}
	}
	return !tagged
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

type Reference struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
	Note  string `yaml:"note,omitempty"`
}

type referencesFile struct {
	References []Reference `yaml:"references"`
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
	Languages        []string `yaml:"languages,omitempty"`
	ComposeRules     []string `yaml:"compose_rules,omitempty"`
	ComposePatterns  []string `yaml:"compose_patterns,omitempty"`
	ComposeWorkflows []string `yaml:"compose_workflows,omitempty"`
	ComposeSkills    []string `yaml:"compose_skills,omitempty"`
	NoMerge          bool     `yaml:"no_merge,omitempty"`
}

func (skill Skill) Merged() bool { return !skill.NoMerge }

type skillsFile struct {
	Skills []Skill `yaml:"skills"`
}

type SkillOverride struct {
	ComposeRules    []string `yaml:"compose_rules,omitempty"`
	ComposePatterns []string `yaml:"compose_patterns,omitempty"`
}

func (override SkillOverride) Apply(skill *Skill) {
	if override.ComposeRules != nil {
		skill.ComposeRules = override.ComposeRules
	}
	if override.ComposePatterns != nil {
		skill.ComposePatterns = override.ComposePatterns
	}
}

type Profile struct {
	ID          string                   `yaml:"id"`
	Axis        string                   `yaml:"axis"`
	Description string                   `yaml:"description"`
	Overrides   map[string]SkillOverride `yaml:"overrides,omitempty"`
}

type profilesFile struct {
	Profiles []Profile `yaml:"profiles"`
}

type RouterBaselineEntry struct {
	Skill string `yaml:"skill"`
	Note  string `yaml:"note"`
}

type RouterBaseline struct {
	Always      []RouterBaselineEntry `yaml:"always"`
	CommitLast  []RouterBaselineEntry `yaml:"commit_last"`
	Conditional []RouterBaselineEntry `yaml:"conditional"`
}

type RouterDispatchRow struct {
	Signal string   `yaml:"signal"`
	Skills []string `yaml:"skills"`
	Note   string   `yaml:"note,omitempty"`
}

type RouterOrderStep struct {
	Skills []string `yaml:"skills"`
	Note   string   `yaml:"note"`
}

type RouterSpec struct {
	Baseline RouterBaseline      `yaml:"baseline"`
	Dispatch []RouterDispatchRow `yaml:"dispatch"`
	Order    []RouterOrderStep   `yaml:"order"`
}

type routerFile struct {
	Router RouterSpec `yaml:"router"`
}
