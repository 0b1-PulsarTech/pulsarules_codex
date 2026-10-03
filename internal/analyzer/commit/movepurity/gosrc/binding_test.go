package gosrc

import "testing"

func TestBinding_Allows(t *testing.T) {
	t.Parallel()

	moves := []MoveInfo{{
		OldFull: "m/a", NewFull: "m/b", Decls: map[string]bool{"Foo": true},
	}}
	testCases := []struct {
		name string
		bind Binding
		qual string
		ref  string
		want bool
	}{
		{"unbound qualifier, known", Binding{}, "q", "X", true},
		{"unbound qualifier, unknown", Binding{IsUnknown: true}, "q", "X", false},
		{"same path", Binding{
			OldPaths: map[string]string{"q": "m/c"}, NewPaths: map[string]string{"q": "m/c"},
		}, "q", "X", true},
		{"same path, name moved away", Binding{
			OldPaths: map[string]string{"q": "m/a"}, NewPaths: map[string]string{"q": "m/a"},
			Moves: moves,
		}, "q", "Foo", false},
		{"same path, another name stays", Binding{
			OldPaths: map[string]string{"q": "m/a"}, NewPaths: map[string]string{"q": "m/a"},
			Moves: moves,
		}, "q", "Bar", true},
		{"rebound to the move target", Binding{
			OldPaths: map[string]string{"q": "m/a"}, NewPaths: map[string]string{"q": "m/b"},
			Moves: moves,
		}, "q", "Foo", true},
		{"rebound, name not moved", Binding{
			OldPaths: map[string]string{"q": "m/a"}, NewPaths: map[string]string{"q": "m/b"},
			Moves: moves,
		}, "q", "Bar", false},
		{"rebound without a move", Binding{
			OldPaths: map[string]string{"q": "m/a"}, NewPaths: map[string]string{"q": "m/z"},
			Moves: moves,
		}, "q", "Foo", false},
		{"newly bound qualifier", Binding{
			NewPaths: map[string]string{"q": "m/b"},
			Moves:    []MoveInfo{{NewFull: "m/b", Decls: map[string]bool{"Foo": true}}},
		}, "q", "Foo", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := testCase.bind.allows(testCase.qual, testCase.ref); got != testCase.want {
				t.Fatalf("allows() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestBinding_MovedAway(t *testing.T) {
	t.Parallel()

	bind := Binding{Moves: []MoveInfo{
		{OldFull: "m/a", NewFull: "m/b", Decls: map[string]bool{"Foo": true}},
		{OldFull: "m/c", NewFull: "m/c", Decls: map[string]bool{"Bar": true}},
	}}
	testCases := []struct {
		name       string
		importPath string
		ref        string
		want       bool
	}{
		{"declared and moved", "m/a", "Foo", true},
		{"not declared by the move", "m/a", "Baz", false},
		{"another package", "m/z", "Foo", false},
		{"the move keeps the path", "m/c", "Bar", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := bind.movedAway(testCase.importPath, testCase.ref); got != testCase.want {
				t.Fatalf("movedAway() = %v, want %v", got, testCase.want)
			}
		})
	}
}
