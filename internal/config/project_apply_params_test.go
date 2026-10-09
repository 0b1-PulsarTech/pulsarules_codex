package config

import (
	"reflect"
	"testing"
)

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

func TestApplyProject_CarriesOverrideParams(t *testing.T) {
	t.Parallel()

	file, err := decodeFile([]byte("version: 1\n" +
		"overrides: [{files: ['**/*.svelte'], " +
		"analyzers: {file-size: {params: {max_lines: 150}}}}]\n"))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	cfg := Defaults()
	if err = cfg.ApplyProject(file); err != nil {
		t.Fatalf("ApplyProject() error = %v", err)
	}
	got := cfg.Overrides[0].Analyzers["file-size"].Params
	want := map[string]any{"max_lines": 150}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("override params = %v, want %v", got, want)
	}
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
