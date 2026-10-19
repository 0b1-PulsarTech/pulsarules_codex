package templates

import (
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestSet_Text(t *testing.T) {
	t.Parallel()

	set := New(fstest.MapFS{"a.txt": {Data: []byte("  hello \n")}})
	testCases := []struct {
		name    string
		asset   string
		want    string
		wantErr string
	}{
		{"trims the asset", "a.txt", "hello", ""},
		{"missing asset", "b.txt", "", "read b.txt"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := set.Text(testCase.asset)
			if got != testCase.want {
				t.Errorf("Text = %q, want %q", got, testCase.want)
			}
			if (err == nil) != (testCase.wantErr == "") ||
				(err != nil && !strings.Contains(err.Error(), testCase.wantErr)) {
				t.Errorf("Text err = %v, want %q", err, testCase.wantErr)
			}
		})
	}
}

func TestSet_Render(t *testing.T) {
	t.Parallel()

	set := New(fstest.MapFS{
		"ok.tmpl":   {Data: []byte("hi {{ .Name }}\n")},
		"bad.tmpl":  {Data: []byte("{{ .Name")},
		"exec.tmpl": {Data: []byte("{{ .Name.Deep }}")},
	})
	testCases := []struct {
		name    string
		asset   string
		want    string
		wantErr string
	}{
		{"renders and trims", "ok.tmpl", "hi dev", ""},
		{"missing asset", "none.tmpl", "", "read none.tmpl"},
		{"parse failure", "bad.tmpl", "", "parse bad.tmpl"},
		{"execute failure", "exec.tmpl", "", "render exec.tmpl"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			got, err := set.Render(testCase.asset, struct{ Name string }{"dev"})
			if got != testCase.want {
				t.Errorf("Render = %q, want %q", got, testCase.want)
			}
			if (err == nil) != (testCase.wantErr == "") ||
				(err != nil && !strings.Contains(err.Error(), testCase.wantErr)) {
				t.Errorf("Render err = %v, want %q", err, testCase.wantErr)
			}
		})
	}
}

func TestSet_Open(t *testing.T) {
	t.Parallel()

	set := New(fstest.MapFS{"a.txt": {Data: []byte("hello")}})
	raw, err := fs.ReadFile(set, "a.txt")
	if err != nil || string(raw) != "hello" {
		t.Errorf("ReadFile = %q, %v; want hello, nil", raw, err)
	}
	if _, err = fs.ReadFile(set, "b.txt"); err == nil {
		t.Error("ReadFile of a missing asset: err = nil, want an error")
	}
}
