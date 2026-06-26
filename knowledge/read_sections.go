package knowledge

import (
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

func readRouter(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "router.yaml")
	if err != nil {
		// The file is optional; its absence leaves the router prose-only.
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read router.yaml: %w", err)
	}
	var loaded routerFile
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return fmt.Errorf("parse router.yaml: %w", err)
	}
	idx.Router = loaded.Router
	return nil
}

func readReferences(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "references.yaml")
	if err != nil {
		// The file is optional.
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read references.yaml: %w", err)
	}
	var loaded referencesFile
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return fmt.Errorf("parse references.yaml: %w", err)
	}
	for _, reference := range loaded.References {
		if reference.ID == "" {
			return errors.New("reference with empty id")
		}
		if _, dup := idx.referenceByID[reference.ID]; dup {
			return fmt.Errorf("duplicate reference id %q", reference.ID)
		}
		idx.referenceByID[reference.ID] = reference
		idx.References = append(idx.References, reference)
	}
	return nil
}

func readProfiles(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "profiles.yaml")
	if err != nil {
		// The file is optional; without it no layout variants are offered.
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read profiles.yaml: %w", err)
	}
	var loaded profilesFile
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return fmt.Errorf("parse profiles.yaml: %w", err)
	}
	for _, profile := range loaded.Profiles {
		if profile.ID == "" {
			return errors.New("profile with empty id")
		}
		if _, dup := idx.profileByID[profile.ID]; dup {
			return fmt.Errorf("duplicate profile id %q", profile.ID)
		}
		idx.profileByID[profile.ID] = profile
		idx.Profiles = append(idx.Profiles, profile)
	}
	return nil
}

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
