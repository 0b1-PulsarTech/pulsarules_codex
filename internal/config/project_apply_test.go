package config

import (
	"slices"
	"strings"
	"testing"
)

func globOverrides(sizes ...int) string {
	entries := make([]string, len(sizes))
	for idx, size := range sizes {
		entries[idx] = "{files: [" + strings.Repeat("a, ", size) + "]}"
	}
	return "version: 1\noverrides: [" + strings.Join(entries, ", ") + "]\n"
}

func TestValidateOverrides(t *testing.T) {
	t.Parallel()

	const head = "version: 1\noverrides: "
	const tooMany = "overrides and tests: more than 256 globs in total"
	many := make([]int, 600)
	for idx := range many {
		many[idx] = 2
	}
	runValidateCases(t, []validateCase{
		{
			"without files",
			head + "[{analyzers: {file-size: {enabled: false}}}]\n",
			"overrides[0].files: at least one glob",
		},
		{"bad glob", head + "[{files: [\"\"], analyzers: {}}]\n", "overrides[0].files[0]:"},
		{
			"unknown analyzer",
			head + "[{files: [a], analyzers: {nope: {enabled: false}}}]\n",
			"overrides[0].analyzers.nope: unknown analyzer",
		},
		{
			"bad severity",
			head + "[{files: [a], analyzers: {file-size: {severity: x}}}]\n",
			"overrides[0].analyzers.file-size.severity: invalid severity",
		},
		{
			"params",
			head + "[{files: [a], analyzers: {file-size: {params: {max_lines: 1}}}}]\n",
			"overrides[0].analyzers.file-size.params: per-path params are not supported",
		},
		{
			"sub id",
			head + "[{files: [a], analyzers: {golangci-lint/errcheck: {enabled: false}}}]\n",
			"",
		},
		{"exactly 256 globs in total", globOverrides(128, 128), ""},
		{"257 globs in total", globOverrides(128, 129), tooMany},
		{"two lists of 200 and 57 globs", globOverrides(200, 57), tooMany},
		{"two globs in each of 600 overrides", globOverrides(many...), tooMany},
	})
}

func globList(count, size int) string {
	items := make([]string, count)
	for idx := range items {
		items[idx] = strings.Repeat("a", size)
	}
	return "[" + strings.Join(items, ", ") + "]"
}

func TestValidateGlobBudget(t *testing.T) {
	t.Parallel()

	const (
		tooMany  = "overrides and tests: more than 256 globs in total"
		tooBytes = "overrides and tests: globs exceed 4096 bytes in total"
	)
	aliased := func(count int) string {
		return "version: 1\noverrides:\n  - files: [&g a" + strings.Repeat(", *g", count-1) + "]\n"
	}
	both := func(tests, overrides string) string {
		return "version: 1\ntests: " + tests + "\noverrides: [{files: " + overrides + "}]\n"
	}
	runValidateCases(t, []validateCase{
		{"tests and overrides at 256 globs", both(globList(128, 1), globList(128, 1)), ""},
		{"tests count toward the 256 globs", both(globList(128, 1), globList(129, 1)), tooMany},
		{"tests alone over 256 globs", "version: 1\ntests: " + globList(257, 1) + "\n", tooMany},
		{"256 aliased globs", aliased(256), ""},
		{"257 aliased globs", aliased(257), tooMany},
		{"overrides at 4096 bytes", both("[]", globList(16, 256)), ""},
		{"overrides at 4097 bytes", both("[a]", globList(16, 256)), tooBytes},
		{"tests and overrides at 4096 bytes", both(globList(8, 256), globList(8, 256)), ""},
		{"tests count toward the 4096 bytes", both(globList(8, 256), globList(9, 256)), tooBytes},
		{
			"aliased globs count their expanded bytes",
			"version: 1\noverrides:\n  - files: [&g " + strings.Repeat("a", 256) +
				strings.Repeat(", *g", 16) + "]\n",
			tooBytes,
		},
	})
}

func TestValidateGlobBytes(t *testing.T) {
	t.Parallel()

	const head = "version: 1\noverrides: "
	runValidateCases(t, []validateCase{
		{"glob of 256 bytes", head + "[{files: [" + strings.Repeat("a", 256) + "]}]\n", ""},
		{
			"glob of 257 bytes",
			head + "[{files: [" + strings.Repeat("a", 257) + "]}]\n",
			"overrides[0].files[0]: glob longer than 256 bytes",
		},
		{
			"second glob of 257 bytes",
			head + "[{files: [a, " + strings.Repeat("a", 257) + "]}]\n",
			"overrides[0].files[1]: glob longer than 256 bytes",
		},
		{
			"tests glob of 257 bytes",
			"version: 1\ntests: [" + strings.Repeat("a", 257) + "]\n",
			"tests[0]: glob longer than 256 bytes",
		},
	})
}

func TestValidateGlobTotalStopsFurtherChecks(t *testing.T) {
	t.Parallel()

	content := "version: 1\ntests: [a]\noverrides: [{files: [" + strings.Repeat("a, ", 200) +
		"]}, {files: [" + strings.Repeat("a, ", 57) +
		"], analyzers: {file-size: {severity: x}}}]\n"
	file, err := decodeFile([]byte(content))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	err = file.Validate(testRules())
	want := ProjectFileName + ": overrides and tests: more than 256 globs in total"
	if err == nil || err.Error() != want {
		t.Fatalf("Validate() error = %v, want only %q", err, want)
	}
}

