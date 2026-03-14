package graph_test

import (
	"testing"
	"cloudblocks-tui/internal/graph"
)

func TestAddNode(t *testing.T) {
	arch := graph.New()
	n := &graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "my-vpc", Properties: map[string]interface{}{}}
	arch.AddNode(n)
	if _, ok := arch.Nodes["vpc-1"]; !ok {
		t.Error("expected vpc-1 in Nodes")
	}
	if len(arch.NodeOrder) != 1 || arch.NodeOrder[0] != "vpc-1" {
		t.Errorf("expected NodeOrder=[vpc-1], got %v", arch.NodeOrder)
	}
}

func TestRemoveNode(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	arch.RemoveNode("vpc-1")
	if _, ok := arch.Nodes["vpc-1"]; ok {
		t.Error("vpc-1 should be removed")
	}
	for _, e := range arch.Edges {
		if e.From == "vpc-1" || e.To == "vpc-1" {
			t.Error("edges referencing vpc-1 should be removed")
		}
	}
}

func TestConnect(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	if len(arch.Edges) != 1 || arch.Edges[0].From != "vpc-1" || arch.Edges[0].To != "sub-1" {
		t.Errorf("expected edge vpc-1→sub-1, got %v", arch.Edges)
	}
}

func TestChildren(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	children := arch.Children("vpc-1")
	if len(children) != 1 || children[0].ID != "sub-1" {
		t.Errorf("expected [sub-1], got %v", children)
	}
}

func TestRoots(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	roots := arch.Roots()
	if len(roots) != 1 || roots[0].ID != "vpc-1" {
		t.Errorf("expected [vpc-1], got %v", roots)
	}
}

func TestAddNode_Duplicate(t *testing.T) {
	arch := graph.New()
	n := &graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "my-vpc", Properties: map[string]interface{}{}}
	arch.AddNode(n)
	arch.AddNode(n) // second add must be a no-op
	if len(arch.Nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(arch.Nodes))
	}
	if len(arch.NodeOrder) != 1 {
		t.Errorf("expected NodeOrder len 1, got %d", len(arch.NodeOrder))
	}
}

func TestConnect_Duplicate(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	arch.Connect("vpc-1", "sub-1") // second connect must be a no-op
	if len(arch.Edges) != 1 {
		t.Errorf("expected 1 edge after duplicate connect, got %d", len(arch.Edges))
	}
}

func TestRemoveNode_NodeOrder(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "v", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "s", Properties: map[string]interface{}{}})
	arch.RemoveNode("vpc-1")
	for _, id := range arch.NodeOrder {
		if id == "vpc-1" {
			t.Error("NodeOrder should not contain vpc-1 after removal")
		}
	}
	if len(arch.NodeOrder) != 1 || arch.NodeOrder[0] != "sub-1" {
		t.Errorf("expected NodeOrder=[sub-1], got %v", arch.NodeOrder)
	}
}
