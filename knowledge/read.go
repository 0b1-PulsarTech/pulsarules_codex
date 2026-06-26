package knowledge

import (
	"fmt"
	"io/fs"
)

func readIndex(standards fs.FS) (*Index, error) {
	idx := &Index{
		ruleByID:     map[string]Rule{},
		patternByID:  map[string]Pattern{},
		workflowByID: map[string]Workflow{},
		bodies:       map[string]string{},
	}
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
			return nil, fmt.Errorf("load %s: %w", loader.kind, err)
		}
	}
	return idx, nil
}
