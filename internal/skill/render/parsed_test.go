package render

import (
	"slices"
	"testing"
)

func TestUnknownSections(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		body    string
		want    []string
		wantErr bool
	}{
		{
			"canonical only",
			`pre {{define "must"}}x{{end}}{{define "forbidden"}}y{{end}}`,
			nil,
			false,
		},
		{
			"non-canonical", `{{define "must"}}x{{end}}{{define "musty"}}y{{end}}`,
			[]string{"musty"},
			false,
		},
		{"malformed", "{{define bad", nil, true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := UnknownSections(testCase.body)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, testCase.wantErr)
			}
			if !slices.Equal(got, testCase.want) {
				t.Errorf("UnknownSections = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestBodyParser_ParseCaches(t *testing.T) {
	t.Parallel()

	parser := newBodyParser()
	first, err := parser.parse(`{{define "must"}}x{{end}}`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	second, err := parser.parse(`{{define "must"}}x{{end}}`)
	if err != nil {
		t.Fatalf("parse again: %v", err)
	}
	if first.tmpl != second.tmpl {
		t.Error("the same body parsed twice, want one cached template")
	}
	if len(parser.cache) != 1 {
		t.Errorf("cache size = %d, want 1", len(parser.cache))
	}
}
