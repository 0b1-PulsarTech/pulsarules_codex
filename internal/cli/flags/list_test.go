package flags

import (
	"flag"
	"slices"
	"testing"
)

func TestListSet(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		initial []string
		values  []string
		want    []string
		wantNil bool
	}{
		{name: "never given keeps nil", wantNil: true},
		{name: "never given keeps default", initial: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "plain value", values: []string{"a"}, want: []string{"a"}},
		{name: "commas split", values: []string{"a,b"}, want: []string{"a", "b"}},
		{name: "empties dropped", values: []string{"a,,b"}, want: []string{"a", "b"}},
		{name: "spaces trimmed", values: []string{" a , b "}, want: []string{"a", "b"}},
		{name: "repeated accumulates", values: []string{"a", "b,c"}, want: []string{"a", "b", "c"}},
		{name: "duplicates kept", values: []string{"a", "a"}, want: []string{"a", "a"}},
		{name: "empty value leaves nil", values: []string{""}, wantNil: true},
		{name: "blank value is non-nil and empty", values: []string{" , "}, want: []string{}},
		{
			name:    "empty value replaces the default with nothing",
			initial: []string{"x"},
			values:  []string{""},
			wantNil: true,
		},
		{
			name:    "first value replaces the default",
			initial: []string{"x", "y"},
			values:  []string{"a"},
			want:    []string{"a"},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got := slices.Clone(testCase.initial)
			fs := flag.NewFlagSet("test", flag.ContinueOnError)
			fs.Var(NewList(&got), "list", "")
			for _, value := range testCase.values {
				if err := fs.Parse([]string{"--list", value}); err != nil {
					t.Fatalf("Parse(%q): %v", value, err)
				}
			}
			if testCase.wantNil {
				if got != nil {
					t.Fatalf("got %#v, want nil", got)
				}
				return
			}
			if got == nil || !slices.Equal(got, testCase.want) {
				t.Errorf("got %#v, want %#v", got, testCase.want)
			}
		})
	}
}

func TestListString(t *testing.T) {
	t.Parallel()

	values := []string{"a", "b"}
	if got := NewList(&values).String(); got != "a,b" {
		t.Errorf("String() = %q, want a,b", got)
	}
	if got := (*List)(nil).String(); got != "" {
		t.Errorf("nil String() = %q, want empty", got)
	}
	if got := NewList(nil).String(); got != "" {
		t.Errorf("nil target String() = %q, want empty", got)
	}
}
