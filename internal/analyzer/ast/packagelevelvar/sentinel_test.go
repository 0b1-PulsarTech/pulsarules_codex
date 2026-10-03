package packagelevelvar

import (
	"slices"
	"testing"
)

func typedErrorCases() []levelVarCase {
	const pre = "package p\n\nimport \"io\"\n\ntype e struct{}\n\n" +
		"func compute() error { return nil }\n\n"
	return []levelVarCase{
		{
			name:  "typed error from a call flagged",
			files: map[string]string{"a.go": pre + "var lastErr error = compute()\n"},
			want:  []string{"a.go:9:package-level var lastErr"},
		},
		{
			name:  "typed error nil flagged",
			files: map[string]string{"a.go": pre + "var lastErr error = nil\n"},
			want:  []string{"a.go:9:package-level var lastErr"},
		},
		{
			name:  "typed error re-exporting a non Err name flagged",
			files: map[string]string{"a.go": pre + "var current error = io.EOF\n"},
			want:  []string{"a.go:9:package-level var current"},
		},
		{
			name:  "Err named typed error re-exporting a name exempt",
			files: map[string]string{"a.go": pre + "var ErrEOF error = io.EOF\n"},
		},
		{
			name:  "typed error address of composite exempt",
			files: map[string]string{"a.go": pre + "var ErrX error = &e{}\n"},
		},
		{
			name:  "typed error address of composite with a call flagged",
			files: map[string]string{"a.go": pre + "var ErrX error = &e{n: compute()}\n"},
			want:  []string{"a.go:9:package-level var ErrX"},
		},
		{
			name:  "typed blank nil exempt",
			files: map[string]string{"a.go": pre + "var _ error = nil\n"},
		},
	}
}

func errCompositeCases() []levelVarCase {
	const pre = "package p\n\nimport \"sync\"\n\ntype cfg struct{}\n\n"
	flagged := func(name, decl, sym string) levelVarCase {
		return levelVarCase{
			name:  name,
			files: map[string]string{"a.go": pre + decl + "\n"},
			want:  []string{"a.go:7:package-level var " + sym},
		}
	}
	return []levelVarCase{
		flagged("err named map flagged", "var errCache = map[string]int{}", "errCache"),
		flagged("err named slice flagged", "var errSeen = []string{}", "errSeen"),
		flagged("err named mutex flagged", "var errMu = sync.Mutex{}", "errMu"),
		flagged("err named once pointer flagged", "var errOnce = &sync.Once{}", "errOnce"),
		flagged("err digit named map flagged", "var err0 = map[int]int{}", "err0"),
		flagged("underscore named struct flagged", "var Err_Count = cfg{}", "Err_Count"),
		{
			name: "err named composite of an Err type exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype statusErr struct{ code int }\n\n" +
					"var errTransient = statusErr{code: 503}\n",
			},
		},
		{
			name: "err named composite of a package Error type exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/pkg\"\n\nvar ErrBad = pkg.ParseError{}\n",
			},
		},
		{
			name: "err named composite with a call flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype statusErr struct{ code int }\n\n" +
					"func load() int { return 1 }\n\n" +
					"var errLoaded = statusErr{code: load()}\n",
			},
			want: []string{"a.go:7:package-level var errLoaded"},
		},
	}
}

func constructorCases() []levelVarCase {
	const pre = "package p\n\nimport (\n\t\"errors\"\n\t\"fmt\"\n\t\"x/dberrs\"\n" +
		"\t\"x/prom\"\n)\n\n" +
		"type Counter struct{}\n\nfunc NewErrCounter() *Counter { return nil }\n\n" +
		"func newErr(s string) error { return nil }\n\nfunc load() string { return \"\" }\n\n"
	flag := func(name, decl, sym string) levelVarCase {
		return levelVarCase{
			name:  name,
			files: map[string]string{"a.go": pre + decl + "\n"},
			want:  []string{"a.go:18:package-level var " + sym},
		}
	}
	return []levelVarCase{
		flag("Errand is not an Err constructor", "var ErrCount = NewErrand()", "ErrCount"),
		flag(
			"newErrorLogger is not an Err constructor", "var errLog = newErrorLogger()", "errLog",
		),
		flag("local Err constructor returning a struct pointer flagged",
			"var errCounter = NewErrCounter()", "errCounter"),
		flag("package constructor without err in the import name flagged",
			"var errTotal = prom.NewErrCounterVec()", "errTotal"),
		flag("errors.New with a computed argument flagged",
			"var errLoad = errors.New(fmt.Sprint(load()))", "errLoad"),
		flag("fmt.Errorf with a call argument flagged",
			"var errFmt = fmt.Errorf(\"%d\", load())", "errFmt"),
		flag("NewErr with a call argument flagged", "var ErrLoad = newErr(load())", "ErrLoad"),
		{
			name: "nested constructor with simple arguments exempt",
			files: map[string]string{
				"a.go": pre + "var ErrNF = dberrs.NewErrDatabaseNotFound(\"lead\", \"c-1\", " +
					"errors.New(\"no rows\"))\n",
			},
		},
		{
			name:  "local constructor returning error exempt",
			files: map[string]string{"a.go": pre + "var ErrX = newErr(\"x\")\n"},
		},
		{
			name:  "binary and selector arguments exempt",
			files: map[string]string{"a.go": pre + "var ErrY = errors.New(\"a\" + prom.Name)\n"},
		},
	}
}

func TestAnalyzeSentinels(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(typedErrorCases(), errCompositeCases(), constructorCases()))
}
