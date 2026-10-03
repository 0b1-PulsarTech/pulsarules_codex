package gosrc

import (
	"testing"
)

func TestInclusionOf(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		file string
		want inclusion
	}{
		{"plain source", "a/x.go", inclusion{ext: ".go"}},
		{"test file", "a/x_test.go", inclusion{ext: ".go", isTest: true}},
		{"os suffix", "a/x_windows.go", inclusion{ext: ".go", tags: "windows"}},
		{"arch suffix", "a/x_arm64.go", inclusion{ext: ".go", tags: "arm64"}},
		{"os and arch suffix", "a/x_linux_amd64.go", inclusion{ext: ".go", tags: "linux_amd64"}},
		{
			"os suffix of a test",
			"a/x_linux_test.go",
			inclusion{ext: ".go", isTest: true, tags: "linux"},
		},
		{"wasip1 os suffix", "a/x_wasip1.go", inclusion{ext: ".go", tags: "wasip1"}},
		{"loong64 arch suffix", "a/x_loong64.go", inclusion{ext: ".go", tags: "loong64"}},
		{
			"os and arch with new names",
			"a/x_wasip1_loong64.go",
			inclusion{ext: ".go", tags: "wasip1_loong64"},
		},
		{"multi-dot generated name", "a/x_linux.pb.go", inclusion{ext: ".go", tags: "linux"}},
		{"unknown suffix", "a/x_foo.go", inclusion{ext: ".go"}},
		{"arch then unknown", "a/x_amd64_foo.go", inclusion{ext: ".go"}},
		{"no underscore word", "a/windows.go", inclusion{ext: ".go"}},
		{"test suffix on a c file", "a/x_test.c", inclusion{ext: ".c"}},
		{"leading underscore", "a/_x.go", inclusion{ext: ".go", ignored: true}},
		{"leading dot", "a/.x.go", inclusion{ext: ".go", ignored: true}},
		{"testdata dir", "a/testdata/x.go", inclusion{ext: ".go", ignored: true}},
		{"root vendor dir", "vendor/m/x.go", inclusion{ext: ".go", vendored: true}},
		{"nested vendor dir", "a/vendor/m/x.go", inclusion{ext: ".go"}},
		{"underscore dir", "a/_gen/x.go", inclusion{ext: ".go", ignored: true}},
		{"dot dir", ".github/x.go", inclusion{ext: ".go", ignored: true}},
		{"repo root", "x.go", inclusion{ext: ".go"}},
		{"backup suffix", "a/plugin.go.orig", inclusion{}},
		{"assembly", "a/x_amd64.s", inclusion{ext: ".s", tags: "amd64"}},
		{"upper-case assembly", "a/x.S", inclusion{ext: ".S"}},
		{"system object", "a/x_linux_amd64.syso", inclusion{ext: ".syso", tags: "linux_amd64"}},
		{"extra assembly", "a/x.sx", inclusion{ext: ".sx"}},
		{"c++ source", "a/x.cc", inclusion{ext: ".cc"}},
		{"objective c", "a/x.m", inclusion{ext: ".m"}},
		{"extra header", "a/x.hxx", inclusion{ext: ".hxx"}},
		{"fortran upper", "a/x.F", inclusion{ext: ".F"}},
		{"fortran for", "a/x.for", inclusion{ext: ".for"}},
		{"fortran 90", "a/x.f90", inclusion{ext: ".f90"}},
		{"swig", "a/x.swig", inclusion{ext: ".swig"}},
		{"swig c++", "a/x.swigcxx", inclusion{ext: ".swigcxx"}},
		{"data file under testdata", "a/testdata/x_windows.json", inclusion{}},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := inclusionOf(testCase.file); got != testCase.want {
				t.Fatalf("InclusionOf(%q) = %+v, want %+v", testCase.file, got, testCase.want)
			}
		})
	}
}

func TestSameInclusion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		oldPath string
		newPath string
		want    bool
	}{
		{"another name", "a/x.go", "a/y.go", true},
		{"into a test file", "a/x.go", "a/x_test.go", false},
		{"into an os file", "a/x.go", "a/x_linux.go", false},
		{"into an ignored dir", "a/x.go", "a/testdata/x.go", false},
		{"out of the toolchain", "a/x.go", "a/x.go.orig", false},
		{"a c file into a _test suffix", "a/seven.c", "a/seven_test.c", true},
		{"a syso across directories", "a/x.syso", "b/x.syso", true},
		{"a syso into an os file", "a/x.syso", "a/x_linux.syso", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := SameInclusion(testCase.oldPath, testCase.newPath); got != testCase.want {
				t.Fatalf("SameInclusion() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestIsCompiled(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		file string
		want bool
	}{
		{
			"a/x.c",
			true,
		},
		{"a/x.syso", true},
		{"a/x.go", true},
		{"a/README.md", false},
		{"a/x", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()

			if got := IsCompiled(testCase.file); got != testCase.want {
				t.Fatalf("IsCompiled(%q) = %v, want %v", testCase.file, got, testCase.want)
			}
		})
	}
}

func TestIsOutsidePackage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		file string
		want bool
	}{
		{"a/testdata/x.c", true},
		{"vendor/m/x.go", false},
		{"vendor/x.go", false},
		{"a/vendor/m/x.go", false},
		{"vendor/m/testdata/x.go", true},
		{"a/_x.go", true},
		{"a/.x.go", true},
		{"a/x.go", false},
		{"x.go", false},
	}
	for _, testCase := range testCases {
		t.Run(testCase.file, func(t *testing.T) {
			t.Parallel()

			if got := IsOutsidePackage(testCase.file); got != testCase.want {
				t.Fatalf("IsOutsidePackage(%q) = %v, want %v", testCase.file, got, testCase.want)
			}
		})
	}
}
