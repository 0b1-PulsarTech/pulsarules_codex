package command

import (
	"flag"
	"testing"
)

func TestProject(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{name: "unset is the working directory", want: "."},
		{name: "empty is the working directory", args: []string{"--project", ""}, want: "."},
		{name: "explicit", args: []string{"--project", "p"}, want: "p"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var project Project
			fs := flag.NewFlagSet("x", flag.ContinueOnError)
			project.BindProject(fs, "project `DIR`")
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := project.ProjectDir(); got != testCase.want {
				t.Errorf("ProjectDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}
