package knowledge

import (
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

func readOptionalYAML[Doc any](standards fs.FS, name string) (Doc, bool, error) {
	var loaded Doc
	raw, err := fs.ReadFile(standards, name)
	if err != nil {
		// Each of these files is optional; its absence is not an error.
		if errors.Is(err, fs.ErrNotExist) {
			return loaded, false, nil
		}
		return loaded, false, fmt.Errorf("read %s: %w", name, err)
	}
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return loaded, false, fmt.Errorf("parse %s: %w", name, err)
	}
	return loaded, true, nil
}

func readRouter(standards fs.FS, idx *Index) error {
	loaded, _, err := readOptionalYAML[routerFile](standards, "router.yaml")
	if err != nil {
		return err
	}
	idx.Router = loaded.Router
	return nil
}

func readReferences(standards fs.FS, idx *Index) error {
	loaded, _, err := readOptionalYAML[referencesFile](standards, "references.yaml")
	if err != nil {
		return err
	}
	for _, reference := range loaded.References {
		if reference.ID == "" {
			return errors.New("reference with empty id")
		}
		if idx.references.Has(reference.ID) {
			return fmt.Errorf("duplicate reference id %q", reference.ID)
		}
		idx.references.Put(reference.ID, reference)
		idx.References = append(idx.References, reference)
	}
	return nil
}

func readProfiles(standards fs.FS, idx *Index) error {
	loaded, _, err := readOptionalYAML[profilesFile](standards, "profiles.yaml")
	if err != nil {
		return err
	}
	for _, profile := range loaded.Profiles {
		if profile.ID == "" {
			return errors.New("profile with empty id")
		}
		if idx.profiles.Has(profile.ID) {
			return fmt.Errorf("duplicate profile id %q", profile.ID)
		}
		idx.profiles.Put(profile.ID, profile)
		idx.Profiles = append(idx.Profiles, profile)
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
		if _, ok := idx.skills.Get(sidecar.ID); !ok {
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
