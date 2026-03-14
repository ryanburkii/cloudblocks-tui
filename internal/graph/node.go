// internal/graph/node.go
package graph

// Node represents a single AWS resource in the architecture.
type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`   // e.g. "aws_vpc"
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}
