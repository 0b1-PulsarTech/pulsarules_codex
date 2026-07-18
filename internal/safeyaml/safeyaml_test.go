package safeyaml

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const panicDoc = "b: &b {x: 1}\n<<: ~\n*b : 1\n"

type yamlCase struct {
	name    string
	doc     string
	wantErr string
	wantMap map[string]string
}

func cases() []yamlCase {
	return []yamlCase{
		{name: "plain success", doc: "name: value\n", wantMap: map[string]string{"name": "value"}},
		{name: "yaml error passes through", doc: "a: [b", wantErr: "yaml: line 1: did not find"},
		{
			name:    "panic becomes an error",
			doc:     panicDoc,
			wantErr: "yaml: runtime error: hash of unhashable type",
		},
	}
}

func checkResult(t *testing.T, testCase yamlCase, err error, got map[string]string) {
	t.Helper()
	if testCase.wantErr != "" {
		if err == nil || !strings.HasPrefix(err.Error(), testCase.wantErr) {
			t.Fatalf("error = %v, want prefix %q", err, testCase.wantErr)
		}
		return
	}
	if err != nil || got["name"] != testCase.wantMap["name"] {
		t.Fatalf("got %v, %v, want %v", got, err, testCase.wantMap)
	}
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	for _, testCase := range cases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var out map[string]any
			err := Unmarshal([]byte(testCase.doc), &out)
			got := map[string]string{}
			if value, ok := out["name"].(string); ok {
				got["name"] = value
			}
			checkResult(t, testCase, err, got)
		})
	}
}

func TestDecodeNode(t *testing.T) {
	t.Parallel()

	testCases := []yamlCase{
		{name: "plain success", doc: "name: value\n", wantMap: map[string]string{"name": "value"}},
		{
			name:    "yaml error passes through",
			doc:     "name: [a]\n",
			wantErr: "yaml: unmarshal errors:\n  line 1: cannot unmarshal !!seq into string",
		},
		{
			name:    "panic becomes an error",
			doc:     panicDoc,
			wantErr: "yaml: runtime error: hash of unhashable type",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			var root yaml.Node
			if err := yaml.Unmarshal([]byte(testCase.doc), &root); err != nil {
				t.Fatalf("parse into a Node: %v", err)
			}
			var out map[string]string
			err := DecodeNode(&root, &out)
			checkResult(t, testCase, err, out)
		})
	}
}

func TestDecodeNext(t *testing.T) {
	t.Parallel()

	for _, testCase := range cases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			decoder := yaml.NewDecoder(bytes.NewReader([]byte(testCase.doc)))
			var out map[string]string
			err := DecodeNext(decoder, &out)
			checkResult(t, testCase, err, out)
		})
	}
}

func TestYamlV3StillPanicsOnTheRepro(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("yaml.v3 no longer panics on the repro: re-judge the safeyaml guard")
		}
	}()
	var out map[string]string
	_ = yaml.Unmarshal([]byte(panicDoc), &out)
}
