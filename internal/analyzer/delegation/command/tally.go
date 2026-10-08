package command

import (
	"fmt"
	"strconv"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type drops struct {
	count int
	worst core.Severity
	first diagnostic
}

func (lost *drops) note(item diagnostic) {
	if lost.count == 0 {
		lost.first, lost.worst = item, item.severity
	}
	lost.worst = max(lost.worst, item.severity)
	lost.count++
}

type sorted struct {
	findings []core.Finding
	excluded int
	lost     drops
}

func (where scope) sortOut(rep reports, found []diagnostic) sorted {
	memo := map[string]placed{}
	out := sorted{findings: make([]core.Finding, 0, len(found))}
	for _, item := range found {
		place, seen := memo[item.file]
		if !seen {
			place = where.place(item.file)
			memo[item.file] = place
		}
		switch place.outcome {
		case excluded:
			out.excluded++
		case dropped:
			out.lost.note(item)
		case kept:
			out.findings = append(out.findings, rep.diagnostic(item, place.rel))
		}
	}
	return out
}

func (rep reports) dropped(lost drops) core.Finding {
	code := lost.first.code
	if !validCode(code) {
		code = "diagnostic"
	}
	first := strconv.Quote(lost.first.file)
	if lost.first.line > 0 {
		first += ":" + strconv.Itoa(lost.first.line)
	}
	subject := "diagnostics on paths"
	if lost.count == 1 {
		subject = "diagnostic on a path"
	}
	message := fmt.Sprintf(
		"%d %s outside the project or missing (first: %s %s)", lost.count, subject, first, code,
	)
	return rep.plain("dropped", lost.worst, message)
}
