package dartgen

import (
	"path"
	"strings"
)

func suffixes() []string {
	return []string{
		".g.dart", ".freezed.dart", ".mocks.dart", ".gr.dart", ".config.dart",
		".pb.dart", ".pbenum.dart", ".pbgrpc.dart", ".pbjson.dart", ".template.dart",
	}
}

func IsGenerated(uri string) bool {
	for _, suffix := range suffixes() {
		if strings.HasSuffix(uri, suffix) {
			return true
		}
	}
	return false
}

func Resolve(file, uri string) string {
	// An empty result means the URI is not a plain path inside the project.
	if uri == "" || strings.Contains(uri, ":") || strings.HasPrefix(uri, "/") {
		return ""
	}
	target := path.Join(path.Dir(file), uri)
	if target == ".." || strings.HasPrefix(target, "../") {
		return ""
	}
	return target
}
