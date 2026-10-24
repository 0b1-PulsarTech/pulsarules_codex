package applypatch

import (
	"path/filepath"
	"strings"
)

type Op string

const (
	Add      Op = "add"
	Update   Op = "update"
	Delete   Op = "delete"
	MoveFrom Op = "move_from"
	MoveTo   Op = "move_to"
)

type Edit struct {
	Path string
	Op   Op
}

func (op Op) PreEdit() bool { return op != Delete }

func (op Op) PostEdit() bool { return op != Delete && op != MoveFrom }

type editKey struct {
	path string
	op   Op
}

func Edits(patch, cwd string) []Edit {
	var edits []Edit
	seen := map[editKey]int{}
	// The index of an Update header the very next line may turn into a move source.
	pendingUpdate := -1
	for line := range strings.Lines(patch) {
		op, path := header(strings.TrimRight(line, "\r\n"))
		if path == "" {
			pendingUpdate = -1
			continue
		}
		// With no cwd a relative path stays relative: there is nothing to join it onto.
		if !filepath.IsAbs(path) && cwd != "" {
			path = filepath.Join(cwd, path)
		}
		if op == MoveTo && pendingUpdate >= 0 {
			edits[pendingUpdate].Op = MoveFrom
		}
		pendingUpdate = -1
		key := editKey{path, op}
		index, found := seen[key]
		if !found {
			index = len(edits)
			seen[key] = index
			edits = append(edits, Edit{Path: path, Op: op})
		}
		if op == Update {
			pendingUpdate = index
		}
	}
	return edits
}

func header(line string) (Op, string) {
	// Codex trims the file headers but matches the move line as written.
	if rest, found := strings.CutPrefix(line, "*** Move to:"); found {
		return MoveTo, strings.TrimSpace(rest)
	}
	trimmed := strings.TrimSpace(line)
	for _, candidate := range []struct {
		prefix string
		op     Op
	}{
		{"*** Add File:", Add}, {"*** Update File:", Update}, {"*** Delete File:", Delete},
	} {
		if rest, found := strings.CutPrefix(trimmed, candidate.prefix); found {
			return candidate.op, strings.TrimSpace(rest)
		}
	}
	return "", ""
}
