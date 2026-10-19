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

func (list LineList) Extra(limit int) int { return max(len(list.lines)-limit, 0) }

func (list LineList) Label(limit int) string {
	noun := "line"
	if len(list.lines) > 1 {
		noun = "lines"
	}
	shown := make([]string, 0, min(len(list.lines), limit))
	for _, line := range list.lines[:min(len(list.lines), limit)] {
		shown = append(shown, strconv.Itoa(line))
	}
	return noun + " " + strings.Join(shown, ", ")
}
