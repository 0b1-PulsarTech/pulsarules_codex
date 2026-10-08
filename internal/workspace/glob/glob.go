package glob

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

const anySegments = "**"

type Pattern struct {
	segments []string
	required int
}

func Compile(pattern string) (Pattern, error) {
	trimmed := strings.TrimPrefix(pattern, "./")
	if trimmed == "" {
		return Pattern{}, fmt.Errorf("compile glob %q: empty pattern", pattern)
	}
	segments := strings.Split(trimmed, "/")
	if slices.Contains(segments, "") {
		return Pattern{}, fmt.Errorf("compile glob %q: empty path segment", pattern)
	}
	for idx, segment := range segments {
		segments[idx] = negateClasses(segment)
		if _, err := path.Match(segments[idx], ""); err != nil {
			return Pattern{}, fmt.Errorf("compile glob %q: %w", pattern, err)
		}
	}
	collapsed := slices.CompactFunc(segments, func(left, right string) bool {
		return left == anySegments && right == anySegments
	})
	required := 0
	for _, segment := range collapsed {
		if segment != anySegments {
			required++
		}
	}
	return Pattern{segments: collapsed, required: required}, nil
}

func negateClasses(segment string) string {
	var out strings.Builder
	classStart := -1
	for idx := 0; idx < len(segment); idx++ {
		char := segment[idx]
		switch {
		case char == '\\' && idx+1 < len(segment):
			out.WriteByte(char)
			idx++
			char = segment[idx]
		case classStart < 0:
			if char == '[' {
				classStart = idx
			}
		case idx == classStart+1 && char == '!':
			char = '^'
		case char == ']':
			classStart = -1
		}
		out.WriteByte(char)
	}
	return out.String()
}

func (pattern Pattern) Match(rel string) bool {
	trimmed := strings.TrimPrefix(filepath.ToSlash(rel), "./")
	names := strings.Split(trimmed, "/")
	// Every non-** segment consumes exactly one name, so fewer names can never match.
	if len(names) < pattern.required {
		return false
	}
	memo := make([][]int8, len(pattern.segments)+1)
	for idx := range memo {
		memo[idx] = make([]int8, len(names)+1)
	}
	match := globMatch{patterns: pattern.segments, names: names, memo: memo}
	return match.segments(0, 0)
}

type globMatch struct {
	patterns, names []string
	memo            [][]int8
}

func (match globMatch) segments(patIdx, nameIdx int) bool {
	const (
		unknown = iota
		matched
		failed
	)
	if state := match.memo[patIdx][nameIdx]; state != unknown {
		return state == matched
	}
	result := match.from(patIdx, nameIdx)
	match.memo[patIdx][nameIdx] = failed
	if result {
		match.memo[patIdx][nameIdx] = matched
	}
	return result
}

func (match globMatch) from(patIdx, nameIdx int) bool {
	if patIdx == len(match.patterns) {
		return nameIdx == len(match.names)
	}
	if match.patterns[patIdx] == anySegments {
		for skip := nameIdx; skip <= len(match.names); skip++ {
			if match.segments(patIdx+1, skip) {
				return true
			}
		}
		return false
	}
	if nameIdx == len(match.names) {
		return false
	}
	ok, err := path.Match(match.patterns[patIdx], match.names[nameIdx])
	return err == nil && ok && match.segments(patIdx+1, nameIdx+1)
}
