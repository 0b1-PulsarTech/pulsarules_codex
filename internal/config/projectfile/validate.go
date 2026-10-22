package projectfile

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/finding"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/config"
)

type validator struct {
	rules     Rules
	delegated map[string]bool
	root      *Field
	problems  *Problems
}

func (file *File) Validate(rules Rules) error {
	problems := &Problems{path: file.Path}
	file.check(rules, Root(), problems)
	return problems.Err()
}

func (file *File) check(rules Rules, root *Field, problems *Problems) {
	val := &validator{
		rules: rules, delegated: delegatedIDs(file.Delegated), root: root, problems: problems,
	}
	val.checkHeader(file)
	for _, dir := range file.SkipDirs {
		if dir == "" || strings.ContainsAny(dir, `/\`) {
			val.fail(
				val.root.Key("skip_dirs"), "%q is not a directory basename",
				ClipValue(dir),
			)
		}
	}
	globs := val.checkGlobs(val.root.Key("tests"), file.Tests)
	for idx, entry := range file.Overrides {
		field := val.root.Key("overrides").Item(idx).Key("files")
		globs = append(globs, val.checkGlobs(field, entry.Files)...)
	}
	inBudget := val.checkGlobBudget(globs)
	for _, id := range slices.Sorted(maps.Keys(file.Analyzers)) {
		entry := file.Analyzers[id]
		field := val.root.Key("analyzers").Key(id)
		val.checkAnalyzer(field, id)
		val.checkSeverity(field.Key("severity"), entry.Severity)
		val.checkAnalyzerParams(field.Key("params"), id, entry.Params)
	}
	if inBudget {
		val.checkOverrides(file.Overrides)
	}
	val.checkDelegated(file.Delegated)
}

func (val *validator) fail(at *Field, format string, args ...any) {
	if !at.covered(&val.problems.walked) {
		val.problems.add(0, at, format, args...)
	}
}

func (val *validator) checkHeader(file *File) {
	version := val.root.Key("version")
	switch {
	case file.Version == 0:
		val.fail(version, "required")
	case file.Version != SupportedVersion:
		val.fail(version, "unsupported %d (want %d)", file.Version, SupportedVersion)
	}
	if file.Preset != "" && !config.ValidPreset(file.Preset) {
		val.fail(val.root.Key("preset"), "%s",
			config.UnknownPreset("unknown preset", ClipValue(file.Preset)))
	}
}

func (val *validator) checkAnalyzer(field *Field, id string) {
	known := val.rules.AnyAnalyzer || slices.Contains(val.rules.AnalyzerIDs, id)
	if !known && !val.delegated[id] {
		val.fail(field, "unknown analyzer")
	}
}

func (val *validator) checkOverrideAnalyzer(field *Field, id string) {
	// A sub-ID such as golangci-lint/errcheck names one linter of a delegated analyzer.
	base, _, _ := strings.Cut(id, "/")
	val.checkAnalyzer(field, base)
}

func (val *validator) checkSeverity(field *Field, severity string) {
	if err := finding.ValidateSeverityName(severity); err != nil {
		val.fail(field, "invalid severity %q (want %s)",
			ClipValue(severity), strings.Join(finding.SeverityNames(), "|"))
	}
}

func (val *validator) checkParams(field *Field, id string, params map[string]any) {
	declared := val.rules.Params[id]
	for _, key := range slices.Sorted(maps.Keys(params)) {
		if !slices.Contains(declared, key) {
			val.fail(field.Key(key), "unknown param")
		}
	}
}

func (val *validator) checkDelegated(entries []DelegatedEntry) {
	seen := make(map[string]bool, len(entries))
	for idx, entry := range entries {
		field := val.root.Key("delegated").Item(idx)
		val.checkDelegatedID(field.Key("id"), entry.ID, seen)
		if len(entry.Run) == 0 {
			val.fail(field.Key("run"), "required")
		}
		if entry.Dir != "" && !filepath.IsLocal(entry.Dir) {
			val.fail(field.Key("dir"), "%q must be a relative path inside the project",
				ClipValue(entry.Dir))
		}
		if !slices.Contains(val.rules.DelegatedFormats, entry.Format) {
			val.fail(field.Key("format"), "unknown format %q (want %s)",
				ClipValue(entry.Format), strings.Join(val.rules.DelegatedFormats, "|"))
		}
	}
}

func (val *validator) checkDelegatedID(field *Field, id string, seen map[string]bool) {
	const maxDelegatedIDBytes = 64
	const idAlnum = "abcdefghijklmnopqrstuvwxyz0123456789"
	switch {
	case id == "":
		val.fail(field, "required")
	case slices.Contains(val.rules.ReservedIDs, id):
		val.fail(field, "%q is a built-in analyzer id", ClipValue(id))
	case len(id) > maxDelegatedIDBytes, strings.Trim(id, idAlnum+"_-") != "",
		!strings.Contains(idAlnum, id[:1]):
		val.fail(field, "%q is not a valid id (want [a-z0-9][a-z0-9_-]*, at most %d bytes)",
			ClipValue(id), maxDelegatedIDBytes)
	case seen[id]:
		val.fail(field, "duplicate id %q", ClipValue(id))
	}
	seen[id] = true
}

func delegatedIDs(entries []DelegatedEntry) map[string]bool {
	ids := make(map[string]bool, len(entries))
	for _, entry := range entries {
		ids[entry.ID] = entry.ID != ""
	}
	return ids
}
