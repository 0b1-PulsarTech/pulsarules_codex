package dialect

import (
	"encoding/json"
	"testing"
)

func TestOutput_Take(t *testing.T) {
	t.Parallel()

	type emission struct{ event, context string }
	testCases := []struct {
		name      string
		dialect   string
		emissions []emission
		want      string
	}{
		{"nothing", "", nil, ""},
		{"nothing codex", Codex, nil, ""},
		{
			"claude shape unchanged",
			"",
			[]emission{{"PostToolUse", "hello"}},
			`{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"hello"}}` + "\n",
		},
		{
			"claude stop keeps additionalContext", "",
			[]emission{{"Stop", "found"}},
			`{"hookSpecificOutput":{"hookEventName":"Stop","additionalContext":"found"}}` + "\n",
		},
		{
			"codex stop blocks", Codex,
			[]emission{{"Stop", "found"}},
			`{"decision":"block","reason":"found"}` + "\n",
		},
		{
			"codex non-stop uses additionalContext",
			Codex,
			[]emission{{"SessionStart", "hi"}},
			`{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"hi"}}` + "\n",
		},
		{
			"drift and contract coincide into one object",
			"",
			[]emission{{"SessionStart", "contract"}, {"SessionStart", "drift"}},
			`{"hookSpecificOutput":{"hookEventName":"SessionStart","additionalContext":"contract\n\ndrift"}}` + "\n",
		},
		{
			"identical contexts collapse",
			Codex,
			[]emission{{"PostToolUse", "same"}, {"PostToolUse", "same"}},
			`{"hookSpecificOutput":{"hookEventName":"PostToolUse","additionalContext":"same"}}` + "\n",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			out := NewOutput(testCase.dialect)
			for _, emitted := range testCase.emissions {
				out.Add(emitted.event, emitted.context)
			}
			got, err := out.Take()
			if err != nil {
				t.Fatalf("Take() error = %v", err)
			}
			if string(got) != testCase.want {
				t.Errorf("Take() = %q, want %q", got, testCase.want)
			}
			if len(got) > 0 && !json.Valid(got) {
				t.Errorf("Take() = %q is not valid JSON", got)
			}
			again, _ := out.Take()
			if again != nil {
				t.Errorf("second Take() = %q, want nil after reset", again)
			}
		})
	}
}
