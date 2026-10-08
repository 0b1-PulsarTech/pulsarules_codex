package projectcfg

import (
	"errors"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/text/escape"
)

func joinFaults(baseErr, privateErr error) error {
	// One budget for both files, and for the whole rendered line, not only the message.
	file := faultFile(baseErr, nil)
	first := baseErr.Error()
	text := first + "; " + privateErr.Error()
	if _, whole := config.Fit(text, maxJoinedReport-renderOverhead(file)); whole {
		return &FileError{File: file, Err: errors.New(escape.Line(text))}
	}
	short := config.ClipName(file, maxNameBytes)
	other := config.ClipName(faultFile(privateErr, nil), maxNameBytes)
	tail := "; " + escape.Line(other) + ": also invalid"
	room := maxJoinedReport - renderOverhead(short) - len(tail)
	first = strings.ReplaceAll(first, file, short)
	return &FileError{File: short, Err: errors.New(clipProblems(first, room, short) + tail)}
}

func clipProblems(raw string, room int, name string) string {
	if _, whole := config.Fit(raw, room); whole {
		return escape.Line(raw)
	}
	if cut := boundary(raw, room, name); cut > 0 {
		return escape.Line(raw[:cut])
	}
	return config.ClipHead(raw, room)
}
