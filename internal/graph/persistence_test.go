package graph_test

import (
	"path/filepath"
	"testing"
	"cloudblocks-tui/internal/graph"
)

func TestSaveLoad(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "my-vpc", Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16"}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "my-subnet", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")

	tmp := filepath.Join(t.TempDir(), "arch.json")
	if err := arch.Save(tmp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	arch2 := graph.New()
	if err := arch2.Load(tmp); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(arch2.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(arch2.Nodes))
	}
	if len(arch2.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(arch2.Edges))
	}
	if arch2.NodeOrder[0] != "vpc-1" || arch2.NodeOrder[1] != "sub-1" {
		t.Errorf("expected NodeOrder=[vpc-1, sub-1], got %v", arch2.NodeOrder)
	}
	if n, ok := arch2.Nodes["vpc-1"]; !ok || n.Name != "my-vpc" {
		t.Errorf("expected vpc-1 with name my-vpc")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	arch := graph.New()
	err := arch.Load("/nonexistent/path.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
