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
				LeadingEmojis: []string{"wrench"},
				Type:          "feat",
				Description:   "Add new extractor",
			},
		},
		{
			name:  "feat with scope",
			input: ":wrench: feat(goscan): Detect variable shadowing",
			want: Message{
				LeadingEmojis: []string{"wrench"},
				Type:          "feat",
				Scope:         "goscan",
				Description:   "Detect variable shadowing",
			},
		},
		{
			name:  "breaking change",
			input: ":bug: fix(api)!: Remove deprecated endpoint",
			want: Message{
				LeadingEmojis: []string{"bug"},
				Type:          "fix",
				Scope:         "api",
				Breaking:      true,
				Description:   "Remove deprecated endpoint",
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
				LeadingEmojis: []string{"whale", "rocket"},
				Type:          "build",
				Description:   "Improve image size",
			},
		},
		{
			name:  "ghost initial commit",
			input: ":ghost: Initial Commit",
			want: Message{
				LeadingEmojis: []string{"ghost"},
				Kind:          Initial,
				Description:   "Initial Commit",
			},
		},
		{
			name:  "ghost initial commit lowercase",
			input: ":ghost: Initial commit",
			want: Message{
				LeadingEmojis: []string{"ghost"},
				Kind:          Initial,
				Description:   "Initial commit",
			},
		},
		{
			name:  "volcano merge",
			input: ":volcano: Merge branch 'feature-x'",
			want: Message{
				LeadingEmojis: []string{"volcano"},
				Kind:          Merge,
				Description:   "Merge branch 'feature-x'",
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
				LeadingEmojis: []string{"wrench"},
				Type:          "feat",
				Description:   "[wip] Create initial version",
				IsWIP:         true,
			},
		},
		{
			name:  "wip prefix uppercase",
			input: ":package: feat: [WIP] Create gotalaria module",
			want: Message{
				LeadingEmojis: []string{"package"},
				Type:          "feat",
				Description:   "[WIP] Create gotalaria module",
				IsWIP:         true,
			},
		},
		{
			name: "body with footers",
			input: ":bug: fix: Prevent crash on nil input\n\n" +
				"The handler did not check for nil.\n\nCloses: #42",
			want: Message{
				LeadingEmojis: []string{"bug"},
				Type:          "fix",
				Description:   "Prevent crash on nil input",
				Body:          "The handler did not check for nil.",
				Footers:       []Footer{{Key: "Closes", Value: "#42"}},
			},
		},
		{
			name:  "co-author trailer",
			input: ":wrench: feat: Add thing\n\nCo-Authored-By: Claude <noreply@anthropic.com>",
			want: Message{
				LeadingEmojis: []string{"wrench"},
				Type:          "feat",
				Description:   "Add thing",
				Body:          "",
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
				LeadingEmojis: []string{"memo"},
				Type:          "docs",
			},
		},
		{
			name:  "scope with underscore",
			input: ":tea: test(support_underscore): The Gate",
			want: Message{
				LeadingEmojis: []string{"tea"},
				Type:          "test",
				Scope:         "support_underscore",
				Description:   "The Gate",
			},
		},
		{
			name:  "four leading emojis over the max",
			input: ":sparkles: :bug: :wrench: :zap: feat: Add a thing",
			want: Message{
				LeadingEmojis: []string{"sparkles", "bug", "wrench", "zap"},
				Type:          "feat",
				Description:   "Add a thing",
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

	if !emojisEqual(got.LeadingEmojis, want.LeadingEmojis) {
		t.Errorf("LeadingEmojis = %v, want %v", got.LeadingEmojis, want.LeadingEmojis)
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

	if got.Kind != want.Kind {
		t.Errorf("Kind = %v, want %v", got.Kind, want.Kind)
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

func TestToolTrailer(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		msg  Message
		want string
	}{
		{
			"co-authored-by footer",
			Message{Footers: []Footer{{Key: "Co-Authored-By", Value: "x"}}},
			"Co-Authored-By",
		},
		{
			"claude-session footer",
			Message{Footers: []Footer{{Key: "Claude-Session", Value: "url"}}},
			"Claude-Session",
		},
		{
			"attribution marker in raw",
			Message{Raw: ":wrench: feat: x\n\nGenerated with Claude Code."},
			"generated with claude",
		},
		{
			"legit footer only",
			Message{Footers: []Footer{{Key: "Closes", Value: "#1"}}},
			"",
		},
		{"empty message", Message{}, ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := testCase.msg.ToolTrailer(); got != testCase.want {
				t.Fatalf("ToolTrailer = %q, want %q", got, testCase.want)
			}
		})
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

func TestParseKind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		input         string
		wantKind      Kind
		wantEmojis    []string
		wantCanonical bool
	}{
		{"regular", ":wrench: feat: Add", Regular, []string{"wrench"}, false},
		{"no emoji", "feat: Add", Regular, nil, false},
		{"canonical initial", ":ghost: Initial Commit", Initial, []string{"ghost"}, true},
		{"lowercase initial", ":ghost: Initial commit", Initial, []string{"ghost"}, true},
		{"wrong emoji initial", ":smile: Initial Commit", Initial, []string{"smile"}, false},
		{
			"extra emoji initial", ":ghost::smile: Initial Commit", Initial,
			[]string{"ghost", "smile"},
			false,
		},
		{"bare initial", "Initial Commit", Initial, nil, false},
		{"canonical merge", ":volcano: Merge branch x", Merge, []string{"volcano"}, true},
		{"volcano then more", ":volcano::tea: Merge x", Merge, []string{"volcano", "tea"}, true},
		{"wrong emoji merge", ":smile: Merge branch x", Merge, []string{"smile"}, false},
		{"bare merge", "Merge branch x", Merge, nil, false},
		{"merge needs trailing space", ":volcano: Merged", Regular, []string{"volcano"}, false},
		{"ghost on regular", ":ghost: feat: Ghost", Regular, []string{"ghost"}, false},
		{"leading whitespace", "  :tea: test: Add", Regular, []string{"tea"}, false},
		{"unterminated shortcode", ":wrench feat: Add", Regular, nil, false},
		{"empty", "", Regular, nil, false},
		{"four emojis", ":a: :b: :c: :d: feat: X", Regular, []string{"a", "b", "c", "d"}, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := Parse(testCase.input)
			if got.Kind != testCase.wantKind {
				t.Errorf("Kind = %v, want %v", got.Kind, testCase.wantKind)
			}
			if !emojisEqual(got.LeadingEmojis, testCase.wantEmojis) {
				t.Errorf("LeadingEmojis = %v, want %v", got.LeadingEmojis, testCase.wantEmojis)
			}
			if got.IsCanonicalSpecial() != testCase.wantCanonical {
				t.Errorf("IsCanonicalSpecial = %v, want %v",
					got.IsCanonicalSpecial(), testCase.wantCanonical)
			}
		})
	}
}

func TestIsScopeChar(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		char rune
		want bool
	}{
		{'a', true},
		{'Z', true},
		{'7', true},
		{'-', true},
		{'_', true},
		{' ', false},
		{'/', false},
		{'\u00e9', false},
	}
	for _, testCase := range testCases {
		t.Run(string(testCase.char), func(t *testing.T) {
			t.Parallel()
			if got := IsScopeChar(testCase.char); got != testCase.want {
				t.Errorf("IsScopeChar(%q) = %v, want %v", testCase.char, got, testCase.want)
			}
		})
	}
}
