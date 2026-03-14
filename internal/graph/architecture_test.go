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
