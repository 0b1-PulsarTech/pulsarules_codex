package render

import (
	"strings"
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/knowledge"
)

func TestInstallFilter(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		installed []string
		probe     string
		want      bool
	}{
		{"empty keeps everything", nil, "anything", true},
		{"subset keeps member", []string{"go-style"}, "go-style", true},
		{"subset drops non-member", []string{"go-style"}, "rest-adapter", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			keep := installFilter(testCase.installed)
			if got := keep(testCase.probe); got != testCase.want {
				t.Fatalf("keep(%q) = %v, want %v", testCase.probe, got, testCase.want)
			}
		})
	}
}

func TestFilterDispatch(t *testing.T) {
	t.Parallel()

	rows := []knowledge.RouterDispatchRow{
		{Signal: "a", Skills: []string{"go-style", "rest-adapter"}},
		{Signal: "b", Skills: []string{"observability"}},
	}
	got := filterDispatch(rows, installFilter([]string{"go-style"}))

	if len(got) != 1 {
		t.Fatalf("rows = %d, want 1 (the all-uninstalled row dropped)", len(got))
	}
	if len(got[0].Skills) != 1 || got[0].Skills[0] != "go-style" {
		t.Fatalf("row skills = %v, want [go-style]", got[0].Skills)
	}
}

func TestFilterOrder(t *testing.T) {
	t.Parallel()

	steps := []knowledge.RouterOrderStep{
		{Skills: []string{"code-placement"}, Note: "n"},
		{Skills: []string{"observability"}, Note: "n"},
	}
	got := filterOrder(steps, installFilter([]string{"code-placement"}))

	if len(got) != 1 || len(got[0].Skills) != 1 || got[0].Skills[0] != "code-placement" {
		t.Fatalf("steps = %v, want one code-placement step", got)
	}
}

// TestRenderRouter_FilterTrims proves a subset filter drops uninstalled skills,
// dispatch rows, and the integration-tests callout, while the full (nil) render
// keeps them.
func TestRenderRouter_FilterTrims(t *testing.T) {
	t.Parallel()

	idx, templates, err := knowledge.Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	rnd, err := NewRenderer(templates)
	if err != nil {
		t.Fatalf("NewRenderer: %v", err)
	}

	full, err := rnd.RenderRouter(idx, nil)
	if err != nil {
		t.Fatalf("RenderRouter full: %v", err)
	}
	filtered, err := rnd.RenderRouter(idx, []string{"usecase-layout", "rest-adapter"})
	if err != nil {
		t.Fatalf("RenderRouter filtered: %v", err)
	}

	if !strings.Contains(full, "`database-persistence`") {
		t.Error("full router missing database-persistence")
	}
	if strings.Contains(filtered, "database-persistence") {
		t.Error("filtered router should not mention database-persistence")
	}
	if !strings.Contains(filtered, "`rest-adapter`") {
		t.Error("filtered router missing rest-adapter")
	}
	if !strings.Contains(full, "Tests are not optional routing") {
		t.Error("full router should keep the integration-tests callout")
	}
	if strings.Contains(filtered, "Tests are not optional routing") {
		t.Error("filtered router should drop the integration-tests callout")
	}
}
