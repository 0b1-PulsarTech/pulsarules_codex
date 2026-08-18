package knowledge

import (
	"errors"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

// readRouter loads the project-router's baseline, dispatch table, and
// composition order from standards/router.yaml. The file is optional: without it
// the router renders only its static prose and the available-skills list.
func readRouter(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "router.yaml")
	if err != nil {
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

// readReferences loads the optional external references from standards/
// references.yaml. The file is optional: without it, rules/patterns that cite a
// reference fail validation, but a base with no citations renders unchanged.
func readReferences(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "references.yaml")
	if err != nil {
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
			return fmt.Errorf("reference with empty id")
		}
		if _, dup := idx.referenceByID[reference.ID]; dup {
			return fmt.Errorf("duplicate reference id %q", reference.ID)
		}
		idx.referenceByID[reference.ID] = reference
		idx.References = append(idx.References, reference)
	}
	return nil
}

// readProfiles loads the optional install-time customization profiles from
// standards/profiles.yaml. The file is optional: a knowledge base with no profiles
// simply offers no layout variants.
func readProfiles(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "profiles.yaml")
	if err != nil {
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
			return fmt.Errorf("profile with empty id")
		}
		if _, dup := idx.profileByID[profile.ID]; dup {
			return fmt.Errorf("duplicate profile id %q", profile.ID)
		}
		idx.profileByID[profile.ID] = profile
		idx.Profiles = append(idx.Profiles, profile)
	}
	return nil
}

// readSkillSidecars loads per-skill sidecar bodies from
// standards/skills/<id>.md: orientation text - what a skill governs, when
// to reach for it - transcluded as the skill head above the composed
// clauses. Optional at load time; an id matching no declared skill errors,
// so a stale sidecar cannot drift from skills.yaml unnoticed.
func readSkillSidecars(standards fs.FS, idx *Index) error {
	paths, pathsErr := markdownPaths(standards, "skills")
	if pathsErr != nil {
		if errors.Is(pathsErr, fs.ErrNotExist) {
			return nil // no sidecars present; validate reports missing coverage.
		}
		return pathsErr
	}
	for _, p := range paths {
		sidecar, body, err := readItem[struct {
			ID string `yaml:"id"`
		}](standards, p)
		if err != nil {
			return err
		}
		if sidecar.ID == "" {
			return fmt.Errorf("skills file %q has empty id", p)
		}
		if _, ok := idx.skillByID[sidecar.ID]; !ok {
			return fmt.Errorf(
				"sidecar %q references unknown skill %q: %w",
				p,
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
