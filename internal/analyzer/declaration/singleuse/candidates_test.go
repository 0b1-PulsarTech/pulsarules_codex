package singleuse

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type exemptionsTestCase struct {
	name   string
	decl   Declaration
	kinds  declKinds
	params pass.ParamSet
	want   bool
}

func TestExemptions_Eligible(t *testing.T) {
	t.Parallel()

	for _, testCase := range exemptionsTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ex := exemptionsFromParams(testCase.params)
			if got := ex.eligible(testCase.decl, testCase.kinds); got != testCase.want {
				t.Fatalf("eligible = %v, want %v", got, testCase.want)
			}
		})
	}
}

func exemptionsTestCases() []exemptionsTestCase {
	return append(
		append(exemptionsBasicCases(), exemptionsHoistCases()...),
		exemptionsParamCases()...)
}

func exemptionsBasicCases() []exemptionsTestCase {
	bothKinds := declKinds{checkConst: true, checkVar: true}
	return []exemptionsTestCase{
		{
			name: "unexported literal var is eligible",
			decl: Declaration{
				Name: "magicNumber",
				Kind: DeclVar,
				Init: InitLiteral,
			},
			kinds: bothKinds,
			want:  true,
		},
		{
			name:  "unexported literal const is eligible",
			decl:  Declaration{Name: "magicNumber", Kind: DeclConst},
			kinds: bothKinds,
			want:  true,
		},
		{
			name: "exported identifier is never a candidate",
			decl: Declaration{
				Name: "Exported", Kind: DeclVar, Exported: true, Init: InitLiteral,
			},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "blank identifier is never a candidate",
			decl:  Declaration{Name: "_", Kind: DeclVar, Init: InitLiteral},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "const kind disabled by declKinds is skipped",
			decl:  Declaration{Name: "c", Kind: DeclConst},
			kinds: declKinds{checkConst: false, checkVar: true},
			want:  false,
		},
		{
			name:  "var kind disabled by declKinds is skipped",
			decl:  Declaration{Name: "v", Kind: DeclVar, Init: InitLiteral},
			kinds: declKinds{checkConst: true, checkVar: false},
			want:  false,
		},
		{
			name: "atomic-typed var must keep its identity across calls",
			decl: Declaration{
				Name:     "counter",
				Kind:     DeclVar,
				Init:     InitLiteral,
				TypeName: "atomic.Int64",
			},
			kinds: bothKinds,
			want:  false,
		},
		{
			name: "sync-typed var must keep its identity across calls",
			decl: Declaration{
				Name: "once", Kind: DeclVar, Init: InitLiteral, TypeName: "sync.Once",
			},
			kinds: bothKinds,
			want:  false,
		},
	}
}

func exemptionsHoistCases() []exemptionsTestCase {
	bothKinds := declKinds{checkConst: true, checkVar: true}
	return []exemptionsTestCase{
		{
			name:  "call-initialised var is hoist-exempt",
			decl:  Declaration{Name: "pattern", Kind: DeclVar, Init: InitCall},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "composite-literal-initialised var is hoist-exempt",
			decl:  Declaration{Name: "names", Kind: DeclVar, Init: InitComposite},
			kinds: bothKinds,
			want:  false,
		},
		{
			name: "go:embed annotation pins the var package-level",
			decl: Declaration{
				Name: "assets", Kind: DeclVar, Init: InitLiteral,
				Annotations: []string{"//go:embed assets"},
			},
			kinds: bothKinds,
			want:  false,
		},
	}
}

func exemptionsParamCases() []exemptionsTestCase {
	bothKinds := declKinds{checkConst: true, checkVar: true}
	return []exemptionsTestCase{
		{
			name: "a hypothetical language's pinned annotation is data, not code",
			decl: Declaration{
				Name: "cfg", Kind: DeclVar, Init: InitLiteral,
				Annotations: []string{"//dart:pin config"},
			},
			kinds:  bothKinds,
			params: pass.ParamSet{"pinned_annotations": []string{"//dart:pin"}},
			want:   false,
		},
		{
			name: "a hypothetical language's stateful type prefix is data, not code",
			decl: Declaration{
				Name:     "counter",
				Kind:     DeclVar,
				Init:     InitLiteral,
				TypeName: "isolate.Counter",
			},
			kinds:  bothKinds,
			params: pass.ParamSet{"stateful_type_prefixes": []string{"isolate."}},
			want:   false,
		},
		{
			name:   "overriding hoisted_init_kinds stops exempting call-initialised vars",
			decl:   Declaration{Name: "pattern", Kind: DeclVar, Init: InitCall},
			kinds:  bothKinds,
			params: pass.ParamSet{"hoisted_init_kinds": []string{"none"}},
			want:   true,
		},
	}
}
