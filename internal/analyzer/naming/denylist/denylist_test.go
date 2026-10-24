package denylist

import (
	"reflect"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

func TestAnalyze_InertWithoutTokens(t *testing.T) {
	t.Parallel()

	ctx := &pass.Context{ChangedFiles: []pass.FileChange{{Path: "draft.go", Extension: ".go"}}}
	if got := NewAnalyzer().Analyze(ctx); len(got) != 0 {
		t.Fatalf("findings = %v, want none while tokens is empty", got)
	}
}

func TestDescribe(t *testing.T) {
	t.Parallel()

	want := pass.Descriptor{
		ID:     "forbidden-token",
		Rule:   "naming",
		Params: []string{"tokens", "allowed_names"},
	}
	if got := Describe(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Describe() = %+v, want %+v", got, want)
	}
}
