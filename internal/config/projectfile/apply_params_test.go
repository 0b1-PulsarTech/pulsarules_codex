package projectfile

import (
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

func TestApplyTo_CarriesOverrideParams(t *testing.T) {
	t.Parallel()

	file, err := decodeFile([]byte("version: 1\n" +
		"overrides: [{files: ['**/*.svelte'], " +
		"analyzers: {file-size: {params: {max_lines: 150}}}}]\n"))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	cfg := config.Defaults()
	if err = file.ApplyTo(cfg); err != nil {
		t.Fatalf("ApplyTo() error = %v", err)
	}
	got := cfg.Overrides[0].Analyzers["file-size"].Params
	want := map[string]any{"max_lines": 150}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("override params = %v, want %v", got, want)
	}
}
