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
	return &Index{bodies: map[string]string{}, summaries: map[string]string{}}
}

func loadKinds(standards fs.FS, idx *Index) error {
	kindLoaders := []struct {
		kind Kind
		load func(fs.FS) error
	}{
		{KindRule, func(fsys fs.FS) error {
			return readKind(fsys, KindRule, kindSink[Rule]{
				items: &idx.Rules, byID: &idx.rules,
				bodies: idx.bodies, summaries: idx.summaries, idOf: ruleID,
			})
		}},
		{KindPattern, func(fsys fs.FS) error {
			return readKind(fsys, KindPattern, kindSink[Pattern]{
				items: &idx.Patterns, byID: &idx.patterns,
				bodies: idx.bodies, summaries: idx.summaries, idOf: patternID,
			})
		}},
		{KindWorkflow, func(fsys fs.FS) error {
			return readKind(fsys, KindWorkflow, kindSink[Workflow]{
				items: &idx.Workflows, byID: &idx.workflows,
				bodies: idx.bodies, summaries: idx.summaries, idOf: workflowID,
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
