package generate

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
		{name: "defaults to ./generated", want: filepath.Join(".", "generated")},
		{name: "explicit dir", args: []string{"--out", "elsewhere"}, want: "elsewhere"},
		{
			name: "an explicit empty dir is the default",
			args: []string{"--out", ""},
			want: filepath.Join(".", "generated"),
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var cmd Command
			fs := flag.NewFlagSet("generate", flag.ContinueOnError)
			cmd.Bind(fs)
			if err := fs.Parse(testCase.args); err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := cmd.outDir(); got != testCase.want {
				t.Errorf("outDir() = %q, want %q", got, testCase.want)
			}
		})
	}
}
