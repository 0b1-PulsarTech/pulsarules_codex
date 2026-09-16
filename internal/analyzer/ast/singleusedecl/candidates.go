package singleusedecl

import (
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/analyzer/core"
)

type exemptions struct {
	pinnedAnnotations    []string
	statefulTypePrefixes []string
	hoistedInitKinds     map[core.InitKind]bool
}

func exemptionsFromParams(params core.ParamSet) exemptions {
	const (
		paramPinnedAnnotations    = "pinned_annotations"
		paramStatefulTypePrefixes = "stateful_type_prefixes"
		paramHoistedInitKinds     = "hoisted_init_kinds"
	)
	defaultPinnedAnnotations := []string{"//go:embed"}
	defaultStatefulTypePrefixes := []string{"atomic.", "sync."}
	defaultHoistedInitKinds := []string{string(core.InitCall), string(core.InitComposite)}
	kinds := params.Strings(paramHoistedInitKinds, defaultHoistedInitKinds)
	hoisted := make(map[core.InitKind]bool, len(kinds))
	for _, kind := range kinds {
		hoisted[core.InitKind(kind)] = true
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

func (exempt exemptions) eligible(decl core.Declaration, kinds declKinds) bool {
	if decl.Name == "_" || decl.Exported || !kinds.allows(decl.Kind) {
		return false
	}
	if decl.Kind != core.DeclVar {
		return true
	}
	return !exempt.hoistedInitKinds[decl.Init] && !exempt.mustStayPackageLevel(decl)
}

func (exempt exemptions) mustStayPackageLevel(decl core.Declaration) bool {
	// A //go:embed only compiles at package level, and a stateful type resets per call.
	return exempt.hasPinnedAnnotation(decl) || exempt.isStatefulType(decl)
}

func (exempt exemptions) hasPinnedAnnotation(decl core.Declaration) bool {
	for _, annotation := range decl.Annotations {
		for _, pinned := range exempt.pinnedAnnotations {
			if strings.HasPrefix(annotation, pinned) {
				return true
			}
		}
	}
	return false
}

func (exempt exemptions) isStatefulType(decl core.Declaration) bool {
	for _, prefix := range exempt.statefulTypePrefixes {
		if strings.HasPrefix(decl.TypeName, prefix) {
			return true
		}
	}
	return false
}
