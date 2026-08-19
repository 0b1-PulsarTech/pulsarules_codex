package arch

import (
	"fmt"
	"strings"
)

type depGraph map[string][]string

func buildGraph(pkgs map[string]*pkgImports, modulePath string) depGraph {
	g := make(depGraph, len(pkgs))
	for pkg, info := range pkgs {
		g[pkg] = localImports(modulePath, info.Imports)
	}
	return g
}

func findCycles(g depGraph) [][]string {
	var cycles [][]string
	visited := make(map[string]bool)
	inStack := make(map[string]bool)
	stack := make([]string, 0, len(g))

	var dfs func(node string)
	dfs = func(node string) {
		if inStack[node] {
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

func checkBoundaries(g depGraph, modulePath string) []string {
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

	for pkg, deps := range g {
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
