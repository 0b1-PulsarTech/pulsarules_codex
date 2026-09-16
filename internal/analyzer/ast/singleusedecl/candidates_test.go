package singleusedecl

import (
	"testing"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type exemptionsTestCase struct {
	name   string
	decl   core.Declaration
	kinds  declKinds
	params core.ParamSet
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
			decl: core.Declaration{
				Name: "magicNumber",
				Kind: core.DeclVar,
				Init: core.InitLiteral,
			},
			kinds: bothKinds,
			want:  true,
		},
		{
			name:  "unexported literal const is eligible",
			decl:  core.Declaration{Name: "magicNumber", Kind: core.DeclConst},
			kinds: bothKinds,
			want:  true,
		},
		{
			name: "exported identifier is never a candidate",
			decl: core.Declaration{
				Name: "Exported", Kind: core.DeclVar, Exported: true, Init: core.InitLiteral,
			},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "blank identifier is never a candidate",
			decl:  core.Declaration{Name: "_", Kind: core.DeclVar, Init: core.InitLiteral},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "const kind disabled by declKinds is skipped",
			decl:  core.Declaration{Name: "c", Kind: core.DeclConst},
			kinds: declKinds{checkConst: false, checkVar: true},
			want:  false,
		},
		{
			name:  "var kind disabled by declKinds is skipped",
			decl:  core.Declaration{Name: "v", Kind: core.DeclVar, Init: core.InitLiteral},
			kinds: declKinds{checkConst: true, checkVar: false},
			want:  false,
		},
		{
			name: "atomic-typed var must keep its identity across calls",
			decl: core.Declaration{
				Name:     "counter",
				Kind:     core.DeclVar,
				Init:     core.InitLiteral,
				TypeName: "atomic.Int64",
			},
			kinds: bothKinds,
			want:  false,
		},
		{
			name: "sync-typed var must keep its identity across calls",
			decl: core.Declaration{
				Name: "once", Kind: core.DeclVar, Init: core.InitLiteral, TypeName: "sync.Once",
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
			decl:  core.Declaration{Name: "pattern", Kind: core.DeclVar, Init: core.InitCall},
			kinds: bothKinds,
			want:  false,
		},
		{
			name:  "composite-literal-initialised var is hoist-exempt",
			decl:  core.Declaration{Name: "names", Kind: core.DeclVar, Init: core.InitComposite},
			kinds: bothKinds,
			want:  false,
		},
		{
			name: "go:embed annotation pins the var package-level",
			decl: core.Declaration{
				Name: "assets", Kind: core.DeclVar, Init: core.InitLiteral,
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
			decl: core.Declaration{
				Name: "cfg", Kind: core.DeclVar, Init: core.InitLiteral,
				Annotations: []string{"//dart:pin config"},
			},
			kinds:  bothKinds,
			params: core.ParamSet{"pinned_annotations": []string{"//dart:pin"}},
			want:   false,
		},
		{
			name: "a hypothetical language's stateful type prefix is data, not code",
			decl: core.Declaration{
				Name:     "counter",
				Kind:     core.DeclVar,
				Init:     core.InitLiteral,
				TypeName: "isolate.Counter",
			},
			kinds:  bothKinds,
			params: core.ParamSet{"stateful_type_prefixes": []string{"isolate."}},
			want:   false,
		},
		{
			name:   "overriding hoisted_init_kinds stops exempting call-initialised vars",
			decl:   core.Declaration{Name: "pattern", Kind: core.DeclVar, Init: core.InitCall},
			kinds:  bothKinds,
			params: core.ParamSet{"hoisted_init_kinds": []string{"none"}},
			want:   true,
		},
	}
}
