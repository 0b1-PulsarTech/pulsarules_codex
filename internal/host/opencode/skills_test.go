package opencode

import (
	"encoding/json"
	"testing"
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

			config := map[string]json.RawMessage{}
			for key, value := range testCase.seed {
				config[key] = json.RawMessage(value)
			}
			err := mergeSkills(config)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("mergeSkills error = %v, want error %v", err, testCase.wantErr)
			}
			if testCase.wantErr {
				return
			}
			if got := string(config["skills"]); got != testCase.want {
				t.Errorf("skills = %s, want %s", got, testCase.want)
			}
		})
	}
}
