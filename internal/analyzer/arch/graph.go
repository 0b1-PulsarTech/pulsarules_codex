package arch

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

type depGraph map[string][]string

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

type layer struct {
	name     string
	rank     int
	segments []string
}

func layerTable() []layer {
	// Rank 0 is the outermost layer: a package may import only the same rank or a higher one.
	const (
		rankCmd = iota
		rankTransport
		rankInfra
		rankDomain
	)
	return []layer{
		{name: "cmd", rank: rankCmd, segments: []string{"cmd"}},
		{
			name:     "transport",
			rank:     rankTransport,
			segments: []string{"transport", "rest", "grpc", "http"},
		},
		{name: "infra", rank: rankInfra, segments: []string{"infra"}},
		{name: "domain", rank: rankDomain, segments: []string{"domain"}},
	}
}

func classifyLayer(pkgPath string) (layer, bool) {
	table := layerTable()
	for segment := range strings.SplitSeq(pkgPath, "/") {
		for _, candidate := range table {
			if slices.Contains(candidate.segments, segment) {
				return candidate, true
			}
		}
	}
	return layer{}, false
}

type violation struct {
	message string
	site    importSite
}

func checkBoundaries(graph importGraph, modulePath string) []violation {
	var violations []violation
	for _, pkg := range slices.Sorted(maps.Keys(graph.deps)) {
		// An unclassified importer is the zero layer, rank 0, and never violates.
		pkgLayer, _ := classifyLayer(pkg)
		for _, dep := range graph.deps[pkg] {
			depLayer, depClassified := classifyLayer(dep)
			if !depClassified {
				continue
			}
			if pkgLayer.rank > depLayer.rank {
				violations = append(violations, violation{
					message: fmt.Sprintf("%s (%s) imports %s (%s): inner layer depends on outer",
						stripModule(pkg, modulePath), pkgLayer.name,
						stripModule(dep, modulePath), depLayer.name),
					site: graph.sites[edge{from: pkg, to: dep}],
				})
			}
		}
	}
	return violations
}

func stripModule(pkg, modulePath string) string {
	return strings.TrimPrefix(pkg, modulePath+"/")
}
