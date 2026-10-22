package projectfile

import "testing"

func TestValidateOverrides(t *testing.T) {
	t.Parallel()

	const head = "version: 1\noverrides: "
	const tooMany = "overrides and tests: more than 256 globs in total"
	many := make([]int, 600)
	for idx := range many {
		many[idx] = 2
	}
	runValidateCases(t, []validateCase{
		{
			"without files",
			head + "[{analyzers: {file-size: {enabled: false}}}]\n",
			"overrides[0].files: at least one glob",
		},
		{"bad glob", head + "[{files: [\"\"], analyzers: {}}]\n", "overrides[0].files[0]:"},
		{
			"unknown analyzer",
			head + "[{files: [a], analyzers: {nope: {enabled: false}}}]\n",
			"overrides[0].analyzers.nope: unknown analyzer",
		},
		{
			"bad severity",
			head + "[{files: [a], analyzers: {file-size: {severity: x}}}]\n",
			"overrides[0].analyzers.file-size.severity: invalid severity",
		},
		{
			"params",
			head + "[{files: [a], analyzers: {big-comment: {params: {max_lines: 1}}}}]\n",
			"overrides[0].analyzers.big-comment.params: " +
				"per-path params are only supported for file-size",
		},
		{
			"sub id",
			head + "[{files: [a], analyzers: {golangci-lint/errcheck: {enabled: false}}}]\n",
			"",
		},
		{"exactly 256 globs in total", globOverrides(128, 128), ""},
		{"257 globs in total", globOverrides(128, 129), tooMany},
		{"two lists of 200 and 57 globs", globOverrides(200, 57), tooMany},
		{"two globs in each of 600 overrides", globOverrides(many...), tooMany},
	})
}

func TestValidateFileSizeParams(t *testing.T) {
	t.Parallel()

	const head = "version: 1\noverrides: "
	const prefix = "overrides[0].analyzers.file-size.params"
	scoped := func(params string) string {
		return head + "[{files: ['**/*.svelte'], analyzers: " +
			"{file-size: {params: " + params + "}}}]\n"
	}
	runValidateCases(t, []validateCase{
		{"max_lines alone", scoped("{max_lines: 150}"), ""},
		{"both limits", scoped("{max_lines: 150, error_lines: 290}"), ""},
		{"zero error_lines disables the cap", scoped("{error_lines: 0}"), ""},
		{"zero max_lines", scoped("{max_lines: 0}"), prefix + ".max_lines: must be an integer"},
		{"text max_lines", scoped("{max_lines: x}"), prefix + ".max_lines: must be an integer"},
		{"negative error_lines", scoped("{error_lines: -1}"), prefix + ".error_lines: must be"},
		{"unknown key", scoped("{nope: 1}"), prefix + ".nope: unknown param"},
		{
			"extensions is not a per-path key",
			scoped("{extensions: ['.svelte']}"),
			prefix + ".extensions: extensions is not supported per path",
		},
		{
			"error_lines equal to max_lines",
			scoped("{max_lines: 200, error_lines: 200}"),
			prefix + ".error_lines: must be above max_lines (200)",
		},
		{
			"error_lines below max_lines",
			scoped("{max_lines: 200, error_lines: 100}"),
			prefix + ".error_lines: must be above max_lines (200)",
		},
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
