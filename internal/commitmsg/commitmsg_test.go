package commitmsg

import (
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	t.Parallel()

	for _, testCase := range parseTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(testCase.input)
			checkCommit(t, got, testCase.want)
		})
	}
}

type parseTestCase struct {
	name  string
	input string
	want  Message
}

func parseTestCases() []parseTestCase {
	return slices.Concat(
		parseCasesBasic(), parseCasesMerge(), parseCasesExtended(), parseCasesFooters(),
	)
}

func parseCasesBasic() []parseTestCase {
	return []parseTestCase{
		{
			name:  "simple feat",
			input: ":wrench: feat: Add new extractor",
			want: Message{
				Emojis:      []string{"wrench"},
				Type:        "feat",
				Description: "Add new extractor",
			},
		},
		{
			name:  "feat with scope",
			input: ":wrench: feat(goscan): Detect variable shadowing",
			want: Message{
				Emojis:      []string{"wrench"},
				Type:        "feat",
				Scope:       "goscan",
				Description: "Detect variable shadowing",
			},
		},
		{
			name:  "breaking change",
			input: ":bug: fix(api)!: Remove deprecated endpoint",
			want: Message{
				Emojis:      []string{"bug"},
				Type:        "fix",
				Scope:       "api",
				Breaking:    true,
				Description: "Remove deprecated endpoint",
			},
		},
		{
			name:  "no emoji",
			input: "feat: Add something",
			want: Message{
				Type:        "feat",
				Description: "Add something",
			},
		},
	}
}

func parseCasesMerge() []parseTestCase {
	return []parseTestCase{
		{
			name:  "two emojis",
			input: ":whale: :rocket: build: Improve image size",
			want: Message{
				Emojis:      []string{"whale", "rocket"},
				Type:        "build",
				Description: "Improve image size",
			},
		},
		{
			name:  "ghost initial commit",
			input: ":ghost: Initial Commit",
			want: Message{
				Emojis:      []string{"ghost"},
				IsInitial:   true,
				Description: "Initial Commit",
			},
		},
		{
			name:  "ghost initial commit lowercase",
			input: ":ghost: Initial commit",
			want: Message{
				Emojis:      []string{"ghost"},
				IsInitial:   true,
				Description: "Initial commit",
			},
		},
		{
			name:  "volcano merge",
			input: ":volcano: Merge branch 'feature-x'",
			want: Message{
				Emojis:      []string{"volcano"},
				IsMerge:     true,
				Description: "Merge branch 'feature-x'",
			},
		},
	}
}

func parseCasesExtended() []parseTestCase {
	return []parseTestCase{
		{
			name:  "wip prefix lowercase",
			input: ":wrench: feat: [wip] Create initial version",
			want: Message{
				Emojis:      []string{"wrench"},
				Type:        "feat",
				Description: "[wip] Create initial version",
				IsWIP:       true,
			},
		},
		{
			name:  "wip prefix uppercase",
			input: ":package: feat: [WIP] Create gotalaria module",
			want: Message{
				Emojis:      []string{"package"},
				Type:        "feat",
				Description: "[WIP] Create gotalaria module",
				IsWIP:       true,
			},
		},
		{
			name: "body with footers",
			input: ":bug: fix: Prevent crash on nil input\n\n" +
				"The handler did not check for nil.\n\nCloses: #42",
			want: Message{
				Emojis:      []string{"bug"},
				Type:        "fix",
				Description: "Prevent crash on nil input",
				Body:        "The handler did not check for nil.",
				Footers:     []Footer{{Key: "Closes", Value: "#42"}},
			},
		},
		{
			name:  "co-author trailer",
			input: ":wrench: feat: Add thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
			want: Message{
				Emojis:      []string{"wrench"},
				Type:        "feat",
				Description: "Add thing",
				Body:        "",
				Footers: []Footer{
					{Key: "Co-Authored-By", Value: "Claude <noreply@anthropic.com>"},
				},
			},
		},
	}
}

func parseCasesFooters() []parseTestCase {
	return []parseTestCase{
		{
			name:  "empty input",
			input: "",
			want:  Message{Raw: ""},
		},
		{
			name:  "type only no description",
			input: ":memo: docs:",
			want: Message{
				Emojis: []string{"memo"},
				Type:   "docs",
			},
		},
		{
			name:  "scope with underscore",
			input: ":tea: test(support_underscore): The Gate",
			want: Message{
				Emojis:      []string{"tea"},
				Type:        "test",
				Scope:       "support_underscore",
				Description: "The Gate",
			},
		},
		{
			name:  "four leading emojis over the max",
			input: ":sparkles: :bug: :wrench: :zap: feat: Add a thing",
			want: Message{
				Emojis:      []string{"sparkles", "bug", "wrench", "zap"},
				Type:        "feat",
				Description: "Add a thing",
			},
		},
	}
}

func checkCommit(t *testing.T, got, want Message) {
	t.Helper()

	checkHeader(t, got, want)
	checkBody(t, got, want)
}

func checkHeader(t *testing.T, got, want Message) {
	t.Helper()

	if !emojisEqual(got.Emojis, want.Emojis) {
		t.Errorf("Emojis = %v, want %v", got.Emojis, want.Emojis)
	}
	if got.Type != want.Type {
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
	if got.Scope != want.Scope {
		t.Errorf("Scope = %q, want %q", got.Scope, want.Scope)
	}
	if got.Breaking != want.Breaking {
		t.Errorf("Breaking = %v, want %v", got.Breaking, want.Breaking)
	}
	if got.Description != want.Description {
		t.Errorf("Description = %q, want %q", got.Description, want.Description)
	}
}

func checkBody(t *testing.T, got, want Message) {
	t.Helper()

	if got.IsInitial != want.IsInitial {
		t.Errorf("IsInitial = %v, want %v", got.IsInitial, want.IsInitial)
	}
	if got.IsMerge != want.IsMerge {
		t.Errorf("IsMerge = %v, want %v", got.IsMerge, want.IsMerge)
	}
	if got.IsWIP != want.IsWIP {
		t.Errorf("IsWIP = %v, want %v", got.IsWIP, want.IsWIP)
	}
	if got.Body != want.Body {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
	if !footersEqual(got.Footers, want.Footers) {
		t.Errorf("Footers = %v, want %v", got.Footers, want.Footers)
	}
}

func emojisEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}

func footersEqual(left, right []Footer) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx].Key != right[idx].Key || left[idx].Value != right[idx].Value {
			return false
		}
	}
	return true
}
