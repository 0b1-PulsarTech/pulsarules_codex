package singleuse

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/pass"
)

type exemptions struct {
	pinnedAnnotations    []string
	statefulTypePrefixes []string
	hoistedInitKinds     map[InitKind]bool
}

func exemptionsFromParams(params pass.ParamSet) exemptions {
	const (
		paramPinnedAnnotations    = "pinned_annotations"
		paramStatefulTypePrefixes = "stateful_type_prefixes"
		paramHoistedInitKinds     = "hoisted_init_kinds"
	)
	defaultPinnedAnnotations := []string{"//go:embed"}
	defaultStatefulTypePrefixes := []string{"atomic.", "sync."}
	defaultHoistedInitKinds := []string{string(InitCall), string(InitComposite)}
	kinds := params.Strings(paramHoistedInitKinds, defaultHoistedInitKinds)
	hoisted := make(map[InitKind]bool, len(kinds))
	for _, kind := range kinds {
		hoisted[InitKind(kind)] = true
	}
	return exemptions{
		pinnedAnnotations: params.Strings(paramPinnedAnnotations, defaultPinnedAnnotations),
		statefulTypePrefixes: params.Strings(
			paramStatefulTypePrefixes,
			defaultStatefulTypePrefixes,
		),
		hoistedInitKinds: hoisted,
	}
}

func (exempt exemptions) eligible(decl Declaration, kinds declKinds) bool {
	if decl.Name == "_" || decl.Exported || !kinds.allows(decl.Kind) {
		return false
	}
	if decl.Kind != DeclVar {
		return true
	}
	return !exempt.hoistedInitKinds[decl.Init] && !exempt.mustStayPackageLevel(decl)
}

func (exempt exemptions) mustStayPackageLevel(decl Declaration) bool {
	// A //go:embed only compiles at package level, and a stateful type resets per call.
	return exempt.hasPinnedAnnotation(decl) || exempt.isStatefulType(decl)
}

func (exempt exemptions) hasPinnedAnnotation(decl Declaration) bool {
	for _, annotation := range decl.Annotations {
		for _, pinned := range exempt.pinnedAnnotations {
			if strings.HasPrefix(annotation, pinned) {
				return true
			}
		}
	}
	return false
}

func (exempt exemptions) isStatefulType(decl Declaration) bool {
	for _, prefix := range exempt.statefulTypePrefixes {
		if strings.HasPrefix(decl.TypeName, prefix) {
			return true
		}
	}
	return false
}
