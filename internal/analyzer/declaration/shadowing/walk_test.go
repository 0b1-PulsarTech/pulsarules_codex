package shadowing

import "testing"

func TestShadowing_ShortDeclSignatureNames(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()

	for _, testCase := range functionBlockTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCheckFile(t, analyzer, testCase)
		})
	}
}

func functionBlockTestCases() []checkFileTestCase {
	return append(functionBlockReuseCases(), functionBlockShadowCases()...)
}

func functionBlockReuseCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "tracer start at the top of the body reuses the ctx param",
			source: "package foo\n" +
				"func f(ctx int) {\n" +
				"	ctx, span := start(ctx)\n" +
				"	_, _ = ctx, span\n" +
				"}\n" +
				"func start(c int) (int, int) { return c, 0 }\n",
			expect:    0,
			wantReuse: 1,
		},
		{
			name: "named result reused at the top of the body",
			source: "package foo\n" +
				"func f() (out int) {\n" +
				"	out, extra := 1, 2\n" +
				"	_ = extra\n" +
				"	return out\n" +
				"}\n",
			expect:    0,
			wantReuse: 1,
		},
		{
			name: "receiver reused at the top of the body",
			source: "package foo\n" +
				"type T struct{}\n" +
				"func (w T) f(other T) {\n" +
				"	w, extra := other, 1\n" +
				"	_, _ = w, extra\n" +
				"}\n",
			expect:    0,
			wantReuse: 1,
		},
		{
			name: "every signature name reused in one statement is reported once each",
			source: "package foo\n" +
				"func f(a int, b int) {\n" +
				"	a, b, c := 1, 2, 3\n" +
				"	_, _, _ = a, b, c\n" +
				"}\n",
			expect:    0,
			wantReuse: 2,
		},
	}
}

func functionBlockShadowCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "the same reassignment one block deeper is real shadowing",
			source: "package foo\n" +
				"func f(ctx int) {\n" +
				"	if true {\n" +
				"		for i := 0; i < 2; i++ {\n" +
				"			ctx := 1\n" +
				"			_ = ctx\n" +
				"		}\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "receiver shadowed by a range variable",
			source: "package foo\n" +
				"type T struct{}\n" +
				"func (w T) f(items []T) {\n" +
				"	for _, w := range items {\n" +
				"		_ = w\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "err carried across successive short declarations is silent",
			source: "package foo\n" +
				"func f() {\n" +
				"	a, err := g()\n" +
				"	if err != nil {\n" +
				"		return\n" +
				"	}\n" +
				"	b, err := g()\n" +
				"	_, _, _ = a, b, err\n" +
				"}\n" +
				"func g() (int, error) { return 0, nil }\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			name: "a var redeclaring a param one block deeper still shadows",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	{\n" +
				"		var x int\n" +
				"		_ = x\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "a blank receiver name binds nothing",
			source: "package foo\n" +
				"type T struct{}\n" +
				"func (T) f() {\n" +
				"	x := 1\n" +
				"	_ = x\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
	}
}

func TestReuseMessage(t *testing.T) {
	t.Parallel()

	analyzer := NewAnalyzer()
	source := "package foo\n" +
		"func f(ctx int) {\n" +
		"	ctx, span := start(ctx)\n" +
		"	_, _ = ctx, span\n" +
		"}\n" +
		"func start(c int) (int, int) { return c, 0 }\n"

	want := `"ctx" is reassigned by := in the block that declares it`
	if got := analyzeMessage(t, analyzer, source); got != want {
		t.Fatalf("message = %q, want %q", got, want)
	}
}
