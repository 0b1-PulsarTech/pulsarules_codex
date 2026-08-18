package arch

// projectIndex holds the parsed package map and the built dependency graph
// for one project.
type projectIndex struct {
	pkgMap map[string]*pkgImports
	graph  depGraph
}

// why: the boundary and cycle analyzers each walk the tree, and this used to be shared through a
// package-level sync.Map - a mutable global this codebase forbids. Measured cost of dropping it on
// ~60 packages: 30.8ms vs 16.1ms per governance run. Real, but not worth threading a cache through
// the pipeline's generic analyzer wiring. Upgrade path: give registerForScope an *IndexCache built
// once per Session.Analyze if this ever dominates latency on a large target.
func loadProjectIndex(projectDir, modulePath string) *projectIndex {
	pkgMap := discoverPackages(projectDir, modulePath)
	graph := buildGraph(pkgMap, modulePath)
	return &projectIndex{pkgMap: pkgMap, graph: graph}
}
