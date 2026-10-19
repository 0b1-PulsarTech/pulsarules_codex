package command

import (
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

const (
	maxLineBytes = 4096
	maxCodeBytes = 64
)

func quotedSize(unit string) int { return len(strconv.Quote(unit)) - len(`""`) }

func lineRoom(id string) int {
	const lineFrameBytes = 14

	return maxLineBytes - lineFrameBytes - len(printable.Line(id))
}

func codeSuffix(code string) string {
	quoted := strconv.Quote(code)
	if len(quoted) > maxCodeBytes {
		quoted = strconv.Quote(code[:printable.HeadEnd(code, maxCodeBytes-len(`""`), quotedSize)])
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
