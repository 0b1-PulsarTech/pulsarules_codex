package knowledge

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

func ruleID(rule Rule) string             { return rule.ID }
func patternID(pattern Pattern) string    { return pattern.ID }
func workflowID(workflow Workflow) string { return workflow.ID }

type kindSink[Item any] struct {
	items  *[]Item
	byID   map[string]Item
	bodies map[string]string
	idOf   func(Item) string
}

func readKind[Item any](standards fs.FS, kind string, sink kindSink[Item]) error {
	paths, pathsErr := markdownPaths(standards, kind)
	if pathsErr != nil {
		return pathsErr
	}
	for _, path := range paths {
		item, body, err := readItem[Item](standards, path)
		if err != nil {
			return err
		}
		id := sink.idOf(item)
		if id == "" {
			return fmt.Errorf("%s file %q has empty id", kind, path)
		}
		if _, dup := sink.byID[id]; dup {
			return fmt.Errorf("duplicate %s id %q", kind, id)
		}
		*sink.items = append(*sink.items, item)
		sink.byID[id] = item
		sink.bodies[kind+":"+id] = body
	}
	return nil
}

func readItem[Item any](standards fs.FS, path string) (Item, string, error) {
	var item Item

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

func markdownPaths(fsys fs.FS, dir string) ([]string, error) {
	var paths []string
	err := fs.WalkDir(fsys, dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") ||
			entry.Name() == "README.md" {
			return nil
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %q: %w", dir, err)
	}
	sort.Strings(paths)
	return paths, nil
}
