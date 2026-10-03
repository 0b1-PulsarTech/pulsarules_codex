package singleletter

import "testing"

func TestTestingExemption(t *testing.T) {
	t.Parallel()

	testCases := exemptCases()
	testCases = append(testCases, wrappedCases()...)
	testCases = append(testCases, notExemptCases()...)
	testCases = append(testCases, dotImportCases()...)
	testCases = append(testCases, dotImportRejectCases()...)
	testCases = append(testCases, importSetCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assertMessages(t, testCase.source, defaultSettings(), testCase.want)
		})
	}
}

func exemptCases() []testCase {
	return []testCase{
		{
			"blank identifier",
			"package pkg\n\nfunc run(_ int) { _ = 1 }\n",
			nil,
		},
		{
			"two-letter names",
			"package pkg\n\nvar ok, id int\n\n" +
				"func (sv Srv) Do(db int) {}\n",
			nil,
		},
		{
			"testing T",
			"package pkg\n\nimport \"testing\"\n\nfunc TestX(t *testing.T) {}\n",
			nil,
		},
		{
			"testing B",
			"package pkg\n\nimport \"testing\"\n\nfunc Bench(b *testing.B) {}\n",
			nil,
		},
		{
			"testing F",
			"package pkg\n\nimport \"testing\"\n\nfunc FuzzX(f *testing.F) {}\n",
			nil,
		},
		{
			"testing M",
			"package pkg\n\nimport \"testing\"\n\nfunc TestMain(m *testing.M) {}\n",
			nil,
		},
		{
			"testing TB",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t testing.TB) {}\n",
			nil,
		},
	}
}

func wrappedCases() []testCase {
	return []testCase{
		{
			"testing function literal parameter",
			"package pkg\n\nimport \"testing\"\n\nvar run = func(t *testing.T) {}\n",
			nil,
		},
		{
			"aliased testing import",
			"package pkg\n\nimport tst \"testing\"\n\nfunc TestX(t *tst.T) {}\n",
			nil,
		},
		{
			"parenthesised pointer type",
			"package pkg\n\nimport \"testing\"\n\nfunc paren(t (*testing.T)) {}\n",
			nil,
		},
		{
			"parenthesised testing selector",
			"package pkg\n\nimport \"testing\"\n\nfunc paren(t *(testing.T)) {}\n",
			nil,
		},
		{
			"double parenthesised pointer type",
			"package pkg\n\nimport \"testing\"\n\nfunc paren(t ((*testing.T))) {}\n",
			nil,
		},
	}
}

func notExemptCases() []testCase {
	return []testCase{
		{
			"aliased import does not exempt the plain name",
			"package pkg\n\nimport tst \"testing\"\n\nfunc TestX(t *testing.T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"file without the testing import",
			"package pkg\n\nfunc TestX(t *testing.T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"non-pointer testing T is not exempt",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t testing.T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"pointer to testing TB is not exempt",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t *testing.TB) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"other testing type is not exempt",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t *testing.C) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"other package selector is not exempt",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t *other.T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"non-selector type is not exempt",
			"package pkg\n\nimport \"testing\"\n\nfunc help(t int) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"bare pointer to a local type is not exempt",
			"package pkg\n\nimport \"testing\"\n\ntype T struct{}\n\nfunc help(t *T) {}\n",
			[]string{`type "T" is a single letter`, `parameter "t" is a single letter`},
		},
		{
			"testing exemption does not cover a result",
			"package pkg\n\nimport \"testing\"\n\nfunc help() (t *testing.T) { return }\n",
			[]string{`result "t" is a single letter`},
		},
	}
}

func dotImportCases() []testCase {
	return []testCase{
		{
			"dot-imported testing T",
			"package pkg\n\nimport . \"testing\"\n\nfunc TestSwap(t *T) {}\n",
			nil,
		},
		{
			"dot-imported testing B",
			"package pkg\n\nimport . \"testing\"\n\nfunc Bench(b *B) {}\n",
			nil,
		},
		{
			"dot-imported testing F",
			"package pkg\n\nimport . \"testing\"\n\nfunc FuzzX(f *F) {}\n",
			nil,
		},
		{
			"dot-imported testing M",
			"package pkg\n\nimport . \"testing\"\n\nfunc TestMain(m *M) {}\n",
			nil,
		},
		{
			"dot-imported testing TB",
			"package pkg\n\nimport . \"testing\"\n\nfunc help(t TB) {}\n",
			nil,
		},
	}
}

func dotImportRejectCases() []testCase {
	return []testCase{
		{
			"dot-imported pointer to TB is not exempt",
			"package pkg\n\nimport . \"testing\"\n\nfunc help(t *TB) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"dot-imported non-pointer T is not exempt",
			"package pkg\n\nimport . \"testing\"\n\nfunc help(t T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"dot-imported other type is not exempt",
			"package pkg\n\nimport . \"testing\"\n\nfunc help(t *C) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"dot-imported testing does not exempt the qualified name",
			"package pkg\n\nimport . \"testing\"\n\nfunc help(t *testing.T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
		{
			"bare T without the testing import",
			"package pkg\n\nfunc help(t *T) {}\n",
			[]string{`parameter "t" is a single letter`},
		},
	}
}

func importSetCases() []testCase {
	return []testCase{
		{
			"plain testing import listed before the aliased one",
			"package pkg\n\nimport \"testing\"\nimport tst \"testing\"\n\n" +
				"func TestX(t *tst.T, b *testing.B) {}\n",
			nil,
		},
		{
			"testing imported twice under two names",
			"package pkg\n\nimport tst \"testing\"\nimport \"testing\"\n\n" +
				"func TestX(t *tst.T, b *testing.B) {}\n",
			nil,
		},
		{
			"plain and dot imports of testing",
			"package pkg\n\nimport . \"testing\"\nimport \"testing\"\n\n" +
				"func TestX(t *T, b *testing.B) {}\n",
			nil,
		},
	}
}
