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

// classifyLayer names the layer a package sits in, or layerUnclassified when no
// rule recognizes it.
// simplification: prefix-based classification fits the standard app layout.
// Upgrade path: an explicit config map for a non-standard one.
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
	return layerUnclassified
}

// layerUnclassified marks a package no layer rule recognizes. It is SKIPPED
// rather than ranked: ranked innermost, an apps/ + libs/ + proto/ tree classified
// as unrecognized and every ordinary import read as an inward violation.
const layerUnclassified = ""

// layer rank constants used by checkBoundaries' inward-only ordering: a
// higher rank must never depend on a lower one.
const (
	layerRankCmd = iota
	layerRankTransport
	layerRankInfra
	layerRankDomain
)

// checkBoundaries verifies that inner layers do not depend on outer layers.
// Returns human-readable violation messages.
func checkBoundaries(g depGraph, modulePath string) []string {
	var violations []string
	layerOrder := map[string]int{
		"cmd":       layerRankCmd,
		"transport": layerRankTransport,
		"infra":     layerRankInfra,
		"domain":    layerRankDomain,
	}

	for pkg, deps := range g {
		pkgLayer := classifyLayer(pkg)
		if pkgLayer == layerUnclassified {
			continue
		}
		for _, dep := range deps {
			depLayer := classifyLayer(dep)
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
