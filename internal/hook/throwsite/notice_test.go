package throwsite

import (
	"strings"
	"testing"
)

func TestNotice(t *testing.T) {
	t.Parallel()

	const tail = ": return a result (the error last, or named) instead of throwing; " +
		"a throw stays only for a broken invariant or a framework-mandated boundary."
	testCases := []struct {
		name  string
		hits  []Hit
		skill string
		want  string
	}{
		{"one line", hitsAt(7), "", "`throw` at line 7 of a.ts" + tail},
		{"two lines", hitsAt(12, 40), "", "`throw` at lines 12, 40 of a.ts" + tail},
		{
			"exactly the cap",
			hitsAt(1, 2, 3, 4, 5),
			"",
			"`throw` at lines 1, 2, 3, 4, 5 of a.ts" + tail,
		},
		{
			"past the cap",
			hitsAt(1, 2, 3, 4, 5, 6, 7, 8),
			"",
			"`throw` at lines 1, 2, 3, 4, 5 and 3 more of a.ts" + tail,
		},
		{
			"installed skill",
			hitsAt(3),
			"ts-errors",
			"`throw` at line 3 of a.ts" + tail + " See the ts-errors skill.",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Notice("a.ts", testCase.hits, testCase.skill); got != testCase.want {
				t.Errorf("Notice = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestNotice_SaysWhatToDoInsteadOfWhatMatched(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		hits []Hit
		want string
	}{
		{"throw", []Hit{{1, "throw"}}, "instead of throwing; a throw stays"},
		{"rethrow", []Hit{{1, "rethrow"}}, "instead of throwing; a throw stays"},
		{
			"promise reject",
			[]Hit{{1, "Promise.reject"}},
			"instead of returning a rejected promise; a throw stays",
		},
		{
			"future error",
			[]Hit{{1, "Future.error"}},
			"instead of returning a failed future; a throw stays",
		},
		{
			"both on one line",
			[]Hit{{1, "Promise.reject"}, {1, "throw"}},
			"at line 1 of a.ts: return a result (the error last, or named) " +
				"instead of returning a rejected promise or throwing; a throw stays",
		},
		{
			"both future error and throw",
			[]Hit{{1, "Future.error"}, {1, "rethrow"}},
			"instead of returning a failed future or throwing; a throw stays",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Notice("a.ts", testCase.hits, ""); !strings.Contains(got, testCase.want) {
				t.Errorf("Notice = %q, want it to contain %q", got, testCase.want)
			}
		})
	}
}

func TestNotice_NamesWhatMatched(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		hits []Hit
		want string
	}{
		{"rethrow", []Hit{{2, "rethrow"}}, "`rethrow` at line 2 of a.dart"},
		{"reject", []Hit{{1, "Promise.reject"}}, "`Promise.reject` at line 1 of a.dart"},
		{"future error", []Hit{{1, "Future.error"}}, "`Future.error` at line 1 of a.dart"},
		{
			"mixed words once each",
			[]Hit{{1, "throw"}, {2, "Promise.reject"}, {3, "throw"}},
			"`throw` / `Promise.reject` at lines 1, 2, 3 of a.dart",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := Notice(
				"a.dart",
				testCase.hits,
				"",
			); !strings.HasPrefix(
				got,
				testCase.want+":",
			) {
				t.Errorf("Notice = %q, want prefix %q", got, testCase.want)
			}
		})
	}
}

func hitsAt(lines ...int) []Hit {
	hits := make([]Hit, 0, len(lines))
	for _, line := range lines {
		hits = append(hits, Hit{Line: line, Word: "throw"})
	}
	return hits
}

func TestSkillFor(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		ext  string
		want string
	}{
		{".dart", "dart-errors"},
		{".DART", "dart-errors"},
		{".ts", "ts-errors"},
		{".js", "ts-errors"},
		{".svelte", "ts-errors"},
		{".go", ""},
		{"", ""},
	}
	for _, testCase := range testCases {
		t.Run(testCase.ext, func(t *testing.T) {
			t.Parallel()

			if got := SkillFor(testCase.ext); got != testCase.want {
				t.Errorf("SkillFor(%q) = %q, want %q", testCase.ext, got, testCase.want)
			}
		})
	}
}
