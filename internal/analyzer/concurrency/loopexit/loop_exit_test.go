package loopexit

import (
	"testing"
)

type loopCanTerminateTestCase struct {
	name   string
	source string
	label  string
	want   bool
}

func loopCanTerminateReturnCases() []loopCanTerminateTestCase {
	return []loopCanTerminateTestCase{
		{
			name: "a return directly in the loop body terminates it",
			source: "package foo\n" +
				"func run() {\n" +
				"\tfor {\n" +
				"\t\treturn\n" +
				"\t}\n" +
				"}\n",
			want: true,
		},
		{
			name: "a return inside an if inside the loop body still terminates it",
			source: "package foo\n" +
				"func run(count int) {\n" +
				"\tfor {\n" +
				"\t\tif count >= 10 {\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: true,
		},
		{
			name: "a return inside a select case still terminates the loop",
			source: "package foo\n" +
				"func run(stopCh <-chan struct{}) {\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-stopCh:\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: true,
		},
		{
			name: "a return inside a nested func literal does not terminate the loop",
			source: "package foo\n" +
				"func run() {\n" +
				"\tfor {\n" +
				"\t\tgo func() {\n" +
				"\t\t\treturn\n" +
				"\t\t}()\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a return inside a nested loop does not terminate this loop",
			source: "package foo\n" +
				"func run() {\n" +
				"\tfor {\n" +
				"\t\tfor {\n" +
				"\t\t\treturn\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "no return or break at all does not terminate the loop",
			source: "package foo\n" +
				"func run() {\n" +
				"\tfor {\n" +
				"\t\twork()\n" +
				"\t}\n" +
				"}\n" +
				"func work() {}\n",
			want: false,
		},
	}
}

func loopCanTerminateBreakCases() []loopCanTerminateTestCase {
	return []loopCanTerminateTestCase{
		{
			name: "a bare break directly in the loop body terminates it",
			source: "package foo\n" +
				"func run(count int) {\n" +
				"\tfor {\n" +
				"\t\tif count >= 10 {\n" +
				"\t\t\tbreak\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: true,
		},
		{
			name: "a bare break absorbed by a nested switch does not terminate the loop",
			source: "package foo\n" +
				"func run(count int) {\n" +
				"\tfor {\n" +
				"\t\tswitch count {\n" +
				"\t\tcase 1:\n" +
				"\t\t\tbreak\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a bare break absorbed by a nested select does not terminate the loop",
			source: "package foo\n" +
				"func run(ch chan int) {\n" +
				"\tfor {\n" +
				"\t\tselect {\n" +
				"\t\tcase <-ch:\n" +
				"\t\t\tbreak\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a labeled break naming this loop escapes a nested switch",
			source: "package foo\n" +
				"func run(count int) {\n" +
				"\touter:\n" +
				"\tfor {\n" +
				"\t\tswitch count {\n" +
				"\t\tcase 1:\n" +
				"\t\t\tbreak outer\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			label: "outer",
			want:  true,
		},
		{
			name: "a labeled break naming a different loop does not terminate this one",
			source: "package foo\n" +
				"func run(count int) {\n" +
				"\tfor {\n" +
				"\t\tswitch count {\n" +
				"\t\tcase 1:\n" +
				"\t\t\tbreak other\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			label: "outer",
			want:  false,
		},
	}
}

func loopCanTerminateNestedCases() []loopCanTerminateTestCase {
	return []loopCanTerminateTestCase{
		{
			name: "a bare break absorbed by a nested range does not terminate the loop",
			source: "package foo\n" +
				"func run(items []int) {\n" +
				"\tfor {\n" +
				"\t\tfor range items {\n" +
				"\t\t\tbreak\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a bare break absorbed by a nested type switch does not terminate the loop",
			source: "package foo\n" +
				"func run(value any) {\n" +
				"\tfor {\n" +
				"\t\tswitch value.(type) {\n" +
				"\t\tcase int:\n" +
				"\t\t\tbreak\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			want: false,
		},
		{
			name: "a labeled break naming this loop escapes a nested range",
			source: "package foo\n" +
				"func run(items []int) {\n" +
				"outer:\n" +
				"\tfor {\n" +
				"\t\tfor range items {\n" +
				"\t\t\tbreak outer\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			label: "outer",
			want:  true,
		},
		{
			name: "a labeled break naming this loop escapes a nested type switch",
			source: "package foo\n" +
				"func run(value any) {\n" +
				"outer:\n" +
				"\tfor {\n" +
				"\t\tswitch value.(type) {\n" +
				"\t\tcase int:\n" +
				"\t\t\tbreak outer\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			label: "outer",
			want:  true,
		},
		{
			name: "a labeled break naming this loop escapes a nested for",
			source: "package foo\n" +
				"func run() {\n" +
				"outer:\n" +
				"\tfor {\n" +
				"\t\tfor {\n" +
				"\t\t\tbreak outer\n" +
				"\t\t}\n" +
				"\t}\n" +
				"}\n",
			label: "outer",
			want:  true,
		},
	}
}

func TestLoopCanTerminate(t *testing.T) {
	t.Parallel()

	testCases := append(loopCanTerminateReturnCases(), loopCanTerminateBreakCases()...)
	testCases = append(testCases, loopCanTerminateNestedCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			body := parseFirstForBody(t, testCase.source)
			got := loopCanTerminate(body, testCase.label)
			if got != testCase.want {
				t.Fatalf("loopCanTerminate() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestCollectLoopLabels(t *testing.T) {
	t.Parallel()

	source := "package foo\n" +
		"func run() {\n" +
		"\touter:\n" +
		"\tfor {\n" +
		"\t\tfor {\n" +
		"\t\t\tbreak outer\n" +
		"\t\t}\n" +
		"\t}\n" +
		"}\n"

	_, _, body := parseFirstFuncBody(t, source)
	labels := collectLoopLabels(body)
	if len(labels) != 1 {
		t.Fatalf("labels = %v, want exactly one labeled loop", labels)
	}
	for _, label := range labels {
		if label != "outer" {
			t.Fatalf("label = %q, want %q", label, "outer")
		}
	}
}
