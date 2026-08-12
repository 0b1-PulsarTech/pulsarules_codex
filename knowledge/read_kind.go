package knowledge

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func ruleID(r Rule) string         { return r.ID }
func patternID(p Pattern) string   { return p.ID }
func workflowID(w Workflow) string { return w.ID }

// kindSink collects one knowledge kind: the ordered items, the id lookup, and
// the bodies/summaries kept for transclusion.
type kindSink[T any] struct {
	items     *[]T
	byID      map[string]T
	bodies    map[string]string
	summaries map[string]string
	idOf      func(T) string
}

func readKind[T any](standards fs.FS, kind string, sink kindSink[T]) error {
	paths, pathsErr := markdownPaths(standards, kind)
	if pathsErr != nil {
		return pathsErr
	}
	for _, p := range paths {
		item, body, err := readItem[T](standards, p)
		if err != nil {
			return err
		}
		id := sink.idOf(item)
		if id == "" {
			return fmt.Errorf("%s file %q has empty id", kind, p)
		}
		if _, dup := sink.byID[id]; dup {
			return fmt.Errorf("duplicate %s id %q", kind, id)
		}
		*sink.items = append(*sink.items, item)
		sink.byID[id] = item
		key := kind + ":" + id
		sink.bodies[key] = body
		sink.summaries[key] = firstBlockquote(body)
	}
	return nil
}

// readItem decodes one frontmatter-carrying markdown file into T plus its body.
func readItem[T any](standards fs.FS, path string) (T, string, error) {
	var item T

	raw, err := fs.ReadFile(standards, path)
	if err != nil {
		return item, "", fmt.Errorf("read %q: %w", path, err)
	}
	meta, body, err := splitFrontmatter(raw)
	if err != nil {
		return item, "", fmt.Errorf("frontmatter %q: %w", path, err)
	}
	if err = yaml.Unmarshal(meta, &item); err != nil {
		return item, "", fmt.Errorf("parse %q: %w", path, err)
	}
	return item, body, nil
}

// markdownPaths returns the sorted paths of all .md files (excluding README.md)
// under dir within fsys, including subdirectories. Paths are relative to the
// fsys root (e.g., "rules/style/effective-go.md").
func markdownPaths(fsys fs.FS, dir string) ([]string, error) {
	var paths []string
	err := fs.WalkDir(fsys, dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || d.Name() == "README.md" {
			return nil
		}
		paths = append(paths, p)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", dir, err)
	}
	sort.Strings(paths)
	return paths, nil
}

func readSkills(standards fs.FS, idx *Index) error {
	raw, err := fs.ReadFile(standards, "skills.yaml")
	if err != nil {
		return fmt.Errorf("read skills.yaml: %w", err)
	}
	var loaded skillsFile
	if err = yaml.Unmarshal(raw, &loaded); err != nil {
		return fmt.Errorf("parse skills.yaml: %w", err)
	}
	for _, skill := range loaded.Skills {
		if _, dup := idx.skillByID[skill.ID]; dup {
			return fmt.Errorf("duplicate skill id %q", skill.ID)
		}
		idx.skillByID[skill.ID] = skill
		idx.Skills = append(idx.Skills, skill)
	}
	return nil
}
