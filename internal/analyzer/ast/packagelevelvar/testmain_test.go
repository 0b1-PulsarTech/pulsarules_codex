package packagelevelvar

import "testing"

func TestAnalyzeTestMain(t *testing.T) {
	t.Parallel()

	const hdr = "package p\n\nimport \"testing\"\n\n"
	runCases(t, []levelVarCase{
		{
			name: "assigned in TestMain of the same file exempt",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\n" +
					"func TestMain(m *testing.M) { factory = 1 }\n",
			},
		},
		{
			name: "tuple assignment exempt",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\nfunc TestMain(m *testing.M) {\n" +
					"\tvar err error\n" +
					"\tfactory, err = 1, nil\n\t_ = err\n}\n",
			},
		},
		{
			name: "address taken in TestMain exempt",
			files: map[string]string{
				"a_test.go": hdr + "var newEngine func()\n\nfunc run(p *func()) {}\n\n" +
					"func TestMain(m *testing.M) { run(&newEngine) }\n",
			},
		},
	})
}

func TestAnalyzeTestMainSiblings(t *testing.T) {
	t.Parallel()

	const hdr = "package p\n\nimport \"testing\"\n\n"
	runCases(t, []levelVarCase{
		{
			name: "TestMain in a sibling test file of the same dir exempt",
			files: map[string]string{
				"pkg/a_test.go":    "package p\n\nvar factory int\n",
				"pkg/main_test.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
		},
		{
			name: "TestMain in another dir does not exempt",
			files: map[string]string{
				"pkg/a_test.go":      "package p\n\nvar factory int\n",
				"other/main_test.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"pkg/a_test.go:3:package-level var factory"},
		},
		{
			name: "unassigned var beside TestMain flagged",
			files: map[string]string{
				"a_test.go": hdr + "var factory, other int\n\n" +
					"func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"a_test.go:5:package-level var other"},
		},
		{
			name: "define in TestMain does not exempt",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\n" +
					"func TestMain(m *testing.M) { factory := 1; _ = factory }\n",
			},
			want: []string{"a_test.go:5:package-level var factory"},
		},
		{
			name: "non test file never exempt",
			files: map[string]string{
				"a.go":      "package p\n\nvar factory int\n",
				"a_test.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"a.go:3:package-level var factory"},
		},
	})
}

func TestAnalyzeTestMainUnchangedSiblings(t *testing.T) {
	t.Parallel()

	const hdr = "package p\n\nimport \"testing\"\n\n"
	runCases(t, []levelVarCase{
		{
			name:  "TestMain in an unchanged sibling file exempt",
			files: map[string]string{"pkg/a_test.go": "package p\n\nvar factory int\n"},
			extra: map[string]string{
				"pkg/main_test.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
		},
		{
			name:  "TestMain in an unchanged file of another dir does not exempt",
			files: map[string]string{"pkg/a_test.go": "package p\n\nvar factory int\n"},
			extra: map[string]string{
				"other/main_test.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"pkg/a_test.go:3:package-level var factory"},
		},
		{
			name:  "TestMain of the external test package does not exempt",
			files: map[string]string{"pkg/a_test.go": "package p\n\nvar factory int\n"},
			extra: map[string]string{
				"pkg/main_test.go": "package p_test\n\nimport \"testing\"\n\n" +
					"func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"pkg/a_test.go:3:package-level var factory"},
		},
		{
			name:  "TestMain in a non test file does not exempt",
			files: map[string]string{"pkg/a_test.go": "package p\n\nvar factory int\n"},
			extra: map[string]string{
				"pkg/main.go": hdr + "func TestMain(m *testing.M) { factory = 1 }\n",
			},
			want: []string{"pkg/a_test.go:3:package-level var factory"},
		},
	})
}

func TestAnalyzeTestMainExternalPackage(t *testing.T) {
	t.Parallel()

	runCases(t, []levelVarCase{
		{
			name: "external test package var assigned in its own TestMain exempt",
			files: map[string]string{
				"a_test.go": "package p_test\n\nimport \"testing\"\n\nvar factory int\n\n" +
					"func TestMain(m *testing.M) { factory = 1 }\n",
			},
		},
	})
}

func TestAnalyzeTestMainBody(t *testing.T) {
	t.Parallel()

	const hdr = "package p\n\nimport \"testing\"\n\n"
	runCases(t, []levelVarCase{
		{
			name: "var declared inside TestMain shadows",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\nfunc TestMain(m *testing.M) {\n" +
					"\tvar factory int\n\tfactory = 2\n\t_ = factory\n}\n",
			},
			want: []string{"a_test.go:5:package-level var factory"},
		},
		{
			name: "increment in TestMain exempt",
			files: map[string]string{
				"a_test.go": hdr + "var counter int\n\n" +
					"func TestMain(m *testing.M) { counter++ }\n",
			},
		},
		{
			name: "field assignment counts the root identifier",
			files: map[string]string{
				"a_test.go": hdr + "type env struct{ db int }\n\nvar shared env\n\n" +
					"func TestMain(m *testing.M) { shared.db = 1 }\n",
			},
		},
		{
			name: "assignment inside a helper function is not counted",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\nfunc setup() { factory = 1 }\n\n" +
					"func TestMain(m *testing.M) { setup() }\n",
			},
			want: []string{"a_test.go:5:package-level var factory"},
		},
		{
			name: "method named TestMain ignored",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\ntype s struct{}\n\n" +
					"func (s) TestMain() { factory = 1 }\n",
			},
			want: []string{"a_test.go:5:package-level var factory"},
		},
		{
			name: "range define shadows the package var",
			files: map[string]string{
				"a_test.go": hdr + "var factory int\n\nfunc TestMain(m *testing.M) {\n" +
					"\tfor factory := range 3 {\n\t\tfactory = 2\n\t}\n}\n",
			},
			want: []string{"a_test.go:5:package-level var factory"},
		},
		{
			name: "index assignment counts the root identifier",
			files: map[string]string{
				"a_test.go": hdr + "var cfgs []int\n\n" +
					"func TestMain(m *testing.M) { cfgs[0] = 1 }\n",
			},
		},
	})
}
