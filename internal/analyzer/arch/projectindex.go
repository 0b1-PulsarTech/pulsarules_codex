package arch

type projectIndex struct {
	pkgMap map[string]*pkgImports
	graph  depGraph
}

func loadProjectIndex(projectDir, modulePath string) *projectIndex {
	pkgMap := discoverPackages(projectDir, modulePath)
	// Each analyzer walks on its own: sharing one index would need a package-level global.
	graph := buildGraph(pkgMap, modulePath)
	return &projectIndex{pkgMap: pkgMap, graph: graph}
}
