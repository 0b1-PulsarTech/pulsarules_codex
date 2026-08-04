package shadowing

import "testing"

// TestClauseScopes pins the part of the Go spec the walk used to collapse:
// each clause of a switch or select is its own implicit block, and the braces
// around the clauses are not a block at all.
func TestClauseScopes(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	for _, testCase := range clauseTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCheckFile(t, a, testCase)
		})
	}
}

func clauseTestCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			// Every clause is a block of its own, so both declarations shadow
			// the parameter. Sharing one scope hid the second.
			name: "each switch clause shadows the param independently",
			source: "package foo\n" +
				"func f(v int, x int) {\n" +
				"	switch x {\n" +
				"	case 1:\n" +
				"		v := 1\n" +
				"		_ = v\n" +
				"	case 2:\n" +
				"		v := 2\n" +
				"		_ = v\n" +
				"	}\n" +
				"}\n",
			expect:    2,
			wantReuse: 0,
		},
		{
			name: "a select comm clause binds and is checked",
			source: "package foo\n" +
				"func f(v int, ch chan int) {\n" +
				"	select {\n" +
				"	case v := <-ch:\n" +
				"		_ = v\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "a comm clause binding is visible to its own body",
			source: "package foo\n" +
				"func f(ch chan int) {\n" +
				"	select {\n" +
				"	case v := <-ch:\n" +
				"		if true {\n" +
				"			v := 2\n" +
				"			_ = v\n" +
				"		}\n" +
				"		_ = v\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "the type-switch guard is checked like any declaration",
			source: "package foo\n" +
				"func f(v int, i any) {\n" +
				"	switch v := i.(type) {\n" +
				"	case int:\n" +
				"		_ = v\n" +
				"	}\n" +
				"}\n",
			expect:    1,
			wantReuse: 0,
		},
		{
			name: "a clean switch reports nothing",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	switch x {\n" +
				"	case 1:\n" +
				"		y := 1\n" +
				"		_ = y\n" +
				"	case 2:\n" +
				"		z := 2\n" +
				"		_ = z\n" +
				"	}\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
	}
}

// TestClosuresAreNotDescended pins a KNOWN LIMIT, not a desired behaviour:
// function literals are not walked. Enabling the walk is a one-line change,
// but measured against a real monorepo it produced only findings of the shape
// `if err := f(); err != nil` inside a closure - correct by the spec, useless
// as advice. Reversing this test is the signal that `err` in an if-init was
// allowlisted first.
func TestClosuresAreNotDescended(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()

	for _, testCase := range closureTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			runCheckFile(t, a, testCase)
		})
	}
}

func closureTestCases() []checkFileTestCase {
	return []checkFileTestCase{
		{
			name: "a closure assigned to a variable",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	g := func() {\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}\n" +
				"	_ = g\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			name: "a deferred closure",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	defer func() {\n" +
				"		x := 2\n" +
				"		_ = x\n" +
				"	}()\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			name: "a closure parameter",
			source: "package foo\n" +
				"func f(x int) {\n" +
				"	g := func(x int) { _ = x }\n" +
				"	_ = g\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			name: "a closure parameter reused by := in the closure body",
			source: "package foo\n" +
				"func f() {\n" +
				"	g := func(ctx int) {\n" +
				"		ctx, extra := 1, 2\n" +
				"		_, _ = ctx, extra\n" +
				"	}\n" +
				"	_ = g\n" +
				"}\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			name: "a goroutine closure",
			source: "package foo\n" +
				"func f(err error) {\n" +
				"	go func() {\n" +
				"		err := g()\n" +
				"		_ = err\n" +
				"	}()\n" +
				"}\n" +
				"func g() error { return nil }\n",
			expect:    0,
			wantReuse: 0,
		},
		{
			// The commit/rollback shape the transactions skill mandates. This
			// one must stay silent even after closures are descended: the
			// defer ASSIGNS the named result, it never declares it.
			name: "the mandated commit-rollback defer is silent",
			source: "package foo\n" +
				"func f() (err error) {\n" +
				"	defer func() { err = finish(err) }()\n" +
				"	return nil\n" +
				"}\n" +
				"func finish(e error) error { return e }\n",
			expect:    0,
			wantReuse: 0,
		},
	}
}

// TestReuseReportedOncePerName pins the de-duplication: with a named result,
// every later := naming it would otherwise repeat the same advice and bury the
// rest of the report.
func TestReuseReportedOncePerName(t *testing.T) {
	t.Parallel()

	a := NewAnalyzer()
	testCase := checkFileTestCase{
		name: "a named result reused twice reports once",
		source: "package foo\n" +
			"func f() (out int, err error) {\n" +
			"	a, err := g()\n" +
			"	b, err := g()\n" +
			"	_, _, _ = a, b, out\n" +
			"	return out, err\n" +
			"}\n" +
			"func g() (int, error) { return 0, nil }\n",
		expect:    0,
		wantReuse: 1,
	}
	runCheckFile(t, a, testCase)
}
