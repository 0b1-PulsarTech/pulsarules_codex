package printable

import (
	"strconv"
	"strings"
)

type LineList struct {
	lines []int
}

func (list *LineList) Add(line int) {
	// A run of hits on one line lists it once.
	if len(list.lines) > 0 && list.lines[len(list.lines)-1] == line {
		return
	}
	list.lines = append(list.lines, line)
}

func (list LineList) Len() int { return len(list.lines) }

func (list LineList) Noun() string {
	if len(list.lines) > 1 {
		return "lines"
	}
	return "line"
}

func (list LineList) Extra(limit int) int { return max(len(list.lines)-limit, 0) }

func (list LineList) Head(limit int) string {
	shown := make([]string, 0, min(len(list.lines), limit))
	for _, line := range list.lines[:min(len(list.lines), limit)] {
		shown = append(shown, strconv.Itoa(line))
	}
	return strings.Join(shown, ", ")
}
