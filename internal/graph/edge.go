// internal/graph/edge.go
package graph

// Edge represents a directional relationship between two nodes.
// From is the parent (e.g. VPC), To is the child (e.g. Subnet).
type Edge struct {
	From string `json:"from"` // Node ID
	To   string `json:"to"`   // Node ID
}
