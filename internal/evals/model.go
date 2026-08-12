package evals

type Kind string

const (
	KindMachine Kind = "machine"
	KindJudge   Kind = "judge"
)

type CheckType string

const (
	CheckContains    CheckType = "contains"
	CheckNotContains CheckType = "not_contains"
	CheckRegexMatch  CheckType = "regex_match"
	CheckRegexAbsent CheckType = "regex_absent"
)

type Check struct {
	Type    CheckType `json:"type"`
	Pattern string    `json:"pattern"`
}

type Assertion struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Kind  Kind   `json:"kind"`
	Check *Check `json:"check,omitempty"`
}

type Scenario struct {
	Skill      string      `json:"skill"`
	ID         string      `json:"id"`
	Prompt     string      `json:"prompt"`
	Trap       string      `json:"trap"`
	Assertions []Assertion `json:"assertions"`
}

type scenarioFile struct {
	Skill     string     `json:"skill"`
	Scenarios []Scenario `json:"scenarios"`
}
