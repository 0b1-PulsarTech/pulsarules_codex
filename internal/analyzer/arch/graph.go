package arch

import (
	"fmt"
	"strings"
)

type depGraph map[string][]string

// buildGraph builds a directed dependency graph from the package map, keeping
// only local (within-module) dependencies.
func buildGraph(pkgs map[string]*pkgImports, modulePath string) depGraph {
	g := make(depGraph, len(pkgs))
	for pkg, info := range pkgs {
		g[pkg] = localImports(modulePath, info.Imports)
	}
	return g
}

// findCycles returns all cycles in the dependency graph, each expressed as a
// sequence of package paths where the last depends on the first.
func findCycles(g depGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	stack := make([]string, 0, len(g))

	var dfs func(node string)
	dfs = func(node string) {
		if inStack[node] {
			// extract the cycle from the stack
			start := -1
			for i, p := range stack {
				if p == node {
					start = i
					break
				}
			}
			if start >= 0 {
				cycle := make([]string, len(stack)-start)
				copy(cycle, stack[start:])
				cycles = append(cycles, cycle)
			}
			return
		}
		if visited[node] {
			return
		}
		visited[node] = true
		inStack[node] = true
		stack = append(stack, node)

		for _, dep := range g[node] {
			dfs(dep)
		}

		stack = stack[:len(stack)-1]
		inStack[node] = false
	}

	for node := range g {
		if !visited[node] {
			dfs(node)
		}
	}
	return cycles
}

// simplification: boundary rules use a simple prefix-based classification
// (domain, infra, transport). This works for the standard app layout but will
// need an explicit config map for non-standard layouts.
func classifyLayer(pkgPath string) string {
	for p := range strings.SplitSeq(pkgPath, "/") {
		switch p {
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
	return "other"
}

// checkBoundaries verifies that inner layers do not depend on outer layers.
// Returns human-readable violation messages.
func checkBoundaries(g depGraph, modulePath string) []string {
	var violations []string
	layerOrder := map[string]int{
		"cmd":       0,
		"transport": 1,
		"infra":     2,
		"domain":    3,
		"other":     4,
	}

	for pkg, deps := range g {
		pkgLayer := classifyLayer(pkg)
		for _, dep := range deps {
			depLayer := classifyLayer(dep)
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
