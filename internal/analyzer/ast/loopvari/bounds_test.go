package loopvari

import (
	"testing"
)

func TestBounds(t *testing.T) {
	t.Parallel()

	testCases := append(boundCases(), innerLoopCases()...)
	testCases = append(testCases, innerCounterCases()...)
	testCases = append(testCases, builtinShadowCases()...)
	testCases = append(testCases, scopedShadowCases()...)
	runLoopCases(t, testCases)
}

func boundCases() []loopCase {
	return []loopCase{
		{
			name: "compared to cap",
			body: "for i := range s {\n if i+1 < cap(s) {\n  _ = s[i]\n }\n}",
		},
		{
			name: "compared to a constant plus len",
			body: "for i := range s {\n if i < 1+len(s) {\n  _ = s[i]\n }\n}",
		},
		{
			name:   "compared to half of len",
			body:   "for i := range s {\n if i < len(s)/2 {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
		{
			name:   "compared to a constant minus len",
			body:   "for i := range s {\n if i < 1-len(s) {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
		{
			name:   "compared to a product with len",
			body:   "for i := range s {\n if i < n*len(s) {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
	}
}

func builtinShadowCases() []loopCase {
	return []loopCase{
		{
			name: "len shadowed by a local function",
			body: "len := func(x []int) int { return 0 }\nfor i := range s {\n" +
				" if i < len(s) {\n  _ = s[i]\n }\n}",
			expect: 1,
		},
		{
			name:  "cap named as a struct field",
			body:  "for i := range s {\n if i+1 < cap(s) {\n  _ = s[i]\n }\n}",
			decls: "type B struct{ cap int }",
		},
		{
			name:  "len named as a parameter elsewhere",
			body:  "for i := range s {\n if i < len(s) {\n  _ = s[i]\n }\n}",
			decls: "func g(len int) int { return len }",
		},
		{
			name:  "len named as a method",
			body:  "for i := range s {\n if i < len(s) {\n  _ = s[i]\n }\n}",
			decls: "type B struct{}\nfunc (B) len() int { return 0 }",
		},
		{
			name:   "len declared at package level",
			body:   "for i := range s {\n if i < len(s) {\n  _ = s[i]\n }\n}",
			decls:  "func len(x []int) int { return 0 }",
			expect: 1,
		},
		{
			name: "len shadowed by a closure parameter",
			body: "g := func(len func([]int) int) {\n for i := range s {\n  if i < len(s) {\n" +
				"   _ = s[i]\n  }\n }\n}\n_ = g",
			expect: 1,
		},
	}
}

func scopedShadowCases() []loopCase {
	const loop = "for i := range s {\n if i < len(s)-1 {\n  _ = s[i+1]\n }\n}"
	const inner = "func() int {\n" + loop + "\nreturn 0\n}"
	return []loopCase{
		{
			name: "loop inside the initializer of a len define",
			body: "len := " + inner + "\n_ = len",
		},
		{
			name: "loop inside the initializer of a len var",
			body: "var len = " + inner + "\n_ = len",
		},
		{name: "len declared after the loop", body: loop + "\nlen := 3\n_ = len"},
		{
			name: "sibling closure with a len parameter",
			body: "g := func(len int) int { return len }\n_ = g\n" + loop,
		},
		{
			name: "sibling closure with a local len",
			body: "g := func() int { len := 2; return len }\n_ = g\n" + loop,
		},
		{
			name: "function type variable with a len parameter",
			body: "var g func(len int)\n_ = g\n" + loop,
		},
	}
}

func innerLoopCases() []loopCase {
	return []loopCase{
		{
			name: "inner loop starts after i",
			body: "for i := range s {\n for j := i + 1; j < n; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner loop starts before i",
			body: "for i := range s {\n for j := i - 1; j >= 0; j-- {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner loop bounded by i",
			body: "for i := range s {\n for j := 0; j < i; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner loop bounded by i inside a conjunction",
			body: "for i := range s {\n for j := 0; j < i && j < n; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner loop body compares to i",
			body: "for i := range s {\n for j := range s {\n  if j < i {\n   _ = s[j]\n  }\n" +
				" }\n}",
			expect: 1,
		},
		{
			name: "inner loop start from a call of i",
			body: "g := func(x int) int { return x }\nfor i := range s {\n" +
				" for j := g(i); j < n; j++ {\n  _ = s[j]\n }\n}",
			expect: 1,
		},
		{
			name: "inner loop starts at i",
			body: "for i := range s {\n for j := i; j < n; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name:   "plain define of i outside a loop init",
			body:   "for i := range s {\n j := i\n _ = s[j]\n}",
			expect: 1,
		},
		{
			name: "inner loop starts by plain assign",
			body: "j := 0\nfor i := range s {\n for j = i + 1; j < n; j++ {\n  _ = s[j]\n }\n}\n" +
				"_ = j",
		},
	}
}

func innerCounterCases() []loopCase {
	return []loopCase{
		{
			name: "inner counter declared before a bare-init loop",
			body: "for i := range s {\n j := 0\n for ; j < i; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner counter stepped in the body",
			body: "for i := range s {\n j := 0\n for j < i {\n  _ = s[j]\n  j++\n }\n}",
		},
		{
			name: "inner counter with an offset bound",
			body: "for i := range s {\n for j := 0; j+1 < i; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner bound inside a parenthesised conjunction",
			body: "for i := range s {\n for j := 0; (j < i) && j < n; j++ {\n  _ = s[j]\n }\n}",
		},
		{
			name: "inner body redeclares the compared name",
			body: "for i := range s {\n j := 0\n for j < i {\n  j := 5\n  _ = j\n  break\n }\n" +
				" _ = j\n}",
			expect: 1,
		},
		{
			name: "nested loop init redeclares the compared name",
			body: "for i := range s {\n j := 0\n for j < i {\n  for j := 0; j < 3; j++ {\n" +
				"  }\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "inner counter moved by assignment",
			body: "for i := range s {\n j := 0\n for j < i {\n  _ = s[j]\n  j += 1\n }\n}",
		},
		{
			name:   "inner loop never moves the compared name",
			body:   "for i := range s {\n j := 0\n for j < i {\n  break\n }\n _ = j\n}",
			expect: 1,
		},
		{
			name: "inner loop assigns i alongside the counter",
			body: "j := 0\nfor i := range s {\n _ = s[i]\n for j, i = 0, 0; j < n; j++ {\n" +
				"  _ = s[j]\n }\n}\n_ = j",
			expect: 1,
		},
		{
			name:   "bare inner loop condition is not a bound",
			body:   "for i := range s {\n for i < 3 {\n  break\n }\n}",
			expect: 1,
		},
	}
}
