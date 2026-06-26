package cli

import "testing"

func TestJoinOrDash(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name  string
		items []string
		want  string
	}{
		{name: "nil renders a dash", items: nil, want: "-"},
		{name: "empty renders a dash", items: []string{}, want: "-"},
		{name: "single item stands alone", items: []string{"go-style"}, want: "go-style"},
		{
			name:  "several items are comma separated",
			items: []string{"go-style", "commits", "security"},
			want:  "go-style, commits, security",
		},
		{name: "an empty entry is kept, not skipped", items: []string{"a", ""}, want: "a, "},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := joinOrDash(testCase.items); got != testCase.want {
				t.Errorf("joinOrDash(%q) = %q, want %q", testCase.items, got, testCase.want)
			}
		})
	}
}
