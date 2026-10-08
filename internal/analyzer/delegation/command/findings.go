package command

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type reports struct {
	id   string
	rule string
}

func (rep reports) named(sub string, severity finding.Severity) finding.Reporter {
	return finding.NewReporter(rep.id+"/"+sub, severity, finding.CatSyntax)
}

func (rep reports) plain(sub string, severity finding.Severity, message string) finding.Finding {
	room := lineRoom(rep.id + "/" + sub)
	return rep.named(sub, severity).New(fitHead(message, room))
}

func (rep reports) config(message string) finding.Finding {
	return rep.plain("config", finding.SeverityError, message)
}

func (rep reports) unavailable(name string) finding.Finding {
	return rep.plain(
		"unavailable", finding.SeverityInfo, name+" not found on PATH; "+rep.id+" was skipped",
	)
}

func (rep reports) timeout(name string) finding.Finding {
	message := name + " timed out after " + commandTimeout.String()
	return rep.plain("timeout", finding.SeverityError, message)
}

func (rep reports) failed(reason, stderr string) finding.Finding {
	const maxReasonBytes = 512

	reason = fitHead(reason, maxReasonBytes)
	message := reason
	room := lineRoom(rep.id+"/failed") - len(printable.Line(reason)) - len(": ")
	if tail := stderrTail(stderr, room); tail != "" {
		message += ": " + tail
	}
	return rep.plain("failed", finding.SeverityError, message)
}

func (rep reports) diagnostic(item diagnostic, rel string) finding.Finding {
	const arrowFrameBytes = 8

	code, suffix := item.code, ""
	if !validCode(code) {
		code = "diagnostic"
		if item.code != "" {
			suffix = codeSuffix(item.code)
		}
	}
	line := item.line
	room, keep := locationRoom(rel, line)
	if !keep {
		rel, line, room = "", 0, 0
	}
	room = lineRoom(rep.id+"/"+code) - room - len(suffix)
	finding := rep.named(code, item.severity).At(
		rel, line, fitHead(item.message, room)+suffix,
		fitHead(item.suggestion, maxLineBytes-arrowFrameBytes),
	)
	finding.RuleID = rep.rule
	return finding
}

func validCode(code string) bool {
	if code == "" || len(code) > maxCodeBytes {
		return false
	}
	return strings.IndexFunc(code, func(char rune) bool {
		return (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_'
	}) < 0
}

func stderrTail(stderr string, limit int) string {
	text := strings.Map(shown, strings.ToValidUTF8(strings.TrimSpace(stderr), "?"))
	return fitTail(text, limit)
}

func shown(char rune) rune {
	if unicode.IsControl(char) && char != '\n' && char != '\t' {
		return ' '
	}
	return char
}

func relativeTo(root, base, file string) (string, bool) {
	if !filepath.IsAbs(file) {
		file = filepath.Join(base, file)
	}
	rel, err := filepath.Rel(root, file)
	if err != nil || !filepath.IsLocal(rel) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}
