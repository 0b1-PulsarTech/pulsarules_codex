package singleuse

import "testing"

type eligibleTestCase struct {
	name string
	decl Declaration
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
			decl: Declaration{
				Name: "magicNumber",
				Kind: DeclVar,
				Init: InitLiteral,
			},
			want: true,
		},
		{
			name: "unexported literal const is eligible",
			decl: Declaration{Name: "magicNumber", Kind: DeclConst},
			want: true,
		},
		{
			name: "exported identifier is never a candidate",
			decl: Declaration{
				Name: "Exported", Kind: DeclVar, Exported: true, Init: InitLiteral,
			},
			want: false,
		},
		{
			name: "blank identifier is never a candidate",
			decl: Declaration{Name: "_", Kind: DeclVar, Init: InitLiteral},
			want: false,
		},
		{
			name: "atomic-typed var must keep its identity across calls",
			decl: Declaration{
				Name:     "counter",
				Kind:     DeclVar,
				Init:     InitLiteral,
				TypeName: "atomic.Int64",
			},
			want: false,
		},
		{
			name: "sync-typed var must keep its identity across calls",
			decl: Declaration{
				Name: "once", Kind: DeclVar, Init: InitLiteral, TypeName: "sync.Once",
			},
			want: false,
		},
	}
}

func eligibleHoistCases() []eligibleTestCase {
	return []eligibleTestCase{
		{
			name: "call-initialised var is hoist-exempt",
			decl: Declaration{Name: "pattern", Kind: DeclVar, Init: InitCall},
			want: false,
		},
		{
			name: "composite-literal-initialised var is hoist-exempt",
			decl: Declaration{Name: "names", Kind: DeclVar, Init: InitComposite},
			want: false,
		},
		{
			name: "go:embed annotation pins the var package-level",
			decl: Declaration{
				Name: "assets", Kind: DeclVar, Init: InitLiteral,
				Annotations: []string{"//go:embed assets"},
			},
			want: false,
		},
	}
}