func TestValidateWildcardBudget(t *testing.T) {
	t.Parallel()

	const tooWild = "overrides and tests: wildcard classes and ? exceed 256 in total"
	class := func(size int) string { return "[" + strings.Repeat("a", size-2) + "]" }
	marks := func(count int) string { return strings.Repeat("?", count) }
	list := func(globs ...string) string { return "['" + strings.Join(globs, "', '") + "']" }
	both := func(tests, overrides string) string {
		return "version: 1\ntests: " + tests + "\noverrides: [{files: " + overrides + "}]\n"
	}
	aliased := func(copies int) string {
		return "version: 1\noverrides:\n  - files: [&g '" + class(100) +
			"'" + strings.Repeat(", *g", copies-1) + "]\n"
	}
	escaped, inEscape := strings.Repeat(`\[\?`, 64), `[\]a]`
	afterClass := "[a]" + strings.Repeat("b", 250)
	runValidateCases(t, []validateCase{
		{"one class of 256 bytes", both("[]", list(class(256))), ""},
		{"a class of 256 bytes and one ?", both("[]", list(class(256), "?")), tooWild},
		{"256 question marks", both(list(marks(128)), list(marks(128))), ""},
		{"257 question marks", both(list(marks(128)), list(marks(129))), tooWild},
		{"tests count toward the classes", both(list(class(128)), list(class(129))), tooWild},
		{"tests and overrides at 256 bytes", both(list(class(128)), list(class(128))), ""},
		{"literals after a class cost nothing", both("[]", list(afterClass, class(250))), ""},
		{"aliased classes at 200 bytes", aliased(2), ""},
		{"aliased classes count after expansion", aliased(3), tooWild},
		{"escaped brackets and marks cost nothing", both("[]", list(escaped)), ""},
		{"an escape inside a class costs both", both("[]", list(inEscape, marks(252))), tooWild},
		{"five bytes of escaped class leave 251", both("[]", list(inEscape, marks(251))), ""},
	})
}

func TestValidateStarTailBudget(t *testing.T) {
	t.Parallel()

	const tooLong = "overrides and tests: text after the first * of each segment " +
		"exceeds 2048 bytes in total"
	tail := func(size int) string { return "*" + strings.Repeat("b", size) }
	list := func(globs ...string) string { return "['" + strings.Join(globs, "', '") + "']" }
	repeat := func(glob string, count int, more ...string) string {
		return list(append(slices.Repeat([]string{glob}, count), more...)...)
	}
	both := func(tests, overrides string) string {
		return "version: 1\ntests: " + tests + "\noverrides: [{files: " + overrides + "}]\n"
	}
	aliased := func(copies int) string {
		return "version: 1\noverrides:\n  - files: [&g '" + tail(255) +
			"'" + strings.Repeat(", *g", copies-1) + "]\n"
	}
	half := func(extra int) string { return repeat(tail(255), 4, tail(extra)) }
	segments := tail(80) + "/" + tail(80) + "/" + tail(80)
	twoStars := tail(100) + tail(100)
	runValidateCases(t, []validateCase{
		{"2048 bytes after a star", both("[]", repeat(tail(255), 8, tail(8))), ""},
		{"2049 bytes after a star", both("[]", repeat(tail(255), 8, tail(9))), tooLong},
		{"tests count toward the tails", both(half(5), half(5)), tooLong},
		{"tests and overrides at 2048 bytes", both(half(4), half(4)), ""},
		{"aliased tails at 2040 bytes", aliased(8), ""},
		{"aliased tails count after expansion", aliased(9), tooLong},
		{"double stars cost nothing", both("[]", list(strings.Repeat("**/", 80)+"*")), ""},
		{"eight globs of three segment tails", both("[]", repeat(segments, 8)), ""},
		{"nine globs of three segment tails", both("[]", repeat(segments, 9)), tooLong},
		{"a second star is not a tail byte", both("[]", repeat(twoStars, 10, tail(45))), ""},
		{"eleven globs with two stars", both("[]", repeat(twoStars, 11)), tooLong},
	})
}

func TestValidateRejectedInvalidGlobIsOneProblem(t *testing.T) {
	t.Parallel()

	invalid := "a//" + strings.Repeat("b", 254)
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			"in tests",
			"version: 1\ntests: ['" + invalid + "']\n",
			ProjectFileName + ": tests[0]: glob longer than 256 bytes",
		},
		{
			"in an override",
			"version: 1\noverrides: [{files: ['" + invalid + "']}]\n",
			ProjectFileName + ": overrides[0].files[0]: glob longer than 256 bytes",
		},
		{
			"with too many stars",
			"version: 1\ntests: ['a//" + strings.Repeat("*b", 5) + "']\n",
			ProjectFileName + ": tests[0]: glob has more than 4 * " +
				"(** and lone * segments are free)",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			file, err := decodeFile([]byte(testCase.content))
			if err != nil {
				t.Fatalf("decodeFile() error = %v", err)
			}
			if err = file.Validate(testRules()); err == nil || err.Error() != testCase.want {
				t.Fatalf("Validate() error = %v, want only %q", err, testCase.want)
			}
		})
	}
}
