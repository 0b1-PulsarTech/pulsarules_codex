package projectfile

import (
	"slices"
	"strings"
	"testing"
)

func TestValidateGlobStars(t *testing.T) {
	t.Parallel()

	const tooMany = "glob has more than 4 * (** and lone * segments are free)"
	stars := func(count int) string { return strings.Repeat("*a", count) }
	runValidateCases(t, []validateCase{
		{"four stars in one segment", "version: 1\ntests: ['" + stars(4) + "']\n", ""},
		{
			"five stars in one segment",
			"version: 1\ntests: ['" + stars(5) + "']\n",
			"tests[0]: " + tooMany,
		},
		{
			"stars spread over segments",
			"version: 1\ntests: ['*a/*a/*a/*a/*a']\n",
			"tests[0]: " + tooMany,
		},
		{"double-star segments are free", "version: 1\ntests: ['**/**/**/**/**/**/*a']\n", ""},
		{"lone star segments are free", "version: 1\ntests: ['*/*/*/*/*.dart']\n", ""},
		{"many lone star segments", "version: 1\ntests: ['*/*/*/*/*/*/*/*/*/*/*a']\n", ""},
		{
			"lone star segments beside four counted stars",
			"version: 1\ntests: ['*/" + stars(4) + "/*']\n",
			"",
		},
		{
			"a counted star beside lone star segments",
			"version: 1\ntests: ['*/*/" + stars(5) + "/*']\n",
			"tests[0]: " + tooMany,
		},
		{
			"a segment of two stars is not lone",
			"version: 1\ntests: ['*a/**b/*c/*d/*e']\n",
			"tests[0]: " + tooMany,
		},
		{
			"an override glob",
			"version: 1\noverrides: [{files: ['a', '" + stars(5) + "']}]\n",
			"overrides[0].files[1]: " + tooMany,
		},
		{
			"a star run counts each star",
			"version: 1\ntests: ['*****']\n",
			"tests[0]: " + tooMany,
		},
	})
}

func TestValidateGlobBraces(t *testing.T) {
	t.Parallel()

	const refused = "brace alternatives are not supported; list each glob"
	runValidateCases(t, []validateCase{
		{
			"alternation in tests",
			"version: 1\ntests: ['web/src/**/*.{ts,tsx}']\n",
			"tests[0]: " + refused,
		},
		{
			"an opening brace in an override",
			"version: 1\noverrides: [{files: ['a', 'b{']}]\n",
			"overrides[0].files[1]: " + refused,
		},
		{
			"a closing brace in an override",
			"version: 1\noverrides: [{files: ['}b']}]\n",
			"overrides[0].files[0]: " + refused,
		},
		{"a brace inside a class is a literal", "version: 1\ntests: ['lib/[{]x.go']\n", ""},
		{"a closing brace inside a class is a literal", "version: 1\ntests: ['lib/[}]x']\n", ""},
		{
			"a brace after a class is refused",
			"version: 1\ntests: ['lib/[a]{x,y}']\n",
			"tests[0]: " + refused,
		},
		{
			"an escaped bracket does not open a class",
			"version: 1\ntests: ['lib/\\[{x']\n",
			"tests[0]: " + refused,
		},
		{
			"the brace message wins over the star cap",
			"version: 1\ntests: ['{a,b}/*a*a*a*a*a']\n",
			"tests[0]: " + refused,
		},
		{"an escaped opening brace is a literal", `version: 1` + "\ntests: ['a\\{b']\n", ""},
		{"an escaped closing brace is a literal", `version: 1` + "\ntests: ['a\\}b']\n", ""},
		{
			"an escaped backslash does not escape the brace",
			"version: 1\ntests: ['a\\\\{b']\n",
			"tests[0]: " + refused,
		},
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
	want := Name + ": overrides and tests: more than 256 globs in total"
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
			Name + ": tests[0]: glob longer than 256 bytes",
		},
		{
			"in an override",
			"version: 1\noverrides: [{files: ['" + invalid + "']}]\n",
			Name + ": overrides[0].files[0]: glob longer than 256 bytes",
		},
		{
			"with too many stars",
			"version: 1\ntests: ['a//" + strings.Repeat("*b", 5) + "']\n",
			Name + ": tests[0]: glob has more than 4 * " +
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

func TestValidateGlobChecksPrecedeBudgets(t *testing.T) {
	t.Parallel()

	list := func(globs ...string) string { return "['" + strings.Join(globs, "', '") + "']" }
	overlong := "**/*" + strings.Repeat("a", 300)
	tail := "*" + strings.Repeat("b", 255)
	class := "[" + strings.Repeat("a", 127) + "]"
	prefix := Name + ": "
	testCases := []struct {
		name    string
		content string
		want    string
	}{
		{
			"overlong globs are not charged to the budgets",
			"version: 1\ntests: " + list(slices.Repeat([]string{overlong}, 8)...) + "\n",
			overlongProblems(prefix, 8),
		},
		{
			"the class budget is reported before the star text budget",
			"version: 1\ntests: " + list(slices.Repeat([]string{tail}, 9)...) +
				"\noverrides: [{files: " + list(class, class) + "}]\n",
			prefix + "overrides and tests: wildcard classes and ? exceed 256 in total",
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

func TestValidateUncompilableGlobsAreNotCharged(t *testing.T) {
	t.Parallel()

	unclosed := "'[" + strings.Repeat("a", 130) + "'"
	content := "version: 1\ntests: [" + unclosed + ", " + unclosed + "]\n"
	file, err := decodeFile([]byte(content))
	if err != nil {
		t.Fatalf("decodeFile() error = %v", err)
	}
	err = file.Validate(testRules())
	if err == nil || strings.Contains(err.Error(), "overrides and tests") {
		t.Fatalf("Validate() error = %v, want only the compile problems", err)
	}
	if got := strings.Count(err.Error(), "compile glob"); got != 2 {
		t.Fatalf("Validate() error = %v, want 2 compile problems", err)
	}
}
