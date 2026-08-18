package knowledge

// Rule is a mandatory, enforceable engineering rule.
type Rule struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Linters      []string `yaml:"linters,omitempty"`
	// Analyzers lists the ids of this repo's own analyzers (registered under
	// internal/analysis) that enforce this rule, as opposed to Linters, which
	// names external golangci-lint linters. A rule with neither carries no
	// machine-checked enforcement.
	Analyzers  []string `yaml:"analyzers,omitempty"`
	References []string `yaml:"references,omitempty"`
}

// Pattern is an implementation recipe that shows how to apply the rules.
type Pattern struct {
	ID           string   `yaml:"id"`
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Tags         []string `yaml:"tags,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
	Composes     []string `yaml:"composes,omitempty"`
	References   []string `yaml:"references,omitempty"`
}

// Reference is an external canonical source a rule or pattern can cite; the
// renderer surfaces cited references under "## References" in a generated skill.
type Reference struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
	Note  string `yaml:"note,omitempty"`
}

type referencesFile struct {
	References []Reference `yaml:"references"`
}

// Workflow is an ordered sequence that composes rules and patterns for a task.
type Workflow struct {
	ID               string   `yaml:"id"`
	Name             string   `yaml:"name"`
	Description      string   `yaml:"description"`
	Tags             []string `yaml:"tags,omitempty"`
	Steps            []string `yaml:"steps,omitempty"`
	ComposesRules    []string `yaml:"composes_rules,omitempty"`
	ComposesPatterns []string `yaml:"composes_patterns,omitempty"`
}

// Skill is a generated capability that composes rules, patterns, and workflows.
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
	// NoMerge renders composed rules/patterns per source instead of merging
	// them section-by-section. The zero value keeps the merging default, so
	// the field states the departure rather than the norm.
	NoMerge bool `yaml:"no_merge,omitempty"`
}

func (s Skill) Merged() bool { return !s.NoMerge }

type skillsFile struct {
	Skills []Skill `yaml:"skills"`
}

// SkillOverride is a profile's replacement of a skill's composition lists. An
// empty list leaves that dimension unchanged.
type SkillOverride struct {
	ComposeRules    []string `yaml:"compose_rules,omitempty"`
	ComposePatterns []string `yaml:"compose_patterns,omitempty"`
}

// Profile is an install-time customization (e.g. a code-layout variant) that
// overrides what selected skills compose, so one knowledge base serves several
// project shapes.
type Profile struct {
	ID          string                   `yaml:"id"`
	Axis        string                   `yaml:"axis"`
	Description string                   `yaml:"description"`
	Overrides   map[string]SkillOverride `yaml:"overrides,omitempty"`
}

type profilesFile struct {
	Profiles []Profile `yaml:"profiles"`
}

// RouterBaselineEntry is one always-load or conditionally-loaded baseline skill
// with the short reason the router prints next to it.
type RouterBaselineEntry struct {
	Skill string `yaml:"skill"`
	Note  string `yaml:"note"`
}

// RouterBaseline groups the router's always-load baseline skills, the
// commit-last skills, and the conditionally-loaded ones.
type RouterBaseline struct {
	Always      []RouterBaselineEntry `yaml:"always"`
	CommitLast  []RouterBaselineEntry `yaml:"commit_last"`
	Conditional []RouterBaselineEntry `yaml:"conditional"`
}

// RouterDispatchRow maps a task signal to the skills it should load, with an
// optional parenthetical note for the conditional nuance.
type RouterDispatchRow struct {
	Signal string   `yaml:"signal"`
	Skills []string `yaml:"skills"`
	Note   string   `yaml:"note,omitempty"`
}

// RouterOrderStep is one layer of the composition order: the skills applied at
// that layer and the reason they come there.
type RouterOrderStep struct {
	Skills []string `yaml:"skills"`
	Note   string   `yaml:"note"`
}

// RouterSpec is the data behind the project-router skill's baseline, dispatch
// table, and composition order, so the rendered router can be filtered to only
// the skills actually being installed.
type RouterSpec struct {
	Baseline RouterBaseline      `yaml:"baseline"`
	Dispatch []RouterDispatchRow `yaml:"dispatch"`
	Order    []RouterOrderStep   `yaml:"order"`
}

type routerFile struct {
	Router RouterSpec `yaml:"router"`
}
