package evals

// Kind marks whether an assertion is graded automatically by Grade or must be
// judged by a human/LLM reading the produced artifact. Every assertion in an
// eval file carries one, so validate.Validate can reject an assertion that
// declares neither.
type Kind string

const (
	// KindMachine assertions carry a Check that Grade evaluates against the artifact text.
	KindMachine Kind = "machine"
	// KindJudge assertions have no Check; Grade reports them NeedsJudge for a
	// human or LLM reader to score against the artifact.
	KindJudge Kind = "judge"
)

// CheckType names the machine-checkable rule a Check applies to an artifact.
type CheckType string

const (
	// CheckContains passes when the artifact contains Check.Pattern verbatim.
	CheckContains CheckType = "contains"
	// CheckNotContains passes when the artifact does NOT contain Check.Pattern verbatim.
	CheckNotContains CheckType = "not_contains"
	// CheckRegexMatch passes when Check.Pattern, compiled as a Go regexp, matches the artifact.
	CheckRegexMatch CheckType = "regex_match"
	// CheckRegexAbsent passes when Check.Pattern, compiled as a Go regexp, does NOT match the artifact.
	CheckRegexAbsent CheckType = "regex_absent"
)

// Check is the machine-checkable rule a KindMachine assertion evaluates
// against a produced artifact.
type Check struct {
	Type    CheckType `json:"type"`
	Pattern string    `json:"pattern"`
}

// Assertion is one plain-English claim about what a correct artifact does or
// avoids. A KindMachine assertion carries a Check Grade can run directly; a
// KindJudge assertion carries none and is left for a human or LLM reader.
type Assertion struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Kind  Kind   `json:"kind"`
	Check *Check `json:"check,omitempty"`
}

// Scenario is one realistic task used to measure a skill's effect: a prompt
// an operator gives the model, the specific mistake (Trap) the skill exists
// to prevent, and the assertions a produced artifact is graded against.
type Scenario struct {
	Skill       string      `json:"skill"`
	ID          string      `json:"id"`
	Description string      `json:"description,omitempty"`
	Prompt      string      `json:"prompt"`
	Trap        string      `json:"trap"`
	Assertions  []Assertion `json:"assertions"`
}

// scenarioFile is the on-disk shape of one data/<skill>.json file: every
// scenario in the file targets the same skill, named once at the top instead
// of on every scenario.
type scenarioFile struct {
	Skill     string     `json:"skill"`
	Version   string     `json:"version"`
	Scenarios []Scenario `json:"scenarios"`
}
