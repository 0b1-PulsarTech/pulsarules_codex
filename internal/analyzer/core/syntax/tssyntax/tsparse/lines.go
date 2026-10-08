package tsparse

import (
	"slices"
	"strings"
)

type Lines struct{ breaks []int }

func NewLines(text string) Lines {
	var breaks []int
	for offset := 0; ; {
		idx := strings.IndexByte(text[offset:], '\n')
		if idx < 0 {
			return Lines{breaks: breaks}
		}
		breaks = append(breaks, offset+idx)
		offset += idx + 1
	}
}

func (index Lines) Of(offset int) int {
	before, _ := slices.BinarySearch(index.breaks, offset)
	return before + 1
}
