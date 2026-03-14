// internal/graph/architecture.go
package graph

import (
	"encoding/json"
	"os"
)

// Architecture holds the complete set of nodes and edges.
type Architecture struct {
	Nodes     map[string]*Node
	Edges     []Edge
	NodeOrder []string // insertion order for rendering
}

// New returns an empty Architecture.
func New() *Architecture {
	return &Architecture{
		Nodes: make(map[string]*Node),
	}
}

// AddNode adds n to the architecture. No-op if a node with the same ID already exists.
func (a *Architecture) AddNode(n *Node) {
	if _, exists := a.Nodes[n.ID]; exists {
		return
	}
	a.Nodes[n.ID] = n
	a.NodeOrder = append(a.NodeOrder, n.ID)
}

// RemoveNode removes the node with the given ID and all edges connected to it.
func (a *Architecture) RemoveNode(id string) {
	delete(a.Nodes, id)
	// Remove from NodeOrder.
	filtered := a.NodeOrder[:0:0]
	for _, oid := range a.NodeOrder {
		if oid != id {
			filtered = append(filtered, oid)
		}
	}
	a.NodeOrder = filtered
	// Remove edges referencing this node.
	edges := a.Edges[:0:0]
	for _, e := range a.Edges {
		if e.From != id && e.To != id {
			edges = append(edges, e)
		}
	}
	a.Edges = edges
}

// Connect adds a directed edge from → to. Duplicate edges are silently ignored.
func (a *Architecture) Connect(from, to string) {
	for _, e := range a.Edges {
		if e.From == from && e.To == to {
			return
		}
	}
	a.Edges = append(a.Edges, Edge{From: from, To: to})
}

// Children returns the direct children of the node with the given ID,
// in insertion order.
func (a *Architecture) Children(id string) []*Node {
	var result []*Node
	for _, e := range a.Edges {
		if e.From == id {
			if n, ok := a.Nodes[e.To]; ok {
				result = append(result, n)
			}
		}
	}
	return result
}

// Roots returns all nodes that have no incoming edges, in NodeOrder.
func (a *Architecture) Roots() []*Node {
	hasParent := make(map[string]bool)
	for _, e := range a.Edges {
		hasParent[e.To] = true
	}
	var result []*Node
	for _, id := range a.NodeOrder {
		if !hasParent[id] {
			if n, ok := a.Nodes[id]; ok {
				result = append(result, n)
			}
		}
	}
	return result
}

// architectureJSON is the serialization format for Architecture.
type architectureJSON struct {
	Nodes []*Node `json:"nodes"`
	Edges []Edge  `json:"edges"`
}

// Save serializes the Architecture to a JSON file at path.
func (a *Architecture) Save(path string) error {
	ordered := make([]*Node, 0, len(a.NodeOrder))
	for _, id := range a.NodeOrder {
		if n, ok := a.Nodes[id]; ok {
			ordered = append(ordered, n)
		}
	}
	data, err := json.MarshalIndent(architectureJSON{Nodes: ordered, Edges: a.Edges}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Load replaces the Architecture's contents from a JSON file at path.
func (a *Architecture) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var aj architectureJSON
	if err := json.Unmarshal(data, &aj); err != nil {
		return err
	}
	a.Nodes = make(map[string]*Node, len(aj.Nodes))
	a.NodeOrder = make([]string, 0, len(aj.Nodes))
	for _, n := range aj.Nodes {
		a.Nodes[n.ID] = n
		a.NodeOrder = append(a.NodeOrder, n.ID)
	}
	a.Edges = aj.Edges
	return nil
}
