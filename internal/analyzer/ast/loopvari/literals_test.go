package loopvari

import "testing"

func TestKeysAreValues(t *testing.T) {
	t.Parallel()

	testCases := []loopCase{
		{
			name:   "map literal key is a use",
			body:   "for i := range s {\n _ = map[int]int{i: s[i]}\n}",
			expect: 1,
		},
		{
			name:   "elided map in a slice of maps",
			body:   "for i := range s {\n _ = []map[int]int{{i: s[i]}}\n}",
			expect: 1,
		},
		{
			name:   "elided map behind a pointer in a slice",
			body:   "for i := range s {\n _ = []*map[int]bool{{i: true}}\n}",
			expect: 1,
		},
		{
			name:  "elided struct in a slice keeps its keys",
			body:  "for i := range s {\n _ = []T{{i: s[i]}}\n}",
			decls: "type T struct{ i int }",
		},
		{
			name:   "elided map in a map of maps",
			body:   "for i := range s {\n _ = map[int]map[int]int{0: {i: s[i]}}\n}",
			expect: 1,
		},
		{
			name:   "doubly elided map in nested slices",
			body:   "for i := range s {\n _ = [][]map[int]int{{{i: 1}}}\n}",
			expect: 1,
		},
		{
			name:  "doubly elided struct keeps its keys",
			body:  "for i := range s {\n _ = [][]T{{{i: 1}}}\n _ = s[i]\n}",
			decls: "type T struct{ i int }",
		},
		{
			name:  "struct key of a map holding pointers to maps",
			body:  "for i := range s {\n _ = map[st]*map[int]bool{{i: 1}: nil}\n _ = s[i]\n}",
			decls: "type st struct{ i int }",
		},
		{
			name:  "struct key of a map holding maps",
			body:  "for i := range s {\n _ = map[st]map[int]bool{{i: 1}: nil}\n _ = s[i]\n}",
			decls: "type st struct{ i int }",
		},
		{
			name:   "elided map behind a pointer in a map key",
			body:   "for i := range s {\n _ = map[*map[int]bool]int{{i: true}: 1}\n}",
			expect: 1,
		},
		{
			name:   "pointer at an inner elided level",
			body:   "for i := range s {\n _ = []*[]map[int]bool{{{i: true}}}\n}",
			expect: 1,
		},
		{
			name:   "elided map in a parenthesised element type",
			body:   "for i := range s {\n _ = [](map[int]int){{i: 1}}\n}",
			expect: 1,
		},
		{
			name:   "elided map behind a parenthesised pointer",
			body:   "for i := range s {\n _ = []*(map[int]int){{i: 1}}\n}",
			expect: 1,
		},
		{
			name:   "elided map key reached through an array key",
			body:   "for i := range s {\n _ = map[[1]*map[int]int]int{{{i: 1}}: 0}\n _ = s[i]\n}",
			expect: 1,
		},
	}
	runLoopCases(t, testCases)
}
