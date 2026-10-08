package tsfacts

import (
	"slices"
	"testing"
)

type complexityCase struct {
	name   string
	body   string
	isTest bool
	want   []int
}

func runComplexity(t *testing.T, testCases []complexityCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			text := "function f(a, b, c, d) {" + testCase.body + "}\n"
			facts := collectOf(t, text, Options{IsTest: testCase.isTest})
			got := make([]int, 0, len(facts.Functions))
			for _, fn := range facts.Functions {
				got = append(got, fn.Complexity)
			}
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("complexities %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestComplexityStatements(t *testing.T) {
	t.Parallel()

	runComplexity(t, []complexityCase{
		{name: "empty", body: "", want: []int{1}},
		{name: "if", body: "if (a) {}", want: []int{2}},
		{name: "else if", body: "if (a) {} else if (b) {} else {}", want: []int{3}},
		{name: "for", body: "for (;;) {}", want: []int{2}},
		{name: "for in", body: "for (const k in a) {}", want: []int{2}},
		{name: "for of", body: "for (const k of a) {}", want: []int{2}},
		{name: "while", body: "while (a) {}", want: []int{2}},
		{name: "do while", body: "do {} while (a);", want: []int{2}},
		{
			name: "switch counts cases not default",
			body: "switch (a) { case 1: case 2: break; default: }",
			want: []int{3},
		},
		{name: "catch", body: "try {} catch (e) {} finally {}", want: []int{2}},
	})
}

func TestComplexityExpressions(t *testing.T) {
	t.Parallel()

	runComplexity(t, []complexityCase{
		{name: "ternary", body: "return a ? 1 : 2;", want: []int{2}},
		{name: "and", body: "return a && b;", want: []int{2}},
		{name: "or", body: "return a || b;", want: []int{2}},
		{name: "nullish", body: "return a ?? b;", want: []int{2}},
		{name: "logical chain", body: "return a && b || c ?? d;", want: []int{4}},
		{name: "optional chain is free", body: "return a?.b?.[c];", want: []int{1}},
		{name: "arithmetic is free", body: "return a + b * c;", want: []int{1}},
		{name: "assignment operators are free", body: "a &&= b; a ||= b;", want: []int{1}},
	})
}

func TestComplexityClosures(t *testing.T) {
	t.Parallel()

	runComplexity(t, []complexityCase{
		{
			name: "anonymous closure feeds its function",
			body: "list.map((x) => x ? 1 : 2);",
			want: []int{2, 2},
		},
		{
			name: "outer counts the branches of its callback",
			body: "if (a) {} list.forEach((x) => { if (x) {} for (;;) {} });",
			want: []int{4, 3},
		},
		{
			name: "two levels of callbacks feed every enclosing function",
			body: "run(() => { run(() => { if (a) {} }); });",
			want: []int{2, 2, 2},
		},
		{
			name: "method callbacks feed the method",
			body: "list.map(function () { return a && b; });",
			want: []int{2, 2},
		},
		{
			name: "bound closure keeps its branches to itself",
			body: "const g = () => { if (a) {} };",
			want: []int{1, 2},
		},
		{
			name: "nested declaration keeps its own branches",
			body: "function inner() { if (a) {} } if (b) {}",
			want: []int{2, 2},
		},
	})
}

func TestComplexityTestContainers(t *testing.T) {
	t.Parallel()

	runComplexity(t, []complexityCase{
		{
			name:   "container keeps its branches",
			body:   "describe('x', () => { if (a) {} });",
			isTest: true,
			want:   []int{1, 2},
		},
		{
			name:   "case inside a container feeds neither container",
			body:   "describe('x', () => { it('y', () => { if (a) {} }); });",
			isTest: true,
			want:   []int{1, 1, 2},
		},
		{
			name:   "plain callback in a container feeds the container",
			body:   "describe('x', () => { list.map(() => a && b); });",
			isTest: true,
			want:   []int{1, 2, 2},
		},
		{
			name: "same call outside a test file feeds the function",
			body: "describe('x', () => { if (a) {} });",
			want: []int{2, 2},
		},
	})
}
