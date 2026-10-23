package knowledge

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/safeyaml"
)

func readOptionalYAML[Doc any](standards fs.FS, name string) (Doc, error) {
	var loaded Doc
	raw, err := fs.ReadFile(standards, name)
	if err != nil {
		// Each of these files is optional; its absence is not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return loaded, nil
		}
		return loaded, fmt.Errorf("read %s: %w", name, err)
	}
	if err = safeyaml.Unmarshal(raw, &loaded); err != nil {
		return loaded, fmt.Errorf("parse %s: %w", name, err)
	}
	return loaded, nil
}

func readRouter(standards fs.FS, idx *Index) error {
	loaded, err := readOptionalYAML[routerFile](standards, "router.yaml")
	if err != nil {
		return err
	}
	idx.Router = loaded.Router
	return nil
}

func readReferences(standards fs.FS, idx *Index) error {
	loaded, err := readOptionalYAML[referencesFile](standards, "references.yaml")
	if err != nil {
		return err
	}
	for _, reference := range loaded.References {
		if reference.ID == "" {
			return errors.New("reference with empty id")
		}
		if !idx.references.Add(reference.ID, reference) {
			return fmt.Errorf("duplicate reference id %q", reference.ID)
		}
	}
	return nil
}

func readProfiles(standards fs.FS, idx *Index) error {
	loaded, err := readOptionalYAML[profilesFile](standards, "profiles.yaml")
	if err != nil {
		return err
	}
	for _, profile := range loaded.Profiles {
		if profile.ID == "" {
			return errors.New("profile with empty id")
		}
		if !idx.profiles.Add(profile.ID, profile) {
			return fmt.Errorf("duplicate profile id %q", profile.ID)
		}
	}
	return nil
}

func readSkillSidecars(standards fs.FS, idx *Index) error {
	paths, pathsErr := markdownPaths(standards, "skills")
	if pathsErr != nil {
		if errors.Is(pathsErr, fs.ErrNotExist) {
			// No sidecars present; validate reports missing coverage.
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
		if !idx.skills.Has(sidecar.ID) {
			return fmt.Errorf(
				"sidecar %q references unknown skill %q: %w",
				path,
				sidecar.ID,
				errUnknownSkill,
			)
		}
		bodyKey := KindSkill.key(sidecar.ID)
		if _, dup := idx.bodies[bodyKey]; dup {
			return fmt.Errorf("duplicate skill sidecar id %q: %w", sidecar.ID, errDuplicateSkill)
		}
		idx.bodies[bodyKey] = body
	}
	return nil
}
