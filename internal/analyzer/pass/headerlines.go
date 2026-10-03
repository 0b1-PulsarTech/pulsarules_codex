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

func IsExemptHeaderLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if isGeneratedMarkerLine(trimmed) {
		return true
	}
	return isSQLCAnnotation(trimmed) || slices.ContainsFunc(
		directiveHeaderPrefixes(),
		func(prefix string) bool { return strings.HasPrefix(trimmed, prefix) },
	)
}

func isSQLCAnnotation(trimmed string) bool {
	body, ok := strings.CutPrefix(trimmed, "-- name: ")
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
