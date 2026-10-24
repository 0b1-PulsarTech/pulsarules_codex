package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsWiresHook(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		seed    func(t *testing.T, dir string)
		want    bool
		wantErr bool
	}{
		{name: "no settings file", seed: func(*testing.T, string) {}},
		{
			name: "unrelated settings",
			seed: func(t *testing.T, dir string) {
				t.Helper()
				writeSettings(t, dir, `{"model":"x"}`)
			},
		},
		{
			name: "settings naming our script",
			seed: func(t *testing.T, dir string) {
				t.Helper()
				writeSettings(t, dir, `{"hooks":{"Stop":"bash x/skill-router-reminder.sh stop"}}`)
			},
			want: true,
		},
		{
			name: "settings is a directory",
			seed: func(t *testing.T, dir string) {
				t.Helper()
				if err := os.Mkdir(filepath.Join(dir, "settings.json"), 0o750); err != nil {
					t.Fatal(err)
				}
			},
			wantErr: true,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			testCase.seed(t, dir)
			got, err := SettingsWiresHook(dir)
			if (err != nil) != testCase.wantErr || got != testCase.want {
				t.Errorf("SettingsWiresHook = %v, %v, want %v (error %v)",
					got, err, testCase.want, testCase.wantErr)
			}
		})
	}
}

func writeSettings(tb testing.TB, dir, body string) {
	tb.Helper()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(body), 0o600); err != nil {
		tb.Fatal(err)
	}
}
