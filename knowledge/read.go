package knowledge

import (
	"errors"
	"fmt"
	"io/fs"
)

var (
	errUnknownSkill   = errors.New("unknown skill")
	errDuplicateSkill = errors.New("duplicate skill sidecar")
)

func newIndex() *Index {
	return &Index{
		ruleByID:      map[string]Rule{},
		patternByID:   map[string]Pattern{},
		workflowByID:  map[string]Workflow{},
		skillByID:     map[string]Skill{},
		profileByID:   map[string]Profile{},
		referenceByID: map[string]Reference{},
		bodies:        map[string]string{},
	}
}

func loadKinds(standards fs.FS, idx *Index) error {
	kindLoaders := []struct {
		kind string
		load func(fs.FS) error
	}{
		{"rules", func(fsys fs.FS) error {
			return readKind(fsys, "rules", kindSink[Rule]{
				items: &idx.Rules, byID: idx.ruleByID, bodies: idx.bodies, idOf: ruleID,
			})
		}},
		{"patterns", func(fsys fs.FS) error {
			return readKind(fsys, "patterns", kindSink[Pattern]{
				items: &idx.Patterns, byID: idx.patternByID, bodies: idx.bodies, idOf: patternID,
			})
		}},
		{"workflows", func(fsys fs.FS) error {
			return readKind(fsys, "workflows", kindSink[Workflow]{
				items:  &idx.Workflows,
				byID:   idx.workflowByID,
				bodies: idx.bodies,
				idOf:   workflowID,
			})
		}},
	}
	for _, loader := range kindLoaders {
		if err := loader.load(standards); err != nil {
			return fmt.Errorf("load %s: %w", loader.kind, err)
		}
	}
	return nil
}

func readIndex(standards fs.FS) (*Index, error) {
	idx := newIndex()
	if err := loadKinds(standards, idx); err != nil {
		return nil, err
	}
	if err := readSkills(standards, idx); err != nil {
		return nil, err
	}
	if err := readSkillSidecars(standards, idx); err != nil {
		return nil, fmt.Errorf("load skill sidecars: %w", err)
	}
	if err := readReferences(standards, idx); err != nil {
		return nil, fmt.Errorf("load references: %w", err)
	}
	if err := readProfiles(standards, idx); err != nil {
		return nil, fmt.Errorf("load profiles: %w", err)
	}
	if err := readRouter(standards, idx); err != nil {
		return nil, fmt.Errorf("load router: %w", err)
	}
	return idx, nil
}
