package disk

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRead(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		seed       string
		wantString string
	}{
		{name: "absent file returns no bytes and no error"},
		{name: "existing file returns its content", seed: `{"a":1}`, wantString: `{"a":1}`},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			path := filepath.Join(dir, "config.json")
			if testCase.seed != "" {
				if err := os.WriteFile(path, []byte(testCase.seed), 0o600); err != nil {
					t.Fatalf("seed: %v", err)
				}
			}

			got, err := ReadOptional(path)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if string(got) != testCase.wantString {
				t.Errorf("Read = %q, want %q", got, testCase.wantString)
			}
		})
	}
}

func TestRead_StatFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}
	path := filepath.Join(blocker, "config.json")

	if _, err := ReadOptional(path); err == nil {
		t.Error("expected an error when a path component is not a directory, got nil")
	}
}
