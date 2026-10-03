package loopvari

import (
	"testing"
)

func TestScanner(t *testing.T) {
	t.Parallel()

	testCases := append(scopeCases(), append(declCases(), varCases()...)...)
	runLoopCases(t, testCases)
}

func scopeCases() []loopCase {
	return []loopCase{
		{
			name: "shadowed by define in body",
			body: "for i := range s {\n _ = s[i]\n i := 5\n _ = fmt.Sprint(i)\n}",
		},
		{
			name:   "use before the shadow still counts",
			body:   "for i := range s {\n _ = fmt.Sprint(i)\n i := 5\n _ = i\n}",
			expect: 1,
		},
		{
			name:   "copy of the outer i is a use",
			body:   "for i := range s {\n i := i\n _ = fmt.Sprint(i)\n}",
			expect: 1,
		},
		{
			name: "shadowed by var",
			body: "for i := range s {\n _ = s[i]\n var i int\n _ = fmt.Sprint(i)\n}",
		},
		{
			name: "shadowed by closure parameter",
			body: "for i := range s {\n _ = s[i]\n _ = func(i int) int { return i }\n}",
		},
		{
			name: "shadowed by a nested block",
			body: "for i := range s {\n if true {\n  i := 1\n  _ = fmt.Sprint(i)\n }\n" +
				" _ = s[i]\n}",
		},
		{
			name: "inner loop reuses i and flags on its own",
			body: "for i := range s {\n for i := range s {\n  _ = fmt.Sprint(i)\n }\n" +
				" _ = s[i]\n}",
			expect: 1,
		},
		{
			name: "two clean loops",
			body: "for i := range s {\n s[i]++\n}\nfor i := range s {\n s[i]--\n}",
		},
		{
			name: "field named i is not a use",
			body: "type T struct{ i int }\nvar t T\nfor i := range s {\n _ = t.i + s[i]\n}",
		},
		{
			name:   "switch case body",
			body:   "for i := range s {\n switch {\n case n > 0:\n  _ = fmt.Sprint(i)\n }\n}",
			expect: 1,
		},
		{
			name:   "select comm body",
			body:   "for i := range s {\n select {\n default:\n  _ = fmt.Sprint(i)\n }\n}",
			expect: 1,
		},
	}
}

func declCases() []loopCase {
	return []loopCase{
		{
			name:   "case expression compares i",
			body:   "for i := range s {\n switch {\n case i == 0:\n  _ = n\n }\n}",
			expect: 1,
		},
		{
			name:   "case expression modulo",
			body:   "for i := range s {\n switch {\n case i%2 == 0:\n  _ = n\n }\n}",
			expect: 1,
		},
		{
			name: "comm send of i",
			body: "ch := make(chan int, 1)\nfor i := range s {\n select {\n case ch <- i:\n" +
				" }\n}",
			expect: 1,
		},
		{
			name: "comm receive binding shadows its body",
			body: "ch := make(chan int)\nfor i := range s {\n _ = s[i]\n select {\n" +
				" case i := <-ch:\n  _ = fmt.Sprint(i)\n }\n}",
		},
		{
			name: "if init shadows",
			body: "for i := range s {\n _ = s[i]\n if i := n; i > 0 {\n  _ = fmt.Sprint(i)\n }\n}",
		},
		{
			name:   "if init rhs reads the outer i",
			body:   "for i := range s {\n if i := fmt.Sprint(i); i != \"\" {\n  _ = i\n }\n}",
			expect: 1,
		},
		{
			name: "switch init shadows",
			body: "for i := range s {\n _ = s[i]\n switch i := n; i {\n case 1:\n" +
				"  _ = fmt.Sprint(i)\n }\n}",
		},
		{
			name: "type switch init shadows",
			body: "for i := range s {\n _ = s[i]\n switch i := n; any(i).(type) {\n case int:\n" +
				"  _ = fmt.Sprint(i)\n }\n}",
		},
		{
			name: "type switch binding shadows",
			body: "var v any\nfor i := range s {\n _ = s[i]\n switch i := v.(type) {\n" +
				" case int:\n  _ = fmt.Sprint(i)\n }\n}",
		},
		{
			name: "range value shadows",
			body: "for i := range s {\n _ = s[i]\n for _, i := range s {\n  _ = fmt.Sprint(i)\n" +
				" }\n}",
		},
		{
			name:   "range expression reads the outer i",
			body:   "for i := range s {\n for i := range fmt.Sprint(i) {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
		{
			name:   "inner for init rhs reads the outer i",
			body:   "for i := range s {\n for i := s[0] + i; i < n; i++ {\n }\n}",
			expect: 1,
		},
		{
			name: "closure result shadows",
			body: "for i := range s {\n _ = s[i]\n _ = func() (i int) { return i }\n}",
		},
		{
			name: "multi define shadows",
			body: "for i := range s {\n _ = s[i]\n i, err := n, error(nil)\n" +
				" _ = fmt.Sprint(i, err)\n}",
		},
	}
}

func varCases() []loopCase {
	return []loopCase{
		{
			name: "var declaration shadows the uses after it",
			body: "for i := range s {\n _ = s[i]\n var i = n\n _ = fmt.Sprint(i)\n}",
		},
		{
			name:   "var initialiser calls with the outer i",
			body:   "for i := range s {\n var i = fmt.Sprint(i)\n _ = i\n}",
			expect: 1,
		},
		{
			name:   "var initialiser doubles the outer i",
			body:   "for i := range s {\n var i = i * 2\n _ = i\n}",
			expect: 1,
		},
		{
			name:   "var group reads i before declaring it",
			body:   "for i := range s {\n var (\n  a = i\n  i = 3\n )\n _ = a\n _ = i\n}",
			expect: 1,
		},
		{
			name: "var group reads its own i afterwards",
			body: "for i := range s {\n _ = s[i]\n var (\n  i = 3\n  b = i\n )\n _ = b\n}",
		},
	}
}

func TestScanNode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want bool
	}{
		{name: "empty body", body: ""},
		{name: "index use", body: "_ = s[i]"},
		{name: "call use", body: "_ = fmt.Sprint(i)", want: true},
		{
			name: "stops at the first escape",
			body: "_ = fmt.Sprint(i)\n_ = fmt.Sprint(i)",
			want: true,
		},
		{name: "closure capture", body: "_ = func() int { return i }", want: true},
		{name: "shadowing define stops the scan", body: "i := 1\n_ = fmt.Sprint(i)"},
		{name: "shadowing var stops the scan", body: "var i = 1\n_ = fmt.Sprint(i)"},
		{name: "shadowing closure parameter", body: "_ = func(i int) int { return i }"},
		{name: "range define shadows", body: "for i := range s {\n _ = fmt.Sprint(i)\n}"},
		{name: "case clause list", body: "switch {\ncase i > 0:\n}", want: true},
		{name: "comm clause body", body: "select {\ndefault:\n _ = fmt.Sprint(i)\n}", want: true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			source := "package foo\nimport \"fmt\"\nvar _ = fmt.Sprint\n" +
				"func f(s []int, i int) {\n" + testCase.body + "\n}\n"
			f := parseDecls(t, source)
			body := funcBody(t, f, 2)
			if got := newScanner(nil).scanNode(body); got != testCase.want {
				t.Fatalf("got %v, want %v", got, testCase.want)
			}
		})
	}
}
