package arch

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/workspace/source"
)

type depGraph map[string][]string

func loadDepGraph(provider source.Provider, modulePath string) depGraph {
	return buildGraph(discoverPackages(provider, modulePath), modulePath)
}

func buildGraph(pkgs map[string][]string, modulePath string) depGraph {
	graph := make(depGraph, len(pkgs))
	for pkg, imports := range pkgs {
		graph[pkg] = localImports(modulePath, imports)
	}
	return graph
}

func findCycles(graph depGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	stack := make([]string, 0, len(graph))

	var dfs func(node string)
	dfs = func(node string) {
		if inStack[node] {
			if start := slices.Index(stack, node); start >= 0 {
				cycles = append(cycles, rotateToSmallest(slices.Clone(stack[start:])))
			}
			return
		}
		if visited[node] {
			return
		}
		visited[node] = true
		inStack[node] = true
		stack = append(stack, node)

		for _, dep := range slices.Sorted(slices.Values(graph[node])) {
			dfs(dep)
		}

		stack = stack[:len(stack)-1]
		inStack[node] = false
	}

	for _, node := range slices.Sorted(maps.Keys(graph)) {
		if !visited[node] {
			dfs(node)
		}
	}
	return cycles
}

func rotateToSmallest(cycle []string) []string {
	smallest := slices.Index(cycle, slices.Min(cycle))
	return slices.Concat(cycle[smallest:], cycle[:smallest])
}

func classifyLayer(pkgPath string) string {
	for segment := range strings.SplitSeq(pkgPath, "/") {
		switch segment {
		case "domain":
			return "domain"
		case "infra":
			return "infra"
		case "transport", "rest", "grpc", "http":
			return "transport"
		case "cmd":
			return "cmd"
		}
	}
	return layerUnclassified
}

const layerUnclassified = ""

func checkBoundaries(graph depGraph, modulePath string) []string {
	const (
		layerRankCmd = iota
		layerRankTransport
		layerRankInfra
		layerRankDomain
	)

	var violations []string
	layerOrder := map[string]int{
		"cmd":       layerRankCmd,
		"transport": layerRankTransport,
		"infra":     layerRankInfra,
		"domain":    layerRankDomain,
	}

	for _, pkg := range slices.Sorted(maps.Keys(graph)) {
		deps := graph[pkg]
		pkgLayer := classifyLayer(pkg)
		for _, dep := range deps {
			depLayer := classifyLayer(dep)
			// The zero value would rank an unclassified dependency as cmd, the outermost
			// layer, and report every import of it as an inner layer depending on an outer one.
			if depLayer == layerUnclassified {
				continue
			}
			pkgOrd := layerOrder[pkgLayer]
			depOrd := layerOrder[depLayer]
			if pkgOrd > depOrd {
				violations = append(violations,
					fmt.Sprintf("%s (%s) imports %s (%s): inner layer depends on outer",
						stripModule(pkg, modulePath), pkgLayer,
						stripModule(dep, modulePath), depLayer))
			}
		}
	}
	return violations
}

func stripModule(pkg, modulePath string) string {
	return strings.TrimPrefix(pkg, modulePath+"/")
}
