package commitmsg

type Message struct {
	Emojis      []string
	Type        string
	Scope       string
	Breaking    bool
	Description string
	Body        string
	Footers     []Footer
	IsMerge     bool
	IsInitial   bool
	IsWIP       bool
	Raw         string
}

type Footer struct {
	Key   string
	Value string
}

func AllowedTypes() []string {
	return []string{
		"feat", "fix", "refactor", "chore", "docs",
		"test", "perf", "build", "ci", "style", "revert",
	}
}
