package gomod

import "strings"

func IsHiddenName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func IsPackageDir(dir string) bool {
	if dir == "." {
		return true
	}
	for segment := range strings.SplitSeq(dir, "/") {
		// The ./... dir rule, except that vendor stays (a nested module's vendor/ is compiled);
		// the CLI skip sets drop it anyway.
		if IsHiddenName(segment) || segment == "testdata" {
			return false
		}
	}
	return true
}
