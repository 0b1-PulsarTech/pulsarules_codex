package report

import "fmt"

// Report collects a producer's progress notes and non-fatal warnings, so
// one type flows from a producer through its installer to the printer that
// owns stdout/stderr. A producer decides for itself whether its work is
// note-worthy, so a caller merges unconditionally instead of re-deriving
// that from a bool, slice length, or status field.
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
