package dartsrc

import (
	"errors"
	"slices"
	"testing"
)

type parseCase struct {
	name       string
	src        string
	wantKinds  []string
	wantURIs   [][]string
	wantBody   string
	wantShapes []string
}

func TestParse(t *testing.T) {
	t.Parallel()

	runParseCases(t, []parseCase{
		{
			name:      "plain imports and exports",
			src:       "import 'a.dart';\nexport \"b.dart\";\n\nclass Draft {}\n",
			wantKinds: []string{"import", "export"},
			wantURIs:  [][]string{{"a.dart"}, {"b.dart"}},
			wantBody:  "\n\n\nclass Draft {}\n",
		},
		{
			name:      "conditional import yields every uri",
			src:       "import 'a.dart' if (dart.library.io) 'b.dart' if (dart.library.html == 'true') 'c.dart';\n",
			wantKinds: []string{"import"},
			wantURIs:  [][]string{{"a.dart", "b.dart", "c.dart"}},
			wantShapes: []string{
				"import \x00 if (dart.library.io) \x00 if (dart.library.html == 'true') \x00;",
			},
			wantBody: "\n",
		},
	})
}

func TestParse_Boundaries(t *testing.T) {
	t.Parallel()

	runParseCases(t, []parseCase{
		{
			name:      "part and part of",
			src:       "library scale;\npart 'a.g.dart';\npart of 'main.dart';\npart of scale;\n",
			wantKinds: []string{"library", "part", "part of", "part of"},
			wantURIs:  [][]string{nil, {"a.g.dart"}, {"main.dart"}, nil},
			wantBody:  "\n\n\n\n",
		},
		{
			name:      "comments and a shebang stay in the body",
			src:       "#!/usr/bin/env dart\n// import 'x.dart';\nimport /* why */ 'a.dart'; // tail\nvoid main() {}\n",
			wantKinds: []string{"import"},
			wantURIs:  [][]string{{"a.dart"}},
			wantBody:  "#!/usr/bin/env dart\n// import 'x.dart';\n // tail\nvoid main() {}\n",
		},
		{
			name:      "imports after code are body",
			src:       "import 'a.dart';\nvoid f() {}\nimport 'b.dart';\n",
			wantKinds: []string{"import"},
			wantURIs:  [][]string{{"a.dart"}},
			wantBody:  "\nvoid f() {}\nimport 'b.dart';\n",
		},
		{
			name:      "library called like a function stays in the body",
			src:       "library() {}\n",
			wantKinds: []string{},
			wantURIs:  [][]string{},
			wantBody:  "library() {}\n",
		},
		{
			name:      "export called like a function stays in the body",
			src:       "export() => 'a.dart';\n",
			wantKinds: []string{},
			wantURIs:  [][]string{},
			wantBody:  "export() => 'a.dart';\n",
		},
		{
			name:      "import called like a function stays in the body",
			src:       "import() => 'a.dart';\n",
			wantKinds: []string{},
			wantURIs:  [][]string{},
			wantBody:  "import() => 'a.dart';\n",
		},
		{
			name:      "part called like a function stays in the body",
			src:       "part(x) => 'b.dart';\n",
			wantKinds: []string{},
			wantURIs:  [][]string{},
			wantBody:  "part(x) => 'b.dart';\n",
		},
		{
			name:      "part of with a callable stays in the body",
			src:       "part of() => 1;\n",
			wantKinds: []string{},
			wantURIs:  [][]string{},
			wantBody:  "part of() => 1;\n",
		},
		{
			name:      "raw triple quoted",
			src:       "import r'''a.dart''';\n",
			wantKinds: []string{"import"},
			wantURIs:  [][]string{{"a.dart"}},
			wantBody:  "\n",
		},
	})
}

func runParseCases(t *testing.T, testCases []parseCase) {
	t.Helper()
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Parse(testCase.src)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			kinds := make([]string, 0, len(got.Directives))
			uris := make([][]string, 0, len(got.Directives))
			for _, directive := range got.Directives {
				kinds = append(kinds, directive.Kind)
				uris = append(uris, directive.URIs)
			}
			if !slices.Equal(kinds, testCase.wantKinds) {
				t.Fatalf("kinds = %q, want %q", kinds, testCase.wantKinds)
			}
			if !slices.EqualFunc(uris, testCase.wantURIs, slices.Equal) {
				t.Fatalf("uris = %q, want %q", uris, testCase.wantURIs)
			}
			if got.Body != testCase.wantBody {
				t.Fatalf("body = %q, want %q", got.Body, testCase.wantBody)
			}
			for position, shape := range testCase.wantShapes {
				if got.Directives[position].Shape != shape {
					t.Fatalf("shape = %q, want %q", got.Directives[position].Shape, shape)
				}
			}
		})
	}
}

func TestParse_Malformed(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		src  string
	}{
		{name: "unterminated directive", src: "import 'a.dart'\n"},
		{name: "unterminated string", src: "import 'a.dart;\n"},
		{name: "interpolated uri", src: "import 'a$b.dart';\n"},
		{name: "escaped uri", src: "import 'a\\'b.dart';\n"},
		{name: "adjacent literals", src: "import 'a''b.dart';\n"},
		{name: "second literal outside a condition", src: "import 'a.dart' 'b.dart';\n"},
		{name: "second literal after a clause", src: "import 'a.dart' show A 'b.dart';\n"},
		{
			name: "second literal after a clause",
			src:  "import 'a.dart' if (dart.library.io) deferred 'b.dart';\n",
		},
		{name: "part of name then literal", src: "part of scale 'b.dart';\n"},
		{name: "percent in a uri", src: "import 'a%2Fb.dart';\n"},
		{name: "query in a uri", src: "import 'a.dart?x=1';\n"},
		{name: "fragment in a uri", src: "import 'a.dart#x';\n"},
		{name: "backslash in a uri", src: "import 'a\\\\b.dart';\n"},
		{name: "package dot segment", src: "import 'package:scale/./a.dart';\n"},
		{name: "package dotdot segment", src: "import 'package:scale/../a.dart';\n"},
		{name: "package empty segment", src: "import 'package:scale//a.dart';\n"},
		{name: "package without a path", src: "import 'package:scale';\n"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if _, err := Parse(testCase.src); !errors.Is(err, ErrMalformed) {
				t.Fatalf("Parse() error = %v, want ErrMalformed", err)
			}
		})
	}
}
