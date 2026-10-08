package projectfile

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/0b1-PulsarTech/pulsarules_codex/internal/printable"
)

type extent struct {
	count  int
	height int
}

type expander struct {
	extents map[*yaml.Node]extent
	active  map[*yaml.Node]bool
	visits  int
}

type overflow struct {
	line   int
	path   *Field
	reason string
}

func checkExpansion(path string, root *yaml.Node) error {
	walker := expander{extents: map[*yaml.Node]extent{}, active: map[*yaml.Node]bool{}}
	// Aliases and merge keys share nodes, so the expanded document can be exponentially
	// larger and deeper than the file. Measuring each node once bounds every later pass.
	_, over := walker.size(root, newRoot(), 0)
	if over == nil {
		return nil
	}
	message := over.reason
	if field := over.path.String(); field != "" {
		message = clipPath(field) + ": " + message
	}
	return fmt.Errorf("%s: line %d: %s", path, over.line, message)
}

func (walker *expander) size(node *yaml.Node, at *Field, above int) (extent, *overflow) {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		span, over := walker.size(node.Alias, at, above)
		if over != nil {
			over.line = node.Line
		}
		return span, over
	}
	if span, done := walker.extents[node]; done {
		if above+span.height > maxDepth {
			return span, depthOverflow(node, at)
		}
		return span, nil
	}
	if walker.active[node] {
		reason := "alias refers to its own anchor"
		return extent{}, &overflow{line: node.Line, path: at, reason: reason}
	}
	if node.Kind == yaml.ScalarNode && len(node.Value) > maxScalarBytes {
		return extent{}, longOverflow(node, at)
	}
	return walker.sizeChildren(node, at, above)
}

func (walker *expander) sizeChildren(
	node *yaml.Node, at *Field, above int,
) (extent, *overflow) {
	walker.visits++
	span, nested := extent{count: 1}, above
	if node.Kind == yaml.MappingNode || node.Kind == yaml.SequenceNode {
		span.height, nested = 1, above+1
		if nested > maxDepth {
			return extent{}, depthOverflow(node, at)
		}
	}
	if over := breadth(node, at); over != nil {
		return extent{}, over
	}
	walker.active[node] = true
	defer delete(walker.active, node)
	for idx, child := range node.Content {
		next := childPath(node, idx, at)
		sub, over := walker.size(child, next, nested)
		if over != nil {
			return extent{}, over
		}
		span.height = max(span.height, sub.height+nested-above)
		if span.count += sub.count; span.count > maxExpandedNodes {
			return extent{}, nodesOverflow(child, next)
		}
	}
	walker.extents[node] = span
	return span, nil
}

func breadth(node *yaml.Node, at *Field) *overflow {
	switch node.Kind {
	case yaml.MappingNode:
		if len(node.Content)/2 > MaxMapKeys {
			reason := fmt.Sprintf("more than %d keys", MaxMapKeys)
			return &overflow{line: node.Line, path: at, reason: reason}
		}
		return duplicateKey(node, at)
	case yaml.SequenceNode:
		if len(node.Content) > MaxSeqItems {
			reason := fmt.Sprintf("more than %d entries", MaxSeqItems)
			return &overflow{line: node.Line, path: at, reason: reason}
		}
	}
	return nil
}

func duplicateKey(node *yaml.Node, at *Field) *overflow {
	type rawKey struct {
		kind  yaml.Kind
		value string
	}
	// Mirrors yaml.v3, which compares raw kind and value, before the quadratic decoder runs.
	seen := make(map[rawKey]int, len(node.Content)/2)
	for idx := 0; idx+1 < len(node.Content); idx += 2 {
		key := node.Content[idx]
		raw := rawKey{kind: key.Kind, value: key.Value}
		if first, dup := seen[raw]; dup {
			reason := fmt.Sprintf("duplicate key, first at line %d", first)
			return &overflow{line: key.Line, path: childPath(node, idx, at), reason: reason}
		}
		seen[raw] = key.Line
	}
	return nil
}

func depthOverflow(node *yaml.Node, at *Field) *overflow {
	reason := fmt.Sprintf("nested deeper than %d levels", maxDepth)
	return &overflow{line: node.Line, path: at, reason: reason}
}

func longOverflow(node *yaml.Node, at *Field) *overflow {
	reason := fmt.Sprintf("value longer than %d bytes", maxScalarBytes)
	return &overflow{line: node.Line, path: at, reason: reason}
}

func nodesOverflow(node *yaml.Node, at *Field) *overflow {
	reason := fmt.Sprintf("document expands past %d nodes", maxExpandedNodes)
	return &overflow{line: node.Line, path: at, reason: reason}
}

func childPath(node *yaml.Node, idx int, path *Field) *Field {
	switch node.Kind {
	case yaml.SequenceNode:
		return path.item(idx)
	case yaml.MappingNode:
		if key := node.Content[idx&^1]; isMerge(key) {
			return path.key("<<")
		}
		return path.key(keyName(node.Content[idx&^1]))
	}
	return path
}

func clipPath(path string) string {
	// Keeps the last MaxClipBytes of escaped text, and the ellipsis comes on top of that.
	return printable.Clip(path, maxClipBytes, 0, printable.TailEdge)
}

func firstAliasLine(node *yaml.Node) int {
	if node.Kind == yaml.AliasNode {
		return node.Line
	}
	for _, child := range node.Content {
		if line := firstAliasLine(child); line > 0 {
			return line
		}
	}
	return 0
}
