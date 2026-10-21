package flags

import (
	"flag"
	"slices"
	"testing"
)

func TestRepeatableSet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		values []string
		want   []string
	}{
		{name: "never given stays nil"},
		{name: "plain value", values: []string{"claude"}, want: []string{"claude"}},
		{
			name:   "commas are not split",
			values: []string{"claude,opencode"},
			want:   []string{"claude,opencode"},
		},
		{
			name:   "spaces are not trimmed",
			values: []string{" opencode "},
			want:   []string{" opencode "},
		},
		{name: "empty value is kept", values: []string{""}, want: []string{""}},
		{name: "blank value is kept", values: []string{" "}, want: []string{" "}},
		{name: "lone comma is kept", values: []string{","}, want: []string{","}},
		{name: "trailing comma is kept", values: []string{"claude,"}, want: []string{"claude,"}},
		{
			name:   "repeats accumulate in order",
			values: []string{"claude", "", "cursor"},
			want:   []string{"claude", "", "cursor"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.Var(NewRepeatable(&got), "list", "")
			for _, value := range testCase.values {
				if err := fs.Parse([]string{"--list", value}); err != nil {
					t.Fatalf("Parse(%q): %v", value, err)
				}
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("got %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestRepeatableString(t *testing.T) {
	t.Parallel()

	values := []string{"a", "b"}
	if got := NewRepeatable(&values).String(); got != "a,b" {
		t.Errorf("String() = %q, want a,b", got)
	}
	if got := (*Repeatable)(nil).String(); got != "" {
		t.Errorf("nil String() = %q, want empty", got)
	}
	if got := NewRepeatable(nil).String(); got != "" {
		t.Errorf("nil target String() = %q, want empty", got)
	}
}
