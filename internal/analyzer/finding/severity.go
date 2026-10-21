package finding

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

type Category int

const (
	CatSyntax Category = iota
	CatAST
	CatArch
	CatCommit
	CatProject
)

var ErrUnknownSeverity = errors.New("unknown severity")

func (severity Severity) String() string {
	switch severity {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "info"
	}
}

func SeverityNames() []string {
	names := []string{SeverityInfo.String(), SeverityWarning.String(), SeverityError.String()}
	slices.Sort(names)
	return names
}

func ParseSeverity(name string) (Severity, error) {
	for _, severity := range []Severity{SeverityInfo, SeverityWarning, SeverityError} {
		if severity.String() == name {
			return severity, nil
		}
	}
	return SeverityInfo, fmt.Errorf(
		"%w %q (want %s)", ErrUnknownSeverity, name, strings.Join(SeverityNames(), "|"),
	)
}

func ValidateSeverityName(name string) error {
	if name == "" {
		return nil
	}
	_, err := ParseSeverity(name)
	return err
}
