package report

import "fmt"

type Report struct {
	Notes    []string
	Warnings []string
}

func (rep *Report) Note(format string, args ...any) {
	rep.Notes = append(rep.Notes, fmt.Sprintf(format, args...))
}

func (rep *Report) Warn(format string, args ...any) {
	rep.Warnings = append(rep.Warnings, fmt.Sprintf(format, args...))
}

func (rep *Report) Merge(sub Report) {
	rep.Notes = append(rep.Notes, sub.Notes...)
	rep.Warnings = append(rep.Warnings, sub.Warnings...)
}
