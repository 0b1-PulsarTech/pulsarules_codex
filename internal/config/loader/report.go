package loader

import (
	"errors"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config/projectfile"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

const (
	maxJoinedReport = 4096
	maxNameBytes    = 1024
	moreProblems    = "; ... and more problems"
)

func renderOverhead(file string) int {
	const indent, label, location = 2, len("[error] "), len(" ()\n")
	// The widest finding layout: indent, "[error] config: ", message, " (file)" and newline.
	return indent + label + len(ConfigAnalyzerID) + len(": ") + location + len(printable.Path(file))
}

func capReport(err error, file *projectfile.File) error {
	if err == nil {
		return nil
	}
	name := faultFile(err, file)
	if _, whole := printable.Fit(err.Error(), maxJoinedReport-renderOverhead(name)); whole {
		return err
	}
	// The renderer escapes and adds the file once more, so only the final line is measured.
	// Two copies of the name, the line overhead and one problem always fit under the cap.
	short := printable.ShortenPath(name, maxNameBytes)
	text, leads := err.Error(), []string{short}
	if display := leadOf(err, file); display != "" && display != name {
		shown := printable.ShortenPath(display, maxNameBytes)
		text = strings.ReplaceAll(text, display, shown)
		leads = []string{shown}
	}
	text = strings.ReplaceAll(text, name, short)
	budget := maxJoinedReport - renderOverhead(short)
	if _, whole := printable.Fit(text, budget); whole {
		return &fileError{File: short, Err: errors.New(printable.Line(text))}
	}
	if cut := boundary(text, budget-len(moreProblems), leads...); cut > 0 {
		return &fileError{File: short, Err: errors.New(printable.Line(text[:cut]) + moreProblems)}
	}
	return &fileError{File: short, Err: errors.New(printable.Shorten(text, budget))}
}

func boundary(raw string, room int, names ...string) int {
	// The last lead of a following problem that still fits, so a cut keeps whole problems.
	kept, _ := printable.Fit(raw, room)
	cut := 0
	for _, name := range names {
		lead := "; " + name + ": "
		window := raw[:min(len(raw), len(kept)+len(lead)-1)]
		cut = max(cut, strings.LastIndex(window, lead))
	}
	return cut
}
