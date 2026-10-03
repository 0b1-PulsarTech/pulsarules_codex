package arch

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"

type projectIndex struct {
	pkgMap map[string]*pkgImports
	graph  depGraph
}

func loadProjectIndex(provider source.Provider, modulePath string) *projectIndex {
	pkgMap := discoverPackages(provider, modulePath)
	// Each analyzer walks on its own: sharing one index would need a package-level global.
	graph := buildGraph(pkgMap, modulePath)
	return &projectIndex{pkgMap: pkgMap, graph: graph}
}
