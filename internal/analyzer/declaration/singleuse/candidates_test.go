package singleuse

import "testing"

type eligibleTestCase struct {
	name string
	decl declaration
	want bool
}

func TestEligible(t *testing.T) {
	t.Parallel()

	for _, testCase := range eligibleTestCases() {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := eligible(testCase.decl); got != testCase.want {
				t.Fatalf("eligible = %v, want %v", got, testCase.want)
			}
		})
	}
}

func eligibleTestCases() []eligibleTestCase {
	return append(eligibleBasicCases(), eligibleHoistCases()...)
}

func eligibleBasicCases() []eligibleTestCase {
	return []eligibleTestCase{
		{
			name: "unexported literal var is eligible",
			decl: declaration{
				Name: "magicNumber",
				Kind: declVar,
				Init: initLiteral,
			},
			want: true,
		},
		{
			name: "unexported literal const is eligible",
			decl: declaration{Name: "magicNumber", Kind: declConst},
			want: true,
		},
		{
			name: "exported identifier is never a candidate",
			decl: declaration{
				Name: "Exported", Kind: declVar, Exported: true, Init: initLiteral,
			},
			want: false,
		},
		{
			name: "blank identifier is never a candidate",
			decl: declaration{Name: "_", Kind: declVar, Init: initLiteral},
			want: false,
		},
		{
			name: "atomic-typed var must keep its identity across calls",
			decl: declaration{
				Name:     "counter",
				Kind:     declVar,
				Init:     initLiteral,
				TypeName: "atomic.Int64",
			},
			want: false,
		},
		{
			name: "sync-typed var must keep its identity across calls",
			decl: declaration{
				Name: "once", Kind: declVar, Init: initLiteral, TypeName: "sync.Once",
			},
			want: false,
		},
	}
}

func eligibleHoistCases() []eligibleTestCase {
	return []eligibleTestCase{
		{
			name: "call-initialised var is hoist-exempt",
			decl: declaration{Name: "pattern", Kind: declVar, Init: initCall},
			want: false,
		},
		{
			name: "composite-literal-initialised var is hoist-exempt",
			decl: declaration{Name: "names", Kind: declVar, Init: initComposite},
			want: false,
		},
		{
			name: "go:embed annotation pins the var package-level",
			decl: declaration{
				Name: "assets", Kind: declVar, Init: initLiteral,
				Annotations: []string{"//go:embed assets"},
			},
			want: false,
		},
	}
}
