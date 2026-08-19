package report

import "fmt"

// Report collects a producer's progress notes and non-fatal warnings, so
// exactly one type flows from a producer (an install/uninstall wire
// function) through its installer to the printer that owns stdout/stderr. A
// producer decides for itself whether its work is note-worthy before
// returning, so a caller merges unconditionally instead of re-deriving
// "did something happen" from a bool, a slice length, or a status field
// scattered at the call site.
type Report struct {
	Notes    []string
	Warnings []string
}

// Note appends a formatted progress line for stdout.
func (r *Report) Note(format string, args ...any) {
	r.Notes = append(r.Notes, fmt.Sprintf(format, args...))
}

// Warn appends a formatted non-fatal notice for stderr.
func (r *Report) Warn(format string, args ...any) {
	r.Warnings = append(r.Warnings, fmt.Sprintf(format, args...))
}

// Merge appends sub's notes and warnings onto r, in order, so a caller can
// fold a collaborator's Report into its own without a separate "did it
// change anything" gate before deciding whether to print.
func (r *Report) Merge(sub Report) {
	r.Notes = append(r.Notes, sub.Notes...)
	r.Warnings = append(r.Warnings, sub.Warnings...)
}
