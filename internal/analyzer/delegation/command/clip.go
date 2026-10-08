package command

import (
	"strconv"
	"unicode/utf8"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

const (
	maxLineBytes = 4096
	maxCodeBytes = 64
)

func escapedSize(unit string) int { return len(printable.Line(unit)) }

func quotedSize(unit string) int { return len(strconv.Quote(unit)) - len(`""`) }

func unitWidth(text string) int {
	_, width := utf8.DecodeRuneInString(text)
	return width
}

func headEnd(text string, limit int, size func(string) int) int {
	used := 0
	pos := 0
	for pos < len(text) {
		width := unitWidth(text[pos:])
		used += size(text[pos : pos+width])
		if used > limit {
			break
		}
		pos += width
	}
	return pos
}

func tailStart(text string, limit int, size func(string) int) int {
	used := 0
	pos := len(text)
	for pos > 0 {
		_, width := utf8.DecodeLastRuneInString(text[:pos])
		used += size(text[pos-width : pos])
		if used > max(limit, 0) {
			break
		}
		pos -= width
	}
	return pos
}

func fitHead(text string, limit int) string {
	if escapedSize(text) <= limit {
		return text
	}
	return printable.Line(text[:headEnd(text, max(limit, 0), escapedSize)])
}

func fitTail(text string, limit int) string {
	if escapedSize(text) <= limit {
		return text
	}
	return printable.Line(text[tailStart(text, limit, escapedSize):])
}

func lineRoom(id string) int {
	const lineFrameBytes = 14

	return maxLineBytes - lineFrameBytes - len(printable.Line(id))
}

func codeSuffix(code string) string {
	quoted := strconv.Quote(code)
	if len(quoted) > maxCodeBytes {
		quoted = strconv.Quote(code[:headEnd(code, maxCodeBytes-len(`""`), quotedSize)])
	}
	return " (code " + quoted + ")"
}

func locationRoom(file string, line int) (room int, keep bool) {
	const maxLocationBytes = 512

	if file == "" {
		return 0, true
	}
	size := len(printable.Path(file))
	frame := len(" (:)")
	if line > 0 {
		frame += len(strconv.Itoa(line))
	}
	return size + frame, size <= maxLocationBytes
}
