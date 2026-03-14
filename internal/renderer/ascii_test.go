// internal/renderer/ascii_test.go
package renderer_test

import (
	"strings"
	"testing"

	"cloudblocks-tui/internal/graph"
	"cloudblocks-tui/internal/renderer"
)

func mkNode(id, name string) *graph.Node {
	return &graph.Node{ID: id, Type: "aws_vpc", Name: name, Properties: map[string]interface{}{}}
}

func TestRender_Empty(t *testing.T) {
	arch := graph.New()
	got := renderer.Render(arch)
	if got != "" {
		t.Errorf("expected empty string for empty arch, got %q", got)
	}
}

func TestRender_SingleNode(t *testing.T) {
	arch := graph.New()
	arch.AddNode(mkNode("vpc-1", "my-vpc"))
	got := renderer.Render(arch)
	if !strings.Contains(got, "my-vpc") {
		t.Errorf("expected my-vpc in output, got %q", got)
	}
	if !strings.Contains(got, "vpc-1") {
		t.Errorf("expected vpc-1 in output, got %q", got)
	}
}

func TestRender_Tree(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "my-vpc", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "my-subnet", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "ecs-1", Type: "aws_ecs_service", Name: "api", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	arch.Connect("sub-1", "ecs-1")

	got := renderer.Render(arch)
	lines := strings.Split(strings.TrimSpace(got), "\n")

	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d:\n%s", len(lines), got)
	}
	if !strings.Contains(lines[0], "my-vpc") {
		t.Errorf("line 0 should contain my-vpc, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "my-subnet") {
		t.Errorf("line 1 should contain my-subnet, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "api") {
		t.Errorf("line 2 should contain api, got %q", lines[2])
	}
	if !strings.Contains(lines[1], "└─") {
		t.Errorf("line 1 should contain └─, got %q", lines[1])
	}
}

func TestRender_MultipleRoots(t *testing.T) {
	arch := graph.New()
	arch.AddNode(mkNode("vpc-1", "my-vpc"))
	arch.AddNode(mkNode("s3-1", "my-bucket"))
	got := renderer.Render(arch)
	if !strings.Contains(got, "my-vpc") || !strings.Contains(got, "my-bucket") {
		t.Errorf("expected both roots in output, got:\n%s", got)
	}
}
