package packaging

import (
	"flag"
	"path/filepath"
	"testing"
)

func TestCommand_Bind(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		args []string
		want string
	}{
		{name: "x", want: filepath.Join(".", "build", "standards-skills.zip")},
		{name: "explicit path", args: []string{"--out", "elsewhere"}, want: "elsewhere"},
		{
			name: "an explicit empty dir is the default",
			args: []string{"--out", ""},
			want: filepath.Join(".", "build", "standards-skills.zip"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var cmd Command
			fs := flag.NewFlagSet("package", flag.ContinueOnError)
			cmd.Bind(fs)
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := cmd.outPath(); got != testCase.want {
				t.Errorf("outPath() = %q, want %q", got, testCase.want)
			}
		})
	}
}
