package gosrc

import (
	"testing"
)

func TestCaptureSets_Captures(t *testing.T) {
	t.Parallel()

	sets := CaptureSets{
		Left:  map[string]bool{"max": true, "Foo": true},
		Taken: map[string]bool{"Bar": true},
	}
	testCases := []struct {
		name string
		ref  string
		want bool
	}{
		{"taken in the new directory", "Bar", true},
		{"left behind and predeclared", "max", true},
		{"left behind only", "Foo", false},
		{"neither", "Baz", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := sets.captures(testCase.ref); got != testCase.want {
				t.Fatalf("captures(%q) = %v, want %v", testCase.ref, got, testCase.want)
			}
		})
	}
}

func TestCapturesBare(t *testing.T) {
	t.Parallel()

	file := mustParse(t, "package p\n\nfunc F() int { return Foo(1) + local }\n\nvar local = 1\n")
	taken := Binding{Captures: CaptureSets{Taken: map[string]bool{"Foo": true, "local": true}}}
	testCases := []struct {
		name string
		lit  string
		bind Binding
		want bool
	}{
		{"unresolved and taken", "Foo", taken, true},
		{"unresolved and free", "Foo", Binding{}, false},
		{"resolved in the file", "local", taken, false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			pos := -1
			for idx, tok := range file.body {
				if tok.lit == testCase.lit {
					pos = idx
					break
				}
			}
			if pos < 0 {
				t.Fatalf("no token %q", testCase.lit)
			}
			if got := capturesBare(file, pos, testCase.bind); got != testCase.want {
				t.Fatalf("capturesBare() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestRelocationUnsafe(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		body string
		want bool
	}{
		{"plain declarations", "func F() int { return 1 }\n", false},
		{"TestMain is not load work", "func TestMain(m *testing.M) { m.Run() }\n", false},
		{"init is not unsafe by itself", "func init() {}\n", false},
		{"dot import", "import . \"strings\"\n\nvar V = ToUpper(\"a\")\n", true},
		{"cgo", "import \"C\"\n\nvar V = C.VAL\n", true},
		{"predeclared name", "func max(a, b int) int { return a }\n", true},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			file := mustParse(t, "package p\n\n"+testCase.body)
			if got := RelocationUnsafe(file, file); got != testCase.want {
				t.Fatalf("RelocationUnsafe() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestRelocationUnsafe_OldFileImports(t *testing.T) {
	t.Parallel()

	cgo := mustParse(t, "package p\n\nimport \"C\"\n")
	plain := mustParse(t, "package p\n")
	if !RelocationUnsafe(cgo, plain) || !RelocationUnsafe(plain, cgo) {
		t.Fatal("an import C on either side must be unsafe")
	}
}
