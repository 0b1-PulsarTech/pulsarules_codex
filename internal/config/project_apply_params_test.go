package config

import "testing"

func TestValidateFileSizeParams(t *testing.T) {
	t.Parallel()

	runValidateCases(t, []validateCase{
		{
			"project level error_lines below max_lines",
			"version: 1\nanalyzers: {file-size: {params: {max_lines: 200, error_lines: 100}}}\n",
			"analyzers.file-size.params.error_lines: must be above max_lines (200)",
		},
	})
}

func TestValidateLimitChecksStayOnFileSize(t *testing.T) {
	t.Parallel()

	rules := testRules()
	rules.Params = map[string][]string{"big-comment": {"max_lines"}}
	file, err := decodeFile(
		[]byte("version: 1\nanalyzers: {big-comment: {params: {max_lines: 0}}}\n"),
	)
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	if err = file.Validate(rules); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}
