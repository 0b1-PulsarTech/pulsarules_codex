package knowledge

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"

	"gopkg.in/yaml.v3"
)

func TestReadSkillSidecars_NonRouter(t *testing.T) {
	t.Parallel()

	idx, err := Load("")
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
		if strings.Contains(body, "## Mandatory workflow") {
			t.Errorf("skill %q sidecar still hand-duplicates ## Mandatory workflow", skill.ID)
		}
		if strings.Contains(body, "## Validation checklist") {
			t.Errorf("skill %q sidecar still hand-duplicates ## Validation checklist", skill.ID)
		}
		if strings.Contains(body, "## Forbidden actions") && skill.ID != "git-history" {
			t.Errorf("skill %q sidecar still hand-duplicates ## Forbidden actions", skill.ID)
		}
	}
}

func TestReadRouter_Loaded(t *testing.T) {
	t.Parallel()

	idx, err := Load("")
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

func TestReadRouter_MissingYAML(t *testing.T) {
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

func TestReadRouter_MalformedYAMLReturnsError(t *testing.T) {
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
		if _, ok := errors.AsType[*yaml.TypeError](err); !ok {
			t.Fatalf("error = %v, want a YAML parse error wrapping yaml.TypeError", err)
		}
	}
}

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

func TestReadSkillSidecars_DuplicateIDRejected(t *testing.T) {
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
