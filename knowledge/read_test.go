package knowledge

import (
	"errors"
	"strings"
	"testing"
	"testing/fstest"
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
