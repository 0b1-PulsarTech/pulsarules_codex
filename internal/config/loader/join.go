package loader

import (
	"errors"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

func joinFaults(baseErr, privateErr error) error {
	// One budget for both files, and for the whole rendered line, not only the message.
	file := faultFile(baseErr, nil)
	first := baseErr.Error()
	text := first + "; " + privateErr.Error()
	if _, whole := printable.Fit(text, maxJoinedReport-renderOverhead(file)); whole {
		return &fileError{File: file, Err: errors.New(printable.Line(text))}
	}
	short := printable.ShortenPath(file, maxNameBytes)
	other := printable.ShortenPath(faultFile(privateErr, nil), maxNameBytes)
	tail := "; " + printable.Line(other) + ": also invalid"
	room := maxJoinedReport - renderOverhead(short) - len(tail)
	first = strings.ReplaceAll(first, file, short)
	return &fileError{File: short, Err: errors.New(clipProblems(first, room, short) + tail)}
}

func clipProblems(raw string, room int, name string) string {
	if _, whole := printable.Fit(raw, room); whole {
		return printable.Line(raw)
	}
	if cut := boundary(raw, room, name); cut > 0 {
		return printable.Line(raw[:cut])
	}
	return printable.Shorten(raw, room)
}
