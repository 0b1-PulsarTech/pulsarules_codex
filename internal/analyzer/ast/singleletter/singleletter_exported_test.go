package singleletter

import "testing"

func TestExportedOnlyKinds(t *testing.T) {
	t.Parallel()

	testCases := skippedKindCases()
	testCases = append(testCases, reportedKindCases()...)
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := defaultSettings()
			cfg.exportedOnly = true
			assertMessages(t, testCase.source, cfg, testCase.want)
		})
	}
}

func skippedKindCases() []testCase {
	return []testCase{
		{"type parameter", "package pkg\n\nfunc Map[T any]() {}\n", nil},
		{"type declaration parameter", "package pkg\n\ntype Box[V any] struct{}\n", nil},
		{"package clause", "package Z\n", nil},
		{"label", "package pkg\n\nfunc run() {\nL:\n\tfor {\n\t\tbreak L\n\t}\n}\n", nil},
		{"receiver", "package pkg\n\ntype Srv struct{}\n\nfunc (T Srv) run() {}\n", nil},
		{
			"range variable",
			"package pkg\n\nfunc run(rows []int) { for K := range rows { _ = K } }\n",
			nil,
		},
		{"for clause variable", "package pkg\n\nfunc run() { for I := 0; I < 3; I++ {} }\n", nil},
		{"parameter", "package pkg\n\nfunc run(Q int) {}\n", nil},
		{"result", "package pkg\n\nfunc run() (R int) { return }\n", nil},
		{
			"type switch binding",
			"package pkg\n\nfunc run(val any) { switch V := val.(type) { case int: _ = V } }\n",
			nil,
		},
		{"import alias", "package pkg\n\nimport F \"fmt\"\n\nvar _ = F.Sprint\n", nil},
		{"non-ASCII lower case constant", "package pkg\n\nconst é = 1\n", nil},
		{"letter without case", "package pkg\n\nconst 中 = 1\n", nil},
		{"lower case function", "package pkg\n\nfunc f() {}\n", nil},
	}
}

func reportedKindCases() []testCase {
	return []testCase{
		{"function", "package pkg\n\nfunc F() {}\n", []string{`function "F" is a single letter`}},
		{
			"method",
			"package pkg\n\ntype Srv struct{}\n\nfunc (srv Srv) M() {}\n",
			[]string{`method "M" is a single letter`},
		},
		{"type", "package pkg\n\ntype T struct{}\n", []string{`type "T" is a single letter`}},
		{"constant", "package pkg\n\nconst C = 1\n", []string{`constant "C" is a single letter`}},
		{"variable", "package pkg\n\nvar V int\n", []string{`variable "V" is a single letter`}},
		{
			"field",
			"package pkg\n\ntype Row struct{ X int }\n",
			[]string{`field "X" is a single letter`},
		},
		{
			"local short declaration",
			"package pkg\n\nfunc run() { X := 1; _ = X }\n",
			[]string{`variable "X" is a single letter`},
		},
		{
			"local grouped variable",
			"package pkg\n\nfunc run() { var ( Y = 2 ); _ = Y }\n",
			[]string{`variable "Y" is a single letter`},
		},
		{
			"local anonymous struct field",
			"package pkg\n\nfunc run() { var anon struct{ K int }; _ = anon }\n",
			[]string{`field "K" is a single letter`},
		},
		{
			"local type",
			"package pkg\n\nfunc run() { type L int }\n",
			[]string{`type "L" is a single letter`},
		},
		{
			"interface method",
			"package pkg\n\ntype Doer interface{ D() }\n",
			[]string{`method "D" is a single letter`},
		},
		{
			"non-ASCII upper case constant",
			"package pkg\n\nconst É = 1\n",
			[]string{"constant \"É\" is a single letter"},
		},
	}
}
