// internal/renderer/ascii.go
package renderer

import (
	"fmt"
	"strings"

	"cloudblocks-tui/internal/graph"
)

// Render returns the Architecture as a multi-line ASCII tree string.
// Root nodes (no incoming edges) are listed first; their children are indented.
func Render(arch *graph.Architecture) string {
	if len(arch.Nodes) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, root := range arch.Roots() {
		renderNode(&sb, arch, root, "", true)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func renderNode(sb *strings.Builder, arch *graph.Architecture, n *graph.Node, prefix string, isRoot bool) {
	if isRoot {
		fmt.Fprintf(sb, "%s (%s)\n", n.Name, n.ID)
	} else {
		fmt.Fprintf(sb, "%s└─ %s (%s)\n", prefix, n.Name, n.ID)
	}
	children := arch.Children(n.ID)
	childPrefix := prefix
	if !isRoot {
		childPrefix = prefix + "   "
	}
	for _, child := range children {
		renderNode(sb, arch, child, childPrefix, false)
	}
}
