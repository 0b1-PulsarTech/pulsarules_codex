package lexer

import (
	"slices"
	"strings"
)

func isNolintDirective(trimmed string) bool {
	rest, ok := strings.CutPrefix(trimmed, "//nolint")
	if !ok {
		return false
	}
	if list, hasList := strings.CutPrefix(rest, ":"); hasList {
		rest, ok = skipLinterList(list)
		if !ok {
			return false
		}
	}
	rest = strings.TrimLeft(rest, " \t")
	return rest == "" || strings.HasPrefix(rest, "//")
}

func skipLinterList(list string) (string, bool) {
	for {
		list = strings.TrimLeft(list, " \t")
		end := strings.IndexFunc(list, func(char rune) bool {
			return !isWordChar(char) && char != '-'
		})
		if end < 0 {
			end = len(list)
		}
		if end == 0 {
			return "", false
		}
		list = strings.TrimLeft(list[end:], " \t")
		next, more := strings.CutPrefix(list, ",")
		if !more {
			return list, true
		}
		list = next
	}
}

func isGlobalDirective(trimmed string) bool {
	for _, opener := range []string{"/* global ", "/* globals ", "/*global ", "/*globals "} {
		list, ok := strings.CutPrefix(trimmed, opener)
		if !ok {
			continue
		}
		list = strings.TrimSuffix(strings.TrimSpace(list), "*/")
		items := strings.FieldsFunc(list, func(char rune) bool {
			return char == ',' || char == ' ' || char == '\t'
		})
		return len(items) > 0 && !slices.ContainsFunc(items, func(item string) bool {
			name, _, _ := strings.Cut(item, ":")
			return name == "" || strings.IndexFunc(name, func(char rune) bool {
				return !isWordChar(char) && char != '$'
			}) >= 0
		})
	}
	return false
}

func isDartFormatDirective(trimmed string) bool {
	option, ok := strings.CutPrefix(trimmed, "// dart format ")
	if !ok {
		return false
	}
	width, isWidth := strings.CutPrefix(option, "width=")
	if isWidth {
		return width != "" && strings.Trim(width, "0123456789") == ""
	}
	return option == "off" || option == "on"
}

func isAtlasDirective(trimmed string) bool {
	word, ok := strings.CutPrefix(trimmed, "-- atlas:")
	return ok && word != "" && word[0] != ' ' && word[0] != '\t'
}

func isWordChar(char rune) bool {
	return char == '_' || char >= '0' && char <= '9' || char >= 'a' && char <= 'z' ||
		char >= 'A' && char <= 'Z'
}

func PulsarulesMarker(body string) (string, bool) {
	token, ok := strings.CutPrefix(strings.TrimSpace(body), "pulsarules:")
	if !ok || token == "" || token[0] == ' ' || token[0] == '\t' {
		return "", false
	}
	return token, true
}

type Dialect string

const (
	DialectGo Dialect = "go"
	DialectTS Dialect = "ts"
)

type dialectSpec struct {
	syntax       CommentSyntax
	linePrefixes []string
	bodyPrefixes []string
	header       bool
	deprecated   []string
	// toolchainText: the toolchain (go/ast CommentGroup.Text) decides what counts as doc text.
	toolchainText bool
}

func dialectSpecs() map[Dialect]dialectSpec {
	slashes := CommentSyntax{
		Line:   []string{"//"},
		Blocks: []BlockDelim{{Open: "/*", Close: "*/"}},
	}
	goSyntax := slashes
	goSyntax.keepStars = true
	return map[Dialect]dialectSpec{
		DialectGo: {
			syntax: goSyntax,
			linePrefixes: []string{
				"//go:", "//nolint", "//lint:",
				"//goverter:", "// goverter:", "//pulsarules:", "// pulsarules:",
				"//line ", "//extern ", "//export ",
			},
			header:        true,
			deprecated:    []string{"Deprecated:"},
			toolchainText: true,
		},
		DialectTS: {
			syntax: slashes,
			bodyPrefixes: []string{
				"eslint-", "@ts-", "pulsarules:", "ignore:", "ignore_for_file:",
				"prettier-ignore", "istanbul ignore", "c8 ignore", "svelte-ignore",
				"biome-ignore", "noinspection",
			},
			deprecated: []string{"@deprecated", "Deprecated:"},
		},
	}
}

func (dialect Dialect) Syntax() CommentSyntax { return dialectSpecs()[dialect].syntax }

func (dialect Dialect) ToolchainText() bool { return dialectSpecs()[dialect].toolchainText }

func (dialect Dialect) IsDeprecatedMarker(body string) bool {
	return hasAnyPrefix(body, dialectSpecs()[dialect].deprecated)
}

func IsDirectiveLine(dialect Dialect, line string) bool {
	spec := dialectSpecs()[dialect]
	trimmed := strings.TrimSpace(line)
	return spec.header && IsExemptHeaderLine(trimmed) ||
		hasAnyPrefix(trimmed, spec.linePrefixes) ||
		IsDirectiveBody(dialect, CommentBody(trimmed, spec.syntax))
}

func IsDirectiveBody(dialect Dialect, body string) bool {
	return hasAnyPrefix(body, dialectSpecs()[dialect].bodyPrefixes)
}
