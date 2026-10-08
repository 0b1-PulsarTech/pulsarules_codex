package config

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core/pathglob"
)

type validator struct {
	rules     Rules
	delegated map[string]bool
	root      *fieldPath
	problems  *collector
}

func (file *ProjectFile) Validate(rules Rules) error {
	problems := &collector{path: file.Path}
	file.check(rules, newRoot(), problems)
	return problems.err()
}

func (file *ProjectFile) check(rules Rules, root *fieldPath, problems *collector) {
	val := &validator{
		rules: rules, delegated: delegatedIDs(file.Delegated), root: root, problems: problems,
	}
	val.checkHeader(file)
	for _, dir := range file.SkipDirs {
		if dir == "" || strings.ContainsAny(dir, `/\`) {
			val.fail(val.root.key("skip_dirs"), "%q is not a directory basename", clipValue(dir))
		}
	}
	globs := val.checkGlobs(val.root.key("tests"), file.Tests)
	for idx, entry := range file.Overrides {
		field := val.root.key("overrides").item(idx).key("files")
		globs = append(globs, val.checkGlobs(field, entry.Files)...)
	}
	inBudget := val.checkGlobBudget(globs)
	for _, id := range slices.Sorted(maps.Keys(file.Analyzers)) {
		entry := file.Analyzers[id]
		field := val.root.key("analyzers").key(id)
		val.checkAnalyzer(field, id)
		val.checkSeverity(field.key("severity"), entry.Severity)
		val.checkParams(field.key("params"), id, entry.Params)
	}
	if inBudget {
		val.checkOverrides(file.Overrides)
	}
	val.checkDelegated(file.Delegated)
}

func (val *validator) fail(at *fieldPath, format string, args ...any) {
	if !at.covered(&val.problems.walked) {
		val.problems.add(0, at, format, args...)
	}
}

func (val *validator) checkHeader(file *ProjectFile) {
	version := val.root.key("version")
	switch {
	case file.Version == 0:
		val.fail(version, "required")
	case file.Version != SupportedVersion:
		val.fail(version, "unsupported %d (want %d)", file.Version, SupportedVersion)
	}
	if file.Preset != "" && !ValidPreset(file.Preset) {
		val.fail(val.root.key("preset"), "unknown preset %q (want %s)",
			clipValue(file.Preset), strings.Join(Presets(), "|"))
	}
}

func (val *validator) checkAnalyzer(field *fieldPath, id string) {
	known := slices.Contains(val.rules.AnalyzerIDs, id)
	if !known && !val.delegated[id] {
		val.fail(field, "unknown analyzer")
	}
}

func (val *validator) checkOverrideAnalyzer(field *fieldPath, id string) {
	// A sub-ID such as golangci-lint/errcheck names one linter of a delegated analyzer.
	base, _, _ := strings.Cut(id, "/")
	val.checkAnalyzer(field, base)
}

func (val *validator) checkSeverity(field *fieldPath, severity string) {
	if severity != "" && !core.ValidSeverityName(severity) {
		val.fail(field, "invalid severity %q (want %s)",
			clipValue(severity), strings.Join(core.SeverityNames(), "|"))
	}
}

func (val *validator) checkParams(field *fieldPath, id string, params map[string]any) {
	declared := val.rules.Params[id]
	for _, key := range slices.Sorted(maps.Keys(params)) {
		if !slices.Contains(declared, key) {
			val.fail(field.key(key), "unknown param")
		}
	}
}

func (val *validator) checkDelegated(entries []DelegatedEntry) {
	seen := make(map[string]bool, len(entries))
	for idx, entry := range entries {
		field := val.root.key("delegated").item(idx)
		val.checkDelegatedID(field.key("id"), entry.ID, seen)
		if len(entry.Run) == 0 {
			val.fail(field.key("run"), "required")
		}
		if entry.Dir != "" && !filepath.IsLocal(entry.Dir) {
			val.fail(field.key("dir"), "%q must be a relative path inside the project",
				clipValue(entry.Dir))
		}
		if !slices.Contains(val.rules.DelegatedFormats, entry.Format) {
			val.fail(field.key("format"), "unknown format %q (want %s)",
				clipValue(entry.Format), strings.Join(val.rules.DelegatedFormats, "|"))
		}
	}
}

func (val *validator) checkDelegatedID(field *fieldPath, id string, seen map[string]bool) {
	const maxDelegatedIDBytes = 64
	const idAlnum = "abcdefghijklmnopqrstuvwxyz0123456789"
	switch {
	case id == "":
		val.fail(field, "required")
	case slices.Contains(val.rules.ReservedIDs, id):
		val.fail(field, "%q is a built-in analyzer id", clipValue(id))
	case len(id) > maxDelegatedIDBytes, strings.Trim(id, idAlnum+"_-") != "",
		!strings.Contains(idAlnum, id[:1]):
		val.fail(field, "%q is not a valid id (want [a-z0-9][a-z0-9_-]*, at most %d bytes)",
			clipValue(id), maxDelegatedIDBytes)
	case seen[id]:
		val.fail(field, "duplicate id %q", clipValue(id))
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

func (val *validator) checkGlobs(at *fieldPath, globs []string) []string {
	passed := make([]string, 0, len(globs))
	for idx, glob := range globs {
		if fault := globFault(glob); fault != "" {
			val.fail(at.item(idx), "%s", fault)
			continue
		}
		if _, err := pathglob.Compile(glob); err != nil {
			reason := strings.TrimPrefix(err.Error(), fmt.Sprintf("compile glob %q: ", glob))
			val.fail(at.item(idx), "compile glob %q: %s", clipValue(glob), clipValue(reason))
			continue
		}
		passed = append(passed, glob)
	}
	return passed
}

func globFault(glob string) string {
	switch {
	case len(glob) > MaxGlobBytes:
		return fmt.Sprintf("glob longer than %d bytes", MaxGlobBytes)
	case hasBrace(glob):
		return "brace alternatives are not supported; list each glob"
	case starCount(glob) > MaxGlobStars:
		return fmt.Sprintf(
			"glob has more than %d * (** and lone * segments are free)", MaxGlobStars,
		)
	}
	return ""
}
