package jsondoc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func seed(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return path
}

func TestLoad(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		content    string
		absent     bool
		wantExists bool
		wantLen    int
		wantErr    bool
	}{
		{name: "absent file is an empty document", absent: true},
		{name: "empty file is an empty document", content: "", wantExists: true},
		{name: "whitespace file is an empty document", content: " \n\t", wantExists: true},
		{name: "null file is an empty document", content: "null", wantExists: true},
		{name: "object keeps its keys", content: `{"a":1,"b":2}`, wantExists: true, wantLen: 2},
		{name: "array is unparseable", content: `[1]`, wantErr: true},
		{name: "garbage is unparseable", content: `{not json`, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "missing.json")
			if !testCase.absent {
				path = seed(t, testCase.content)
			}
			doc, err := Load(path)
			if testCase.wantErr {
				if !errors.Is(err, ErrUnparseable) {
					t.Fatalf("Load err = %v, want ErrUnparseable", err)
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("err = %q, want it to name %q", err, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if doc.Exists() != testCase.wantExists {
				t.Errorf("Exists = %v, want %v", doc.Exists(), testCase.wantExists)
			}
			if doc.Len() != testCase.wantLen {
				t.Errorf("Len = %d, want %d", doc.Len(), testCase.wantLen)
			}
			if err = SetSection(doc, "added", 1); err != nil {
				t.Fatalf("SetSection: %v", err)
			}
		})
	}
}

func TestLoad_ReadError(t *testing.T) {
	t.Parallel()

	if _, err := Load(t.TempDir()); err == nil || errors.Is(err, ErrUnparseable) {
		t.Fatalf("Load(dir) err = %v, want a read error that is not ErrUnparseable", err)
	}
}

func TestSection(t *testing.T) {
	t.Parallel()

	path := seed(t, `{"list":["a","b"],"bad":"text"}`)
	doc, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	list, err := Section[[]string](doc, "list")
	if err != nil || len(list) != 2 {
		t.Fatalf("Section(list) = %v, %v, want 2 entries", list, err)
	}
	absent, err := Section[[]string](doc, "absent")
	if err != nil || absent != nil {
		t.Fatalf("Section(absent) = %v, %v, want nil, nil", absent, err)
	}
	_, err = Section[[]string](doc, "bad")
	if !errors.Is(err, ErrUnparseable) || !strings.Contains(err.Error(), "bad") {
		t.Fatalf("Section(bad) err = %v, want ErrUnparseable naming the key", err)
	}
}

func TestDocument_ChildAndPutChild(t *testing.T) {
	t.Parallel()

	path := seed(t, `{"skills":{"paths":["a"],"extra":1},"other":true,"text":"x"}`)
	doc, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, err = doc.Child("text"); !errors.Is(err, ErrUnparseable) {
		t.Fatalf("Child(text) err = %v, want ErrUnparseable", err)
	}
	child, err := doc.Child("skills")
	if err != nil {
		t.Fatalf("Child: %v", err)
	}
	badDoc, err := Load(seed(t, `{"skills":{"paths":1}}`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	badChild, err := badDoc.Child("skills")
	if err != nil {
		t.Fatalf("Child: %v", err)
	}
	if _, err = Section[[]string](badChild, "paths"); err == nil ||
		!strings.Contains(err.Error(), "skills.paths") {
		t.Fatalf("scoped err = %v, want it to name skills.paths", err)
	}

	child.Delete("extra")
	if err = doc.PutChild("skills", child); err != nil {
		t.Fatalf("PutChild: %v", err)
	}
	if got := string(doc.fields["skills"]); got != `{"paths":["a"]}` {
		t.Errorf("skills = %s, want only paths", got)
	}
	child.Delete("paths")
	if err = doc.PutChild("skills", child); err != nil {
		t.Fatalf("PutChild empty: %v", err)
	}
	if doc.Has("skills") {
		t.Error("an empty child must remove its key")
	}
	absent, err := doc.Child("never")
	if err != nil || absent.Len() != 0 {
		t.Errorf("Child(never) = %v, %v, want empty", absent, err)
	}
}

func TestDocument_RoundTrip(t *testing.T) {
	t.Parallel()

	path := seed(t, `{"user":{"keep":[1,2]},"mine":"old"}`)
	doc, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = SetSection(doc, "mine", "new"); err != nil {
		t.Fatalf("SetSection: %v", err)
	}
	if err = doc.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "{\n  \"mine\": \"new\",\n  \"user\": {\n    \"keep\": [\n      1,\n      2\n    ]\n  }\n}\n"
	if string(got) != want {
		t.Errorf("saved = %q, want %q", got, want)
	}
}

func TestSetSection_Unmarshalable(t *testing.T) {
	t.Parallel()

	doc, err := Load(filepath.Join(t.TempDir(), "x.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err = SetSection(doc, "bad", make(chan int)); err == nil {
		t.Fatal("SetSection(chan) err = nil, want a marshal error")
	}
	if doc.Has("bad") {
		t.Error("a failed marshal must not leave the key behind")
	}
}
