package knowledge

import (
	"errors"
	"fmt"
	"io/fs"
)

func readSkillSidecars(standards fs.FS, idx *Index) error {
	paths, pathsErr := markdownPaths(standards, "skills")
	if pathsErr != nil {
		if errors.Is(pathsErr, fs.ErrNotExist) {
			return nil
		}
		return pathsErr
	}
	for _, path := range paths {
		sidecar, body, err := readItem[struct {
			ID string `yaml:"id"`
		}](standards, path)
		if err != nil {
			return err
		}
		if sidecar.ID == "" {
			return fmt.Errorf("skills file %q has empty id", path)
		}
		if _, ok := idx.skillByID[sidecar.ID]; !ok {
			return fmt.Errorf(
				"sidecar %q references unknown skill %q: %w",
				path,
				sidecar.ID,
				errUnknownSkill,
			)
		}
		bodyKey := "skills:" + sidecar.ID
		if _, dup := idx.bodies[bodyKey]; dup {
			return fmt.Errorf("duplicate skill sidecar id %q: %w", sidecar.ID, errDuplicateSkill)
		}
		idx.bodies[bodyKey] = body
	}
	return nil
}
