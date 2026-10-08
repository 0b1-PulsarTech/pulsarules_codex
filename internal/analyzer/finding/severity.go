package finding

import (
	"fmt"
	"maps"
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

type StageID int

const (
	StageContext StageID = iota
	StageStatic
	StageAST
	StageArch
	StageRuleInjection
	StageOutput
)

func SeverityByName() map[string]Severity {
	return map[string]Severity{
		"error":   SeverityError,
		"warning": SeverityWarning,
		"info":    SeverityInfo,
	}
}

func SeverityNames() []string {
	return slices.Sorted(maps.Keys(SeverityByName()))
}

func ValidateSeverityName(name string) error {
	if name == "" || ValidSeverityName(name) {
		return nil
	}
	return fmt.Errorf(
		"unknown severity %q (want %s)", name, strings.Join(SeverityNames(), "|"),
	)
}

func ValidSeverityName(name string) bool {
	_, ok := SeverityByName()[name]
	return ok
}
