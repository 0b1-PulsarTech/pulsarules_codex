package packagelevelvar

import (
	"slices"
	"testing"
)

const hdr = "package p\n\nimport (\n\t\"errors\"\n\t\"fmt\"\n)\n\n"

func exemptCases() []levelVarCase {
	return []levelVarCase{
		{
			name:  "errors.New sentinel exempt",
			files: map[string]string{"a.go": hdr + "var ErrX = errors.New(\"x\")\n"},
		},
		{
			name:  "fmt.Errorf sentinel exempt",
			files: map[string]string{"a.go": hdr + "var errX = fmt.Errorf(\"x\")\n"},
		},
		{
			name: "mixed error constructors exempt",
			files: map[string]string{
				"a.go": hdr + "var a, b = errors.New(\"a\"), fmt.Errorf(\"b\")\n",
			},
		},
		{
			name: "non error call mixed in is flagged",
			files: map[string]string{
				"a.go": hdr + "func f() error { return nil }\n\n" +
					"var a, b = errors.New(\"a\"), f()\n",
			},
			want: []string{"a.go:10:package-level var b"},
		},
		{
			name: "typed error exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype nf struct{}\n\n" +
					"func (*nf) Error() string { return \"\" }\n\n" +
					"var ErrNotFound error = &nf{}\n",
			},
		},
		{
			name: "re-exported sentinel exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/access\"\n\n" +
					"var ErrForbidden = access.ErrForbidden\n",
			},
		},
		{
			name: "lowercase err selector exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/access\"\n\nvar errDenied = access.errDenied\n",
			},
		},
		{
			name: "non error selector flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/cfg\"\n\nvar limit = cfg.Limit\n",
			},
			want: []string{"a.go:5:package-level var limit"},
		},
	}
}

func nameDecisionCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "sentinel beside a counter flags only the counter",
			files: map[string]string{
				"a.go": hdr + "var ErrA, cnt = errors.New(\"a\"), 0\n",
			},
			want: []string{"a.go:8:package-level var cnt"},
		},
		{
			name: "nested constructor in a sentinel exempt",
			files: map[string]string{
				"a.go": hdr + "func newErr(string) error { return nil }\n\n" +
					"var ErrW = fmt.Errorf(\"w: %w\", newErr(\"x\"))\n",
			},
		},
		{
			name: "pointer typed error constructor exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype MyError struct{}\n\n" +
					"func newErr(string) *MyError { return nil }\n\n" +
					"var ErrX = newErr(\"x\")\n",
			},
		},
		{
			name: "constructor with two results flagged",
			files: map[string]string{
				"a.go": "package p\n\nfunc newErr(string) (a, b error) { return }\n\n" +
					"var ErrX = newErr(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
		{
			name: "constructor returning a look-alike type flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype Terror struct{}\n\n" +
					"func newErr(string) *Terror { return nil }\n\n" +
					"var ErrX = newErr(\"x\")\n",
			},
			want: []string{"a.go:7:package-level var ErrX"},
		},
		{
			name: "composite of a look-alike type flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype Terror struct{}\n\nvar ErrX = Terror{}\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
		{
			name: "composite of a type ending in lowercase err flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype berr struct{}\n\nvar ErrX = berr{}\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
		{
			name: "composite of a lowercase err type exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype err struct{}\n\nvar ErrX = err{}\n",
			},
		},
	}
}

func sentinelFormCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "error typed sentinel exempt",
			files: map[string]string{
				"a.go": hdr + "var ErrX error = errors.New(\"x\")\n",
			},
		},
		{
			name: "multi value constructor into two names flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/apperr\"\n\n" +
					"var ErrA, ErrB = apperr.NewErrPair(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var ErrA", "a.go:5:package-level var ErrB"},
		},
		{
			name: "aliased errors package exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport stderrors \"errors\"\n\n" +
					"var ErrX = stderrors.New(\"x\")\n",
			},
		},
		{
			name: "parenthesized constructor exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"errors\"\n\nvar ErrX = (errors.New(\"x\"))\n",
			},
		},
		{
			name: "errors package not imported is flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport errors \"x/myerrors\"\n\n" +
					"var ErrX = errors.Make(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
		{
			name: "errors.Join flagged",
			files: map[string]string{
				"a.go": hdr + "var ErrBoth = errors.Join(errors.New(\"a\"), errors.New(\"b\"))\n",
			},
			want: []string{"a.go:8:package-level var ErrBoth"},
		},
		{
			name: "err named var from a non err constructor flagged",
			files: map[string]string{
				"a.go": "package p\n\nvar ErrX = build()\n\nfunc build() error { return nil }\n",
			},
			want: []string{"a.go:3:package-level var ErrX"},
		},
		{
			name: "err named var from an Err constructor exempt",
			files: map[string]string{
				"a.go": "package p\n\nvar ErrX = newErr()\n\nfunc newErr() error { return nil }\n",
			},
		},
		{
			name: "blank import does not bind the errors package",
			files: map[string]string{
				"a.go": "package p\n\nimport _ \"errors\"\n\nvar ErrX = errors.New(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
		{
			name: "dot import does not bind the errors package",
			files: map[string]string{
				"a.go": "package p\n\nimport . \"errors\"\n\nvar ErrX = errors.New(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var ErrX"},
		},
	}
}

func sentinelIntentCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "composite literal err value exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype statusErr struct{ code int }\n\n" +
					"func (statusErr) Error() string { return \"\" }\n\n" +
					"var errTransient = statusErr{code: 503}\n",
			},
		},
		{
			name: "address of composite err value exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype statusErr struct{}\n\nvar ErrBusy = &statusErr{}\n",
			},
		},
		{
			name: "composite under a non err name flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype cfg struct{}\n\nvar defaults = cfg{}\n",
			},
			want: []string{"a.go:5:package-level var defaults"},
		},
		{
			name: "composite under errors-like name without boundary flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype cfg struct{}\n\nvar errorsTotal = cfg{}\n",
			},
			want: []string{"a.go:5:package-level var errorsTotal"},
		},
		{
			name: "NewErr callee exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/dberrs\"\n\n" +
					"var errLeadNotFound = dberrs.NewErrDatabaseNotFound(\"lead\")\n",
			},
		},
		{
			name: "err named var from an unrelated call flagged",
			files: map[string]string{
				"a.go": "package p\n\nfunc computeCount() int { return 0 }\n\n" +
					"var errCount = computeCount()\n",
			},
			want: []string{"a.go:5:package-level var errCount"},
		},
	}
}

func selectorSentinelCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "re-exported sentinel exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"x/store\"\n\nvar ErrNotFound = store.ErrNotFound\n",
			},
		},
		{
			name: "slog Error function flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"log/slog\"\n\nvar logErr = slog.Error\n",
			},
			want: []string{"a.go:5:package-level var logErr"},
		},
		{
			name: "http Error function flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"net/http\"\n\nvar writeErr = http.Error\n",
			},
			want: []string{"a.go:5:package-level var writeErr"},
		},
		{
			name: "selector on a local var flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype m struct{ errTotal int }\n\nvar metrics m\n\n" +
					"var n = metrics.errTotal\n",
			},
			want: []string{"a.go:5:package-level var metrics", "a.go:7:package-level var n"},
		},
	}
}

func blankCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "typed blank assertion exempt",
			files: map[string]string{
				"a.go": "package p\n\nvar _ error = (*E)(nil)\n\ntype E struct{}\n",
			},
		},
		{
			name: "pointer conversion assertion exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype E struct{}\n\nvar _ = (*E)(nil)\n",
			},
		},
		{
			name:  "composite blank exempt",
			files: map[string]string{"a.go": "package p\n\ntype E struct{}\n\nvar _ = E{}\n"},
		},
		{
			name:  "address of composite blank exempt",
			files: map[string]string{"a.go": "package p\n\ntype E struct{}\n\nvar _ = &E{}\n"},
		},
		{
			name: "conversion of composite blank exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype E struct{}\ntype I interface{}\n\nvar _ = I(E{})\n",
			},
		},
		{
			name: "blank beside a typed sentinel holding a sentinel exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"errors\"\n\n" +
					"var _, ErrA error = errors.New(\"x\"), errors.New(\"a\")\n",
			},
		},
	}
}

func typedReexportCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "typed re-export under an err name exempt",
			files: map[string]string{
				"a.go": "package p\n\nimport \"io\"\n\nvar ErrEOF error = io.EOF\n",
			},
		},
		{
			name: "typed re-export under a non err name flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"os\"\n\nvar current error = os.ErrClosed\n",
			},
			want: []string{"a.go:5:package-level var current"},
		},
		{
			name: "typed re-export under a bare Err name flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"os\"\n\nvar Err error = os.ErrClosed\n",
			},
			want: []string{"a.go:5:package-level var Err"},
		},
		{
			name: "untyped re-export of a non err selector flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"io\"\n\nvar ErrEOF = io.EOF\n",
			},
			want: []string{"a.go:5:package-level var ErrEOF"},
		},
		{
			name: "typed selector on a local var flagged",
			files: map[string]string{
				"a.go": "package p\n\nvar cfg struct{ ErrF error }\n\nvar ErrX error = cfg.ErrF\n",
			},
			want: []string{"a.go:3:package-level var cfg", "a.go:5:package-level var ErrX"},
		},
		{
			name: "untyped selector on a local var flagged",
			files: map[string]string{
				"a.go": "package p\n\nvar cfg struct{ ErrF error }\n\nvar ErrX = cfg.ErrF\n",
			},
			want: []string{"a.go:3:package-level var cfg", "a.go:5:package-level var ErrX"},
		},
	}
}

func strictShapeCases() []levelVarCase {
	return []levelVarCase{
		{
			name: "typed error composite with a call flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype MyErr struct{ v int }\n\n" +
					"func load() int { return 0 }\n\n" +
					"var ErrX error = MyErr{v: load()}\n",
			},
			want: []string{"a.go:7:package-level var ErrX"},
		},
		{
			name: "typed error under a non err name from a composite flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype state struct{}\n\nvar current error = &state{}\n",
			},
			want: []string{"a.go:5:package-level var current"},
		},
		{
			name: "typed error under a non err name from a plain composite flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype state struct{}\n\nvar current error = state{}\n",
			},
			want: []string{"a.go:5:package-level var current"},
		},
		{
			name: "typed error under an err name from a composite exempt",
			files: map[string]string{
				"a.go": "package p\n\ntype state struct{}\n\nvar ErrState error = &state{}\n",
			},
		},
		{
			name: "typed error under a non err name from a conversion flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype state struct{}\n\nvar current error = (*state)(nil)\n",
			},
			want: []string{"a.go:5:package-level var current"},
		},
		{
			name: "address of err composite under a non err name flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype MyErr struct{}\n\nvar cache = &MyErr{}\n",
			},
			want: []string{"a.go:5:package-level var cache"},
		},
		{
			name: "Err constructor under a non err name flagged",
			files: map[string]string{
				"a.go": "package p\n\nfunc NewErrCounter(s string) error { return nil }\n\n" +
					"var counter = NewErrCounter(\"x\")\n",
			},
			want: []string{"a.go:5:package-level var counter"},
		},
		{
			name: "Err constructor on a local var flagged",
			files: map[string]string{
				"a.go": "package p\n\ntype F struct{}\n\nvar errFactory F\n\n" +
					"var ErrX = errFactory.NewErrX(\"x\")\n",
			},
			want: []string{
				"a.go:5:package-level var errFactory", "a.go:7:package-level var ErrX",
			},
		},
		{
			name: "untyped re-export under a non err name flagged",
			files: map[string]string{
				"a.go": "package p\n\nimport \"net/http\"\n\nvar handler = http.ErrHandler\n",
			},
			want: []string{"a.go:5:package-level var handler"},
		},
	}
}

func TestAnalyzeExempt(t *testing.T) {
	t.Parallel()

	runCases(t, slices.Concat(
		exemptCases(),
		nameDecisionCases(),
		sentinelFormCases(),
		blankCases(),
		sentinelIntentCases(),
		selectorSentinelCases(),
		strictShapeCases(),
		typedReexportCases(),
	))
}
