package arch

import "github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"

func loadDepGraph(provider source.Provider, modulePath string) depGraph {
	// Each analyzer walks on its own: sharing one graph would need a package-level global.
	return buildGraph(discoverPackages(provider, modulePath), modulePath)
}
