package singleuse

import "strings"

func eligible(decl declaration) bool {
	if decl.Name == "_" || decl.Exported {
		return false
	}
	switch decl.Kind {
	case declConst:
		return true
	case declVar:
		return !hoistedInit(decl.Init) && !mustStayPackageLevel(decl)
	default:
		return false
	}
}

func hoistedInit(kind initKind) bool {
	return kind == initCall || kind == initComposite
}

func mustStayPackageLevel(decl declaration) bool {
	// A //go:embed only compiles at package level, and a stateful type resets per call.
	return hasPinnedAnnotation(decl) || isStatefulType(decl)
}

func hasPinnedAnnotation(decl declaration) bool {
	for _, annotation := range decl.Annotations {
		if strings.HasPrefix(annotation, "//go:embed") {
			return true
		}
	}
	return false
}

func isStatefulType(decl declaration) bool {
	return strings.HasPrefix(decl.TypeName, "atomic.") ||
		strings.HasPrefix(decl.TypeName, "sync.")
}
