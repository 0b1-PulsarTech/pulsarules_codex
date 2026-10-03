package pass

import (
	"slices"
	"strings"
)

func directiveHeaderPrefixes() []string {
	return []string{
		"//go:build", "// +build", "//go:generate", "//go:embed", "//nolint:",
		"//lint:ignore ", "//lint:file-ignore ",
		"/// <reference", "/// <amd-module", "/// <amd-dependency",
		"// @vitest-environment", "// @ts-self-types=",
		"/* istanbul ignore file", "/* c8 ignore", "/* v8 ignore file",
		"// coverage:ignore-file", "// ignore_for_file:", "// @dart=", "// @dart =",
		"// @ts-nocheck", "// @ts-check", "// @ts-ignore", "// @ts-expect-error",
		"// eslint-disable", "/* eslint", "// prettier-ignore",
		"// deno-lint-ignore-file", "// deno-fmt-ignore-file",
		"// protolint:disable", "// buf:lint:ignore",
		"<!-- svelte-ignore", "#!", "# -*- coding:", "# coding:",
	}
}

func migrationMarkerPrefixes() []string {
	return []string{"-- +goose ", "-- migrate:", "-- +migrate "}
}

func docblockPragmas() []string {
	return []string{
		"@jest-environment", "@vitest-environment", "@format", "@prettier",
		"@jsx", "@jsxImportSource", "@flow",
	}
}

func IsExemptHeaderLine(line string) bool {
	return IsGeneratedMarkerLine(line) || IsDirectiveHeaderLine(line)
}

func IsDirectiveHeaderLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return IsContentStartLine(trimmed) || hasAnyPrefix(trimmed, directiveHeaderPrefixes())
}

func IsContentStartLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return isSQLCAnnotation(trimmed) || hasAnyPrefix(trimmed, migrationMarkerPrefixes())
}

func hasAnyPrefix(text string, prefixes []string) bool {
	return slices.ContainsFunc(prefixes, func(prefix string) bool {
		return strings.HasPrefix(text, prefix)
	})
}

func ExemptHeaderSpan(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	first := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(first, "/*") {
		if IsDirectiveHeaderLine(first) {
			return 1
		}
		return 0
	}
	span := blockSpan(lines)
	if IsDirectiveHeaderLine(first) || hasDocblockPragma(lines[:span]) {
		return span
	}
	return 0
}

func blockSpan(lines []string) int {
	for idx, line := range lines {
		rest := line
		if idx == 0 {
			rest = strings.TrimSpace(line)[len("/*"):]
		}
		if strings.Contains(rest, "*/") {
			return idx + 1
		}
	}
	return len(lines)
}

func hasDocblockPragma(block []string) bool {
	return slices.ContainsFunc(block, func(line string) bool {
		fields := strings.Fields(strings.TrimLeft(strings.TrimSpace(line), "/* \t"))
		return len(fields) > 0 && slices.Contains(docblockPragmas(), fields[0])
	})
}

func isSQLCAnnotation(trimmed string) bool {
	body, ok := sqlcBody(trimmed)
	if !ok {
		return false
	}
	// The sqlc tool splits on single spaces, so repeated spaces are not an annotation.
	fields := strings.Split(body, " ")
	if len(fields) != 2 {
		return false
	}
	command, ok := strings.CutPrefix(fields[1], ":")
	if !ok {
		return false
	}
	switch command {
	case "one", "many", "exec", "execrows", "execresult", "execlastid",
		"copyfrom", "batchexec", "batchmany", "batchone":
		return true
	}
	return false
}

func sqlcBody(trimmed string) (string, bool) {
	if body, ok := strings.CutPrefix(trimmed, "-- name: "); ok {
		return body, true
	}
	block, ok := strings.CutPrefix(trimmed, "/* name: ")
	if !ok {
		return "", false
	}
	return strings.CutSuffix(block, " */")
}
