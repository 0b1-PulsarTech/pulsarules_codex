package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/jsondoc"
)

func TestMergeSkills(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seed    map[string]string
		want    string
		wantErr bool
	}{
		{
			name: "no skills section gets the paths entry",
			seed: map[string]string{},
			want: `{"paths":[".opencode/skills"]}`,
		},
		{
			name: "an existing path is kept before ours",
			seed: map[string]string{"skills": `{"paths":["custom"]}`},
			want: `{"paths":["custom",".opencode/skills"]}`,
		},
		{
			name: "our path already present is not duplicated",
			seed: map[string]string{"skills": `{"paths":[".opencode/skills"]}`},
			want: `{"paths":[".opencode/skills"]}`,
		},
		{
			name: "other skills keys survive",
			seed: map[string]string{"skills": `{"urls":["https://example.test"]}`},
			want: `{"paths":[".opencode/skills"],"urls":["https://example.test"]}`,
		},
		{
			name:    "a skills section that is not an object fails",
			seed:    map[string]string{"skills": `["x"]`},
			wantErr: true,
		},
		{
			name:    "skills paths that is not a list fails",
			seed:    map[string]string{"skills": `{"paths":"custom"}`},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			content := "{}"
			for key, value := range testCase.seed {
				content = `{"` + key + `":` + value + `}`
			}
			path := filepath.Join(t.TempDir(), configFile)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatalf("seed: %v", err)
			}
			doc, err := jsondoc.Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			err = mergeSkills(doc)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("mergeSkills error = %v, want error %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			got, err := jsondoc.Section[json.RawMessage](doc, "skills")
			if err != nil {
				t.Fatalf("Section: %v", err)
			}
			if string(got) != testCase.want {
				t.Errorf("skills = %s, want %s", got, testCase.want)
			}
		})
	}
}
