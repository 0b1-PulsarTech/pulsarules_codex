package install

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/language/detect"
)

func TestResolveLanguages_Global(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		override []string
		want     []string
		wantErr  bool
	}{
		{name: "no override means every language", want: detect.NewSet(detect.All()...).Strings()},
		{name: "override is used", override: []string{"go"}, want: []string{"go"}},
		{name: "bad override", override: []string{"cobol"}, wantErr: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			home := t.TempDir()
			cfg := installConfig{absBase: home, global: true}
			set, err := resolveLanguages(cfg, testCase.override)
			if testCase.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil || !slices.Equal(set.Strings(), testCase.want) {
				t.Fatalf("languages = %v, %v, want %v", set.Strings(), err, testCase.want)
			}
			configPath := filepath.Join(home, detect.ConfigFileName)
			if _, statErr := os.Stat(configPath); statErr == nil {
				t.Errorf("%s was written under the home directory", detect.ConfigFileName)
			}
		})
	}
}
