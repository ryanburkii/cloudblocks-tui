// internal/terraform/generator_test.go
package terraform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cloudblocks-tui/internal/catalog"
	"cloudblocks-tui/internal/graph"
	tfgen "cloudblocks-tui/internal/terraform"
)

func setup(t *testing.T) (*graph.Architecture, string) {
	t.Helper()
	return graph.New(), t.TempDir()
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func TestGenerate_SingleVPC(t *testing.T) {
	arch, dir := setup(t)
	arch.AddNode(&graph.Node{
		ID:   "vpc-1",
		Type: "aws_vpc",
		Name: "my-vpc",
		Properties: map[string]interface{}{
			"cidr_block":           "10.0.0.0/16",
			"enable_dns_hostnames": true,
		},
	})

	if err := tfgen.Generate(arch, catalog.ByTFType, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	main := readFile(t, dir, "main.tf")
	if !strings.Contains(main, `resource "aws_vpc" "vpc-1"`) {
		t.Errorf("expected vpc resource block, got:\n%s", main)
	}
	// string prop must be quoted
	if !strings.Contains(main, `cidr_block = "10.0.0.0/16"`) {
		t.Errorf("expected quoted cidr_block, got:\n%s", main)
	}
	// bool prop must be unquoted
	if !strings.Contains(main, `enable_dns_hostnames = true`) {
		t.Errorf("expected unquoted bool, got:\n%s", main)
	}
}

func TestGenerate_CrossReference(t *testing.T) {
	arch, dir := setup(t)
	arch.AddNode(&graph.Node{
		ID:   "vpc-1",
		Type: "aws_vpc",
		Name: "vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16", "enable_dns_hostnames": true},
	})
	arch.AddNode(&graph.Node{
		ID:   "sub-1",
		Type: "aws_subnet",
		Name: "subnet",
		Properties: map[string]interface{}{"cidr_block": "10.0.1.0/24", "availability_zone": "us-east-1a"},
	})
	arch.Connect("vpc-1", "sub-1")

	if err := tfgen.Generate(arch, catalog.ByTFType, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	main := readFile(t, dir, "main.tf")
	if !strings.Contains(main, `vpc_id = aws_vpc.vpc-1.id`) {
		t.Errorf("expected vpc_id cross-reference, got:\n%s", main)
	}
}

func TestGenerate_Outputs(t *testing.T) {
	arch, dir := setup(t)
	arch.AddNode(&graph.Node{
		ID:         "vpc-1",
		Type:       "aws_vpc",
		Name:       "vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16", "enable_dns_hostnames": true},
	})
	arch.AddNode(&graph.Node{
		ID:         "ecs-1",
		Type:       "aws_ecs_service",
		Name:       "svc",
		Properties: map[string]interface{}{},
	})

	if err := tfgen.Generate(arch, catalog.ByTFType, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	outputs := readFile(t, dir, "outputs.tf")
	if !strings.Contains(outputs, `"vpc-1"`) {
		t.Errorf("expected vpc-1 output block, got:\n%s", outputs)
	}
	if strings.Contains(outputs, `"ecs-1"`) {
		t.Errorf("ecs-1 should not have output block (empty TFOutputAttr), got:\n%s", outputs)
	}
}

func TestGenerate_Variables(t *testing.T) {
	arch, dir := setup(t)
	arch.AddNode(&graph.Node{
		ID:         "vpc-1",
		Type:       "aws_vpc",
		Name:       "vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16", "enable_dns_hostnames": true},
	})

	if err := tfgen.Generate(arch, catalog.ByTFType, dir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	vars := readFile(t, dir, "variables.tf")
	if !strings.Contains(vars, `variable "aws_region"`) {
		t.Errorf("expected aws_region variable, got:\n%s", vars)
	}
}
