package jsondoc

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

func docWith(t *testing.T, key, raw string) *Document {
	t.Helper()
	content := "{}"
	if raw != "" {
		content = `{"` + key + `":` + raw + `}`
	}
	doc, err := Load(seed(t, content))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return doc
}

func deleting(key string) func(map[string]string) bool {
	return func(servers map[string]string) bool {
		delete(servers, key)
		return true
	}
}

func TestStripMap(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        string
		strip       func(map[string]string) bool
		wantChanged bool
		wantAbsent  bool
		wantValue   string
	}{
		{
			name:       "absent key is a no-op",
			strip:      deleting("gone"),
			wantAbsent: true,
		},
		{
			name:      "strip reporting no change leaves config untouched",
			seed:      `{"a":"1"}`,
			strip:     func(map[string]string) bool { return false },
			wantValue: `{"a":"1"}`,
		},
		{
			name:        "empty result deletes the key",
			seed:        `{"a":"1"}`,
			strip:       deleting("a"),
			wantChanged: true,
			wantAbsent:  true,
		},
		{
			name:        "non-empty result is re-marshaled back",
			seed:        `{"a":"1","b":"2"}`,
			strip:       deleting("a"),
			wantChanged: true,
			wantValue:   `{"b":"2"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			doc := docWith(t, "servers", testCase.seed)
			changed, err := StripMap(doc, "servers", testCase.strip)
			if err != nil {
				t.Fatalf("StripMap: %v", err)
			}
			if changed != testCase.wantChanged {
				t.Errorf("changed = %v, want %v", changed, testCase.wantChanged)
			}
			if testCase.wantAbsent {
				if doc.Has("servers") {
					t.Errorf("servers = %s, want absent", doc.fields["servers"])
				}
				return
			}
			if got := string(doc.fields["servers"]); got != testCase.wantValue {
				t.Errorf("servers = %s, want %s", got, testCase.wantValue)
			}
		})
	}
}

func TestStripMap_Unmarshalable(t *testing.T) {
	t.Parallel()

	doc := docWith(t, "servers", `{"a":"1"}`)
	changed, err := StripMap(doc, "servers", func(servers map[string]json.RawMessage) bool {
		servers["a"] = json.RawMessage(`{not valid`)
		return true
	})
	if err == nil {
		t.Fatal("StripMap err = nil, want a marshal error")
	}
	if changed {
		t.Error("changed = true, want false on marshal failure")
	}
}

func TestStripMap_InvalidJSON(t *testing.T) {
	t.Parallel()

	doc := docWith(t, "servers", `"not an object"`)
	_, err := StripMap(doc, "servers", func(map[string]string) bool { return false })
	if !errors.Is(err, ErrUnparseable) {
		t.Fatalf("err = %v, want ErrUnparseable", err)
	}
	if !strings.Contains(err.Error(), " servers: ") {
		t.Errorf("err = %q, want it to name the section", err.Error())
	}
}

func TestStripSlice_DeleteFunc(t *testing.T) {
	t.Parallel()

	doc := docWith(t, "tags", `["a","b"]`)
	changed, err := StripSlice(doc, "tags", func(tags *[]string) bool {
		before := len(*tags)
		*tags = slices.DeleteFunc(*tags, func(entry string) bool { return entry == "a" })
		return len(*tags) != before
	})
	if err != nil {
		t.Fatalf("StripSlice: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got := string(doc.fields["tags"]); got != `["b"]` {
		t.Errorf("tags = %s, want [\"b\"]", got)
	}
}
