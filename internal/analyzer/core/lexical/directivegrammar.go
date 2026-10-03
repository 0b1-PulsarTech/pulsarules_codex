package lexical

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
