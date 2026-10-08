package command

import (
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
)

type reports struct {
	id   string
	rule string
}

func (rep reports) named(sub string, severity finding.Severity) finding.Reporter {
	return finding.NewReporter(rep.id+"/"+sub, severity, finding.CatSyntax)
}

func (rep reports) plain(sub string, severity finding.Severity, message string) finding.Finding {
	return rep.named(sub, severity).New(message)
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
	message := reason
	if tail := stderrTail(stderr); tail != "" {
		message += ": " + tail
	}
	return rep.plain("failed", finding.SeverityError, message)
}

func (rep reports) diagnostic(item diagnostic, rel string) finding.Finding {
	code, suffix := item.code, ""
	if !validCode(code) {
		code = "diagnostic"
		if item.code != "" {
			suffix = " (code " + strconv.Quote(item.code) + ")"
		}
	}
	finding := rep.named(code, item.severity).At(
		rel, item.line, item.message+suffix, item.suggestion,
	)
	finding.RuleID = rep.rule
	return finding
}

func validCode(code string) bool {
	if code == "" {
		return false
	}
	return strings.IndexFunc(code, func(char rune) bool {
		return (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_'
	}) < 0
}

func stderrTail(stderr string) string {
	return strings.Map(shown, strings.ToValidUTF8(strings.TrimSpace(stderr), "?"))
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
