package gosrc

import (
	"path"
	"strings"
)

type inclusion struct {
	ext      string
	tags     string
	isTest   bool
	ignored  bool
	vendored bool
}

func SameInclusion(oldPath, newPath string) bool {
	return inclusionOf(oldPath) == inclusionOf(newPath)
}

func IsCompiled(file string) bool { return isToolchainExt(path.Ext(file)) }

func IsBuildIgnored(file string) bool {
	// A vendor dir stays compiled: a nested module's own vendor/ is its dependency tree.
	return isHidden(path.Base(file)) || isIgnoredDir(path.Dir(file))
}

func inclusionOf(file string) inclusion {
	base := path.Base(file)
	ext := path.Ext(base)
	if !isToolchainExt(ext) {
		return inclusion{}
	}
	name := strings.TrimSuffix(base, ext)
	return inclusion{
		ext:     ext,
		tags:    platformTags(base),
		isTest:  ext == ".go" && strings.HasSuffix(name, "_test"),
		ignored: IsBuildIgnored(file),
		// The root vendor/ switches the go command to -mod=vendor.
		vendored: strings.HasPrefix(file, "vendor/"),
	}
}

func isIgnoredDir(dir string) bool {
	if dir == "." {
		return false
	}
	for elem := range strings.SplitSeq(dir, "/") {
		if elem == "testdata" || isHidden(elem) {
			return true
		}
	}
	return false
}

func isHidden(name string) bool {
	return strings.HasPrefix(name, "_") || strings.HasPrefix(name, ".")
}

func platformTags(base string) string {
	name, _, _ := strings.Cut(base, ".")
	_, name, found := strings.Cut(name, "_")
	if !found {
		return ""
	}
	words := strings.Split(name, "_")
	if words[len(words)-1] == "test" {
		words = words[:len(words)-1]
	}
	if len(words) == 0 {
		return ""
	}
	count := len(words)
	if count >= 2 && isKnownOS(words[count-2]) && isKnownArch(words[count-1]) {
		return words[count-2] + "_" + words[count-1]
	}
	if last := words[len(words)-1]; isKnownOS(last) || isKnownArch(last) {
		return last
	}
	return ""
}

func isToolchainExt(ext string) bool {
	switch ext {
	case ".go", ".s", ".S", ".sx", ".c", ".h", ".cc", ".cpp", ".cxx", ".hh", ".hpp", ".hxx",
		".m", ".f", ".F", ".for", ".f90", ".swig", ".swigcxx", ".syso":
		return true
	}
	return false
}

func isKnownOS(word string) bool {
	switch word {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js",
		"linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos":
		return true
	}
	return false
}

func isKnownArch(word string) bool {
	switch word {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips",
		"mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64", "ppc64le",
		"riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	}
	return false
}
