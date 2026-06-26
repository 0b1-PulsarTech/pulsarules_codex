package knowledge

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"
)

// TestReadSkillSidecars_Loaded asserts every non-router skill ships a curated
// sidecar body that leads with the Mandatory workflow section, and that the
// router (which renders from its own template) is exempt.
func TestReadSkillSidecars_Loaded(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	for _, skill := range idx.Skills {
		body := idx.Body("skills", skill.ID)
		if skill.ID == "project-router" {
			if body != "" {
				t.Errorf("project-router must not carry a sidecar, got %d bytes", len(body))
			}
			continue
		}
		if body == "" {
			t.Errorf("skill %q has no sidecar body", skill.ID)
			continue
		}
		if !strings.Contains(body, "## Mandatory workflow") {
			t.Errorf("skill %q sidecar missing ## Mandatory workflow", skill.ID)
		}
		if !strings.Contains(body, "## Validation checklist") {
			t.Errorf("skill %q sidecar missing ## Validation checklist", skill.ID)
		}
	}
}

// TestReadRouter_Loaded asserts the embedded router.yaml populates the index
// with the baseline, dispatch table, and composition order.
func TestReadRouter_Loaded(t *testing.T) {
	t.Parallel()

	idx, _, err := Load("")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	router := idx.Router
	if len(router.Baseline.Always) == 0 {
		t.Error("router baseline.always is empty")
	}
	if len(router.Dispatch) == 0 {
		t.Error("router dispatch is empty")
	}
	if len(router.Order) == 0 {
		t.Error("router order is empty")
	}
	if router.Baseline.Always[0].Skill != "go-style" {
		t.Errorf("first baseline skill = %q, want go-style", router.Baseline.Always[0].Skill)
	}
}

// TestReadRouter_Optional asserts a knowledge base without router.yaml loads
// with an empty RouterSpec rather than failing.
func TestReadRouter_Optional(t *testing.T) {
	t.Parallel()

	standards := fstest.MapFS{
		"skills.yaml": {
			Data: []byte("skills:\n  - id: real\n    name: Real\n    order: 1\n"),
		},
		"rules/README.md":     {Data: []byte("# rules\n")},
		"patterns/README.md":  {Data: []byte("# patterns\n")},
		"workflows/README.md": {Data: []byte("# workflows\n")},
		"skills/real.md": {
			Data: []byte("---\nid: real\nname: Real\n---\n## Mandatory workflow\n"),
		},
	}

	idx, err := readIndex(standards)
	if err != nil {
		t.Fatalf("readIndex: %v", err)
	}
	if len(idx.Router.Dispatch) != 0 {
		t.Errorf("expected empty router, got %d dispatch rows", len(idx.Router.Dispatch))
	}
}

// TestReadRouter_Malformed asserts an unparseable router.yaml is reported, not
// silently ignored.
func TestReadRouter_Malformed(t *testing.T) {
	t.Parallel()

	standards := fstest.MapFS{
		"skills.yaml": {
			Data: []byte("skills:\n  - id: real\n    name: Real\n    order: 1\n"),
		},
		"rules/README.md":     {Data: []byte("# rules\n")},
		"patterns/README.md":  {Data: []byte("# patterns\n")},
		"workflows/README.md": {Data: []byte("# workflows\n")},
		"skills/real.md": {
			Data: []byte("---\nid: real\nname: Real\n---\n## Mandatory workflow\n"),
		},
		"router.yaml": {Data: []byte("router: [not, a, mapping]\n")},
	}

	if _, err := readIndex(standards); err == nil {
		t.Fatal("expected error for malformed router.yaml, got nil")
	} else {
		var yamlErr *yaml.TypeError
		if !errors.As(err, &yamlErr) {
			t.Fatalf("error = %v, want a YAML parse error wrapping yaml.TypeError", err)
		}
	}
}

// TestReadSkillSidecars_UnknownID asserts a sidecar whose id matches no
// declared skill is rejected, so a stale sidecar cannot drift from skills.yaml.
func TestReadSkillSidecars_UnknownID(t *testing.T) {
	t.Parallel()

	standards := fstest.MapFS{
		"skills.yaml": {
			Data: []byte("skills:\n  - id: real\n    name: Real\n    order: 1\n"),
		},
		"rules/README.md":     {Data: []byte("# rules\n")},
		"patterns/README.md":  {Data: []byte("# patterns\n")},
		"workflows/README.md": {Data: []byte("# workflows\n")},
		"skills/real.md": {
			Data: []byte("---\nid: real\nname: Real\n---\n## Mandatory workflow\n"),
		},
		"skills/ghost.md": {
			Data: []byte("---\nid: ghost\nname: Ghost\n---\n## Mandatory workflow\n"),
		},
	}

	if _, err := readIndex(standards); err == nil {
		t.Fatal("expected error for sidecar with unknown skill id, got nil")
	} else if !errors.Is(err, errUnknownSkill) {
		t.Fatalf("error = %v, want error wrapping errUnknownSkill", err)
	}
}

// TestReadSkillSidecars_DuplicateID asserts two sidecars for the same skill id
// are rejected.
func TestReadSkillSidecars_DuplicateID(t *testing.T) {
	t.Parallel()

	standards := fstest.MapFS{
		"skills.yaml": {
			Data: []byte("skills:\n  - id: real\n    name: Real\n    order: 1\n"),
		},
		"rules/README.md":     {Data: []byte("# rules\n")},
		"patterns/README.md":  {Data: []byte("# patterns\n")},
		"workflows/README.md": {Data: []byte("# workflows\n")},
		"skills/real.md": {
			Data: []byte("---\nid: real\nname: Real\n---\n## Mandatory workflow\n"),
		},
		"skills/real2.md": {
			Data: []byte("---\nid: real\nname: Real\n---\n## Mandatory workflow\n"),
		},
	}

	if _, err := readIndex(standards); err == nil {
		t.Fatal("expected error for duplicate sidecar id, got nil")
	} else if !errors.Is(err, errDuplicateSkill) {
		t.Fatalf("error = %v, want error wrapping errDuplicateSkill", err)
	}
}
