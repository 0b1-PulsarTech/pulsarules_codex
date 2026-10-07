package gomod

import "strings"

func IsHiddenName(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_")
}

func IsPackageDir(dir string) bool {
	if dir == "." {
		return false
	}
	for segment := range strings.SplitSeq(dir, "/") {
		if IsHiddenName(segment) || segment == "testdata" {
			return true
		}
	}
	return false
}
