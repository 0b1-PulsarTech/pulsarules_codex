package fsx

import (
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

func deleting(key string) func(map[string]string) bool {
	return func(servers map[string]string) bool {
		delete(servers, key)
		return true
	}
}

func TestStripMapSection(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		seed        json.RawMessage
		strip       func(map[string]string) bool
		wantChanged bool
		wantAbsent  bool
		wantValue   string
	}{
		{
			name:        "absent key is a no-op",
			strip:       deleting("gone"),
			wantChanged: false,
			wantAbsent:  true,
		},
		{
			name:        "strip reporting no change leaves config untouched",
			seed:        json.RawMessage(`{"a":"1"}`),
			strip:       func(map[string]string) bool { return false },
			wantChanged: false,
			wantValue:   `{"a":"1"}`,
		},
		{
			name:        "empty result deletes the key",
			seed:        json.RawMessage(`{"a":"1"}`),
			strip:       deleting("a"),
			wantChanged: true,
			wantAbsent:  true,
		},
		{
			name:        "non-empty result is re-marshaled back",
			seed:        json.RawMessage(`{"a":"1","b":"2"}`),
			strip:       deleting("a"),
			wantChanged: true,
			wantValue:   `{"b":"2"}`,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			config := map[string]json.RawMessage{}
			if testCase.seed != nil {
				config["servers"] = testCase.seed
			}

			changed, err := StripMapSection(config, "servers", "servers", testCase.strip)
			if err != nil {
				t.Fatalf("StripMapSection: %v", err)
			}
			if changed != testCase.wantChanged {
				t.Errorf("changed = %v, want %v", changed, testCase.wantChanged)
			}

			raw, present := config["servers"]
			if testCase.wantAbsent {
				if present {
					t.Errorf("servers = %s, want absent", raw)
				}
				return
			}
			if string(raw) != testCase.wantValue {
				t.Errorf("servers = %s, want %s", raw, testCase.wantValue)
			}
		})
	}
}

func TestStripMapSection_Unmarshalable(t *testing.T) {
	t.Parallel()

	config := map[string]json.RawMessage{"servers": json.RawMessage(`{"a":"1"}`)}
	changed, err := StripMapSection(config, "servers", "servers",
		func(servers map[string]json.RawMessage) bool {
			servers["a"] = json.RawMessage(`{not valid`)
			return true
		},
	)
	if err == nil {
		t.Fatal("StripMapSection err = nil, want a marshal error")
	}
	if changed {
		t.Error("changed = true, want false on marshal failure")
	}
}

func TestStripMapSection_InvalidJSON(t *testing.T) {
	t.Parallel()

	config := map[string]json.RawMessage{"servers": json.RawMessage(`not json`)}
	_, err := StripMapSection(
		config,
		"servers",
		"servers",
		func(map[string]string) bool { return false },
	)
	if !errors.Is(err, ErrUnparseableJSON) {
		t.Fatalf("err = %v, want ErrUnparseableJSON", err)
	}
}

func TestStripSliceSection_DeleteFunc(t *testing.T) {
	t.Parallel()

	config := map[string]json.RawMessage{"tags": json.RawMessage(`["a","b"]`)}
	changed, err := StripSliceSection(config, "tags", "tags", func(tags *[]string) bool {
		before := len(*tags)
		*tags = slices.DeleteFunc(*tags, func(entry string) bool { return entry == "a" })
		return len(*tags) != before
	})
	if err != nil {
		t.Fatalf("StripSliceSection: %v", err)
	}
	if !changed {
		t.Fatal("changed = false, want true")
	}
	if got := string(config["tags"]); got != `["b"]` {
		t.Errorf("tags = %s, want [\"b\"]", got)
	}
}
