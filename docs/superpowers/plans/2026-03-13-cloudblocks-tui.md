# CloudBlocks TUI Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a keyboard-driven terminal UI for assembling and deploying AWS infrastructure as Terraform, using Go and Bubble Tea.

**Architecture:** Single Bubble Tea root model with sub-models per panel (catalog, architecture, properties, deploy). The root model owns the Architecture graph and routes `tea.Msg` to the focused sub-model. All business logic (graph operations, ASCII rendering, Terraform generation) is fully decoupled from the UI.

**Tech Stack:** Go 1.21+, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/lipgloss`, `github.com/charmbracelet/bubbles`

**Spec:** `docs/superpowers/specs/2026-03-13-cloudblocks-tui-design.md`

---

## File Structure

### Entry point
- `cmd/cloudblocks/main.go` — program entry; wires `tea.Program` with root model, checks for `cloudblocks.json`

### Graph (`internal/graph/`, package `graph`)
- `node.go` — `Node` struct (ID, Type, Name, Properties)
- `edge.go` — `Edge` struct (From, To)
- `architecture.go` — `Architecture` + methods: AddNode, RemoveNode, Connect, Children, Roots, Save, Load

### Resource definitions (`internal/aws/resources/`, package `resources`)
- `types.go` — `ResourceDef`, `PropDef`, `PropType` types
- `vpc.go`, `subnet.go`, `igw.go`, `natgw.go`, `sg.go` — networking (5 files)
- `ec2.go`, `ecs.go`, `lambda.go` — compute (3 files)
- `rds.go`, `dynamodb.go` — databases (2 files)
- `s3.go` — storage
- `alb.go` — load balancing

### Catalog (`internal/catalog/`, package `catalog`)
- `catalog.go` — `All()`, `ByCategory()`, `ByTFType()` — imports `internal/aws/resources`

### Renderer (`internal/renderer/`, package `renderer`)
- `ascii.go` — `Render(arch) string` — recursive ASCII tree

### Terraform generator (`internal/terraform/`, package `terraform`)
- `generator.go` — `Generate(arch, lookup, outDir)` — writes main.tf, variables.tf, outputs.tf

### Deploy runner (`internal/deploy/`, package `deploy`)
- `runner.go` — `Run(workDir, linesCh, doneCh)` — streams terraform output via channels

### TUI (`internal/tui/`)
- `messages.go` (package `tui`) — all inter-panel message types
- `keymap.go` (package `tui`) — `KeyMap` struct and defaults
- `layout.go` (package `tui`) — lipgloss styles, `panelSizes()`
- `app.go` (package `tui`) — root `Model`, Init/Update/View, panel routing
- `views/catalog.go` (package `views`) — `CatalogModel`
- `views/architecture.go` (package `views`) — `ArchModel`
- `views/properties.go` (package `views`) — `PropsModel`
- `views/deploy.go` (package `views`) — `DeployModel`

### Tests
- `internal/graph/architecture_test.go`
- `internal/renderer/ascii_test.go`
- `internal/terraform/generator_test.go`

---

## Chunk 1: Project Scaffolding + Graph Data Model

### Task 1: Initialize Go module

**Files:**
- Create: `go.mod`
- Create: `cmd/cloudblocks/main.go`

- [ ] **Step 1: Initialize module**

Run from the project root directory (`/home/burkii/cloudblocks-tui`):

```bash
go mod init cloudblocks-tui
```

Expected: `go.mod` created with `module cloudblocks-tui` and `go 1.21` (or current version).

- [ ] **Step 2: Create placeholder main.go**

```go
// cmd/cloudblocks/main.go
package main

import "fmt"

func main() {
	fmt.Println("CloudBlocks TUI")
}
```

- [ ] **Step 3: Add dependencies**

```bash
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
go get github.com/charmbracelet/bubbles@latest
go mod tidy
```

Expected: `go.sum` created, dependencies appear in `go.mod`.

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: exits 0, no errors.

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum cmd/
git commit -m "feat: initialize Go module with Bubble Tea dependencies"
```

---

### Task 2: Graph Node and Edge types

**Files:**
- Create: `internal/graph/node.go`
- Create: `internal/graph/edge.go`
- Create: `internal/graph/architecture_test.go` (first test only)

- [ ] **Step 1: Write the failing test**

```go
// internal/graph/architecture_test.go
package graph_test

import (
	"testing"

	"cloudblocks-tui/internal/graph"
)

func TestNodeFields(t *testing.T) {
	n := &graph.Node{
		ID:         "vpc-1",
		Type:       "aws_vpc",
		Name:       "my-vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16"},
	}
	if n.ID != "vpc-1" {
		t.Errorf("expected ID vpc-1, got %s", n.ID)
	}
	if n.Properties["cidr_block"] != "10.0.0.0/16" {
		t.Errorf("expected cidr_block 10.0.0.0/16, got %v", n.Properties["cidr_block"])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/graph/...
```

Expected: FAIL — `graph.Node` undefined.

- [ ] **Step 3: Implement Node and Edge**

```go
// internal/graph/node.go
package graph

// Node represents an AWS resource in the architecture graph.
type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
}
```

```go
// internal/graph/edge.go
package graph

// Edge represents a directed connection between two Nodes.
type Edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/graph/...
```

Expected: PASS — 1 test.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/node.go internal/graph/edge.go internal/graph/architecture_test.go
git commit -m "feat: add graph Node and Edge types"
```

---

### Task 3: Architecture core operations

**Files:**
- Create: `internal/graph/architecture.go`
- Modify: `internal/graph/architecture_test.go` (add 5 tests)

- [ ] **Step 1: Add failing tests**

Append to `internal/graph/architecture_test.go`:

```go
func TestAddNode(t *testing.T) {
	arch := graph.New()
	n := &graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "my-vpc", Properties: map[string]interface{}{}}
	arch.AddNode(n)
	if _, ok := arch.Nodes["vpc-1"]; !ok {
		t.Fatal("node not in Nodes map after AddNode")
	}
	if len(arch.NodeOrder) != 1 || arch.NodeOrder[0] != "vpc-1" {
		t.Fatalf("NodeOrder incorrect: %v", arch.NodeOrder)
	}
}

func TestRoots_SingleNode(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "vpc", Properties: map[string]interface{}{}})
	roots := arch.Roots()
	if len(roots) != 1 || roots[0].ID != "vpc-1" {
		t.Fatalf("expected [vpc-1], got %v", roots)
	}
}

func TestRoots_WithEdge(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "vpc", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "subnet", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	roots := arch.Roots()
	if len(roots) != 1 || roots[0].ID != "vpc-1" {
		t.Fatalf("expected only vpc-1 as root, got %v", roots)
	}
}

func TestChildren(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "vpc", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "subnet", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "ec2-1", Type: "aws_instance", Name: "ec2", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	arch.Connect("vpc-1", "ec2-1")
	children := arch.Children("vpc-1")
	if len(children) != 2 {
		t.Fatalf("expected 2 children of vpc-1, got %d", len(children))
	}
}

func TestRemoveNode_CleansEdgesAndOrder(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{ID: "vpc-1", Type: "aws_vpc", Name: "vpc", Properties: map[string]interface{}{}})
	arch.AddNode(&graph.Node{ID: "sub-1", Type: "aws_subnet", Name: "subnet", Properties: map[string]interface{}{}})
	arch.Connect("vpc-1", "sub-1")
	arch.RemoveNode("vpc-1")
	if _, ok := arch.Nodes["vpc-1"]; ok {
		t.Fatal("vpc-1 still in Nodes after RemoveNode")
	}
	if len(arch.Edges) != 0 {
		t.Fatalf("edges not cleaned up: %v", arch.Edges)
	}
	if len(arch.NodeOrder) != 1 || arch.NodeOrder[0] != "sub-1" {
		t.Fatalf("NodeOrder not updated: %v", arch.NodeOrder)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/graph/...
```

Expected: FAIL — `graph.New`, `arch.AddNode`, etc. undefined.

- [ ] **Step 3: Implement Architecture**

```go
// internal/graph/architecture.go
package graph

// Architecture is an in-memory directed graph of Nodes and Edges.
type Architecture struct {
	Nodes     map[string]*Node
	Edges     []Edge
	NodeOrder []string // insertion order; governs display order of root nodes
}

// New returns an empty Architecture.
func New() *Architecture {
	return &Architecture{
		Nodes: make(map[string]*Node),
	}
}

// AddNode inserts n into the architecture.
func (a *Architecture) AddNode(n *Node) {
	a.Nodes[n.ID] = n
	a.NodeOrder = append(a.NodeOrder, n.ID)
}

// RemoveNode removes the node with id and all edges that touch it.
func (a *Architecture) RemoveNode(id string) {
	delete(a.Nodes, id)
	for i, oid := range a.NodeOrder {
		if oid == id {
			a.NodeOrder = append(a.NodeOrder[:i], a.NodeOrder[i+1:]...)
			break
		}
	}
	kept := a.Edges[:0]
	for _, e := range a.Edges {
		if e.From != id && e.To != id {
			kept = append(kept, e)
		}
	}
	a.Edges = kept
}

// Connect adds a directed edge from → to.
func (a *Architecture) Connect(from, to string) {
	a.Edges = append(a.Edges, Edge{From: from, To: to})
}

// Children returns all nodes directly reachable from id.
func (a *Architecture) Children(id string) []*Node {
	var children []*Node
	for _, e := range a.Edges {
		if e.From == id {
			if n, ok := a.Nodes[e.To]; ok {
				children = append(children, n)
			}
		}
	}
	return children
}

// Roots returns all nodes with no incoming edges, ordered by NodeOrder.
func (a *Architecture) Roots() []*Node {
	hasParent := make(map[string]bool)
	for _, e := range a.Edges {
		hasParent[e.To] = true
	}
	var roots []*Node
	for _, id := range a.NodeOrder {
		if !hasParent[id] {
			if n, ok := a.Nodes[id]; ok {
				roots = append(roots, n)
			}
		}
	}
	return roots
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/graph/...
```

Expected: PASS — 6 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/architecture.go internal/graph/architecture_test.go
git commit -m "feat: add Architecture graph with AddNode, RemoveNode, Connect, Children, Roots"
```

---

### Task 4: Architecture persistence (Save / Load)

**Files:**
- Modify: `internal/graph/architecture.go` (add imports + Save + Load + architectureJSON)
- Modify: `internal/graph/architecture_test.go` (add round-trip test)

- [ ] **Step 1: Add failing Save/Load test**

First, update the import block at the top of `internal/graph/architecture_test.go` to add `"path/filepath"`:

```go
import (
	"path/filepath"
	"testing"

	"cloudblocks-tui/internal/graph"
)
```

Then append to `internal/graph/architecture_test.go`:

```go
func TestSaveLoad_RoundTrip(t *testing.T) {
	arch := graph.New()
	arch.AddNode(&graph.Node{
		ID:         "vpc-1",
		Type:       "aws_vpc",
		Name:       "my-vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16"},
	})
	arch.AddNode(&graph.Node{
		ID:         "sub-1",
		Type:       "aws_subnet",
		Name:       "my-subnet",
		Properties: map[string]interface{}{},
	})
	arch.Connect("vpc-1", "sub-1")

	dir := t.TempDir()
	path := filepath.Join(dir, "arch.json")

	if err := arch.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded := graph.New()
	if err := loaded.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(loaded.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(loaded.Nodes))
	}
	if len(loaded.Edges) != 1 || loaded.Edges[0].From != "vpc-1" || loaded.Edges[0].To != "sub-1" {
		t.Fatalf("edges mismatch: %v", loaded.Edges)
	}
	if len(loaded.NodeOrder) != 2 || loaded.NodeOrder[0] != "vpc-1" || loaded.NodeOrder[1] != "sub-1" {
		t.Fatalf("NodeOrder mismatch: %v", loaded.NodeOrder)
	}
	if loaded.Nodes["vpc-1"].Properties["cidr_block"] != "10.0.0.0/16" {
		t.Fatalf("Properties not preserved: %v", loaded.Nodes["vpc-1"].Properties)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/graph/...
```

Expected: FAIL — `arch.Save` undefined.

- [ ] **Step 3: Add Save and Load to architecture.go**

Add the following imports at the top of `internal/graph/architecture.go` (add an `import` block):

```go
import (
	"encoding/json"
	"fmt"
	"os"
)
```

Append to `internal/graph/architecture.go`:

```go
// architectureJSON is the wire format for serializing Architecture.
// NodeOrder is reconstructed from slice position on load.
type architectureJSON struct {
	Nodes []*Node `json:"nodes"`
	Edges []Edge  `json:"edges"`
}

// Save writes the architecture to path as JSON, ordered by NodeOrder.
func (a *Architecture) Save(path string) error {
	aj := &architectureJSON{Edges: a.Edges}
	for _, id := range a.NodeOrder {
		if n, ok := a.Nodes[id]; ok {
			aj.Nodes = append(aj.Nodes, n)
		}
	}
	data, err := json.MarshalIndent(aj, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal architecture: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write architecture: %w", err)
	}
	return nil
}

// Load replaces the architecture contents from a JSON file.
// NodeOrder is reconstructed from node slice position.
// Clears all existing state before loading.
func (a *Architecture) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read architecture: %w", err)
	}
	var aj architectureJSON
	if err := json.Unmarshal(data, &aj); err != nil {
		return fmt.Errorf("unmarshal architecture: %w", err)
	}
	a.Nodes = make(map[string]*Node)
	a.NodeOrder = nil
	for _, n := range aj.Nodes {
		a.Nodes[n.ID] = n
		a.NodeOrder = append(a.NodeOrder, n.ID)
	}
	a.Edges = aj.Edges
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/graph/...
```

Expected: PASS — 7 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/graph/architecture.go internal/graph/architecture_test.go
git commit -m "feat: add Architecture Save/Load with JSON persistence"
```

---

## Chunk 2: Resource Definitions + Catalog

### Task 5: ResourceDef type definitions

**Files:**
- Create: `internal/aws/resources/types.go`

- [ ] **Step 1: Create types.go**

```go
// internal/aws/resources/types.go
package resources

// PropType controls how a property value is formatted in Terraform HCL.
type PropType string

const (
	PropTypeString PropType = "string" // HCL: quoted  "value"
	PropTypeInt    PropType = "int"    // HCL: unquoted 512
	PropTypeBool   PropType = "bool"   // HCL: unquoted true
)

// PropDef describes one configurable property on an AWS resource.
type PropDef struct {
	Key      string
	Label    string
	Type     PropType
	Required bool
}

// ResourceDef describes an AWS resource type: its display info, default
// properties, property schema, and hints for Terraform HCL generation.
type ResourceDef struct {
	TFType        string                 // Terraform resource type, e.g. "aws_vpc"
	DisplayName   string                 // Human-readable name, e.g. "VPC"
	Category      string                 // Catalog category, e.g. "Networking"
	DefaultProps  map[string]interface{} // Default property values for new nodes
	PropSchema    []PropDef              // Ordered list of editable properties
	ParentRefAttr string                 // HCL attr on this resource that holds its parent ref (e.g. "vpc_id")
	TFRefAttr     string                 // HCL attr used as the RHS when others reference this resource (almost always "id")
	TFOutputAttr  string                 // HCL attr to expose in outputs.tf; "" = no output block
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/aws/resources/types.go
git commit -m "feat: add ResourceDef, PropDef, PropType types"
```

---

### Task 6: Networking resource definitions

**Files:**
- Create: `internal/aws/resources/vpc.go`
- Create: `internal/aws/resources/subnet.go`
- Create: `internal/aws/resources/igw.go`
- Create: `internal/aws/resources/natgw.go`
- Create: `internal/aws/resources/sg.go`

- [ ] **Step 1: Create networking resources**

```go
// internal/aws/resources/vpc.go
package resources

var VPC = &ResourceDef{
	TFType:       "aws_vpc",
	DisplayName:  "VPC",
	Category:     "Networking",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"cidr_block":           "10.0.0.0/16",
		"enable_dns_hostnames": true,
	},
	PropSchema: []PropDef{
		{Key: "cidr_block", Label: "CIDR Block", Type: PropTypeString, Required: true},
		{Key: "enable_dns_hostnames", Label: "DNS Hostnames", Type: PropTypeBool},
	},
}
```

```go
// internal/aws/resources/subnet.go
package resources

var Subnet = &ResourceDef{
	TFType:        "aws_subnet",
	DisplayName:   "Subnet",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps: map[string]interface{}{
		"cidr_block":        "10.0.1.0/24",
		"availability_zone": "us-east-1a",
	},
	PropSchema: []PropDef{
		{Key: "cidr_block", Label: "CIDR Block", Type: PropTypeString, Required: true},
		{Key: "availability_zone", Label: "AZ", Type: PropTypeString},
	},
}
```

```go
// internal/aws/resources/igw.go
package resources

var IGW = &ResourceDef{
	TFType:        "aws_internet_gateway",
	DisplayName:   "Internet Gateway",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps:  map[string]interface{}{},
	PropSchema:    []PropDef{},
}
```

```go
// internal/aws/resources/natgw.go
package resources

var NatGW = &ResourceDef{
	TFType:        "aws_nat_gateway",
	DisplayName:   "NAT Gateway",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "subnet_id",
	DefaultProps: map[string]interface{}{
		"allocation_id": "",
	},
	PropSchema: []PropDef{
		{Key: "allocation_id", Label: "EIP Allocation ID", Type: PropTypeString, Required: true},
	},
}
```

```go
// internal/aws/resources/sg.go
package resources

var SecurityGroup = &ResourceDef{
	TFType:        "aws_security_group",
	DisplayName:   "Security Group",
	Category:      "Networking",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "vpc_id",
	DefaultProps: map[string]interface{}{
		"name":        "my-sg",
		"description": "Managed by CloudBlocks",
	},
	PropSchema: []PropDef{
		{Key: "name", Label: "Name", Type: PropTypeString, Required: true},
		{Key: "description", Label: "Description", Type: PropTypeString},
	},
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/aws/resources/vpc.go internal/aws/resources/subnet.go internal/aws/resources/igw.go internal/aws/resources/natgw.go internal/aws/resources/sg.go
git commit -m "feat: add networking resource definitions"
```

---

### Task 7: Compute resource definitions

**Files:**
- Create: `internal/aws/resources/ec2.go`
- Create: `internal/aws/resources/ecs.go`
- Create: `internal/aws/resources/lambda.go`

- [ ] **Step 1: Create compute resources**

```go
// internal/aws/resources/ec2.go
package resources

var EC2 = &ResourceDef{
	TFType:        "aws_instance",
	DisplayName:   "EC2 Instance",
	Category:      "Compute",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "subnet_id",
	DefaultProps: map[string]interface{}{
		"instance_type": "t3.micro",
		"ami":           "ami-0c55b159cbfafe1f0",
	},
	PropSchema: []PropDef{
		{Key: "instance_type", Label: "Instance Type", Type: PropTypeString, Required: true},
		{Key: "ami", Label: "AMI", Type: PropTypeString, Required: true},
	},
}
```

```go
// internal/aws/resources/ecs.go
package resources

var ECS = &ResourceDef{
	TFType:       "aws_ecs_service",
	DisplayName:  "ECS Service",
	Category:     "Compute",
	TFRefAttr:    "id",
	TFOutputAttr: "",
	DefaultProps: map[string]interface{}{
		"cpu":           512,
		"memory":        1024,
		"desired_count": 1,
	},
	PropSchema: []PropDef{
		{Key: "cpu", Label: "CPU (units)", Type: PropTypeInt, Required: true},
		{Key: "memory", Label: "Memory (MB)", Type: PropTypeInt, Required: true},
		{Key: "desired_count", Label: "Desired Count", Type: PropTypeInt},
	},
}
```

```go
// internal/aws/resources/lambda.go
package resources

var Lambda = &ResourceDef{
	TFType:       "aws_lambda_function",
	DisplayName:  "Lambda Function",
	Category:     "Compute",
	TFRefAttr:    "arn",
	TFOutputAttr: "arn",
	DefaultProps: map[string]interface{}{
		"runtime":     "python3.11",
		"handler":     "index.handler",
		"memory_size": 128,
	},
	PropSchema: []PropDef{
		{Key: "runtime", Label: "Runtime", Type: PropTypeString, Required: true},
		{Key: "handler", Label: "Handler", Type: PropTypeString, Required: true},
		{Key: "memory_size", Label: "Memory (MB)", Type: PropTypeInt},
	},
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/aws/resources/ec2.go internal/aws/resources/ecs.go internal/aws/resources/lambda.go
git commit -m "feat: add compute resource definitions (EC2, ECS, Lambda)"
```

---

### Task 8: Database, Storage, and Load Balancing resource definitions

**Files:**
- Create: `internal/aws/resources/rds.go`
- Create: `internal/aws/resources/dynamodb.go`
- Create: `internal/aws/resources/s3.go`
- Create: `internal/aws/resources/alb.go`

- [ ] **Step 1: Create remaining resources**

```go
// internal/aws/resources/rds.go
package resources

var RDS = &ResourceDef{
	TFType:        "aws_db_instance",
	DisplayName:   "RDS",
	Category:      "Databases",
	TFRefAttr:     "id",
	TFOutputAttr:  "id",
	ParentRefAttr: "db_subnet_group_name",
	DefaultProps: map[string]interface{}{
		"engine":            "mysql",
		"instance_class":    "db.t3.micro",
		"allocated_storage": 20,
		"username":          "admin",
		"password":          "changeme",
	},
	PropSchema: []PropDef{
		{Key: "engine", Label: "Engine", Type: PropTypeString, Required: true},
		{Key: "instance_class", Label: "Instance Class", Type: PropTypeString, Required: true},
		{Key: "allocated_storage", Label: "Storage (GB)", Type: PropTypeInt},
		{Key: "username", Label: "Username", Type: PropTypeString},
		{Key: "password", Label: "Password", Type: PropTypeString},
	},
}
```

```go
// internal/aws/resources/dynamodb.go
package resources

var DynamoDB = &ResourceDef{
	TFType:       "aws_dynamodb_table",
	DisplayName:  "DynamoDB",
	Category:     "Databases",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"billing_mode": "PAY_PER_REQUEST",
		"hash_key":     "id",
	},
	PropSchema: []PropDef{
		{Key: "billing_mode", Label: "Billing Mode", Type: PropTypeString},
		{Key: "hash_key", Label: "Hash Key", Type: PropTypeString, Required: true},
	},
}
```

```go
// internal/aws/resources/s3.go
package resources

var S3 = &ResourceDef{
	TFType:       "aws_s3_bucket",
	DisplayName:  "S3",
	Category:     "Storage",
	TFRefAttr:    "bucket",
	TFOutputAttr: "bucket",
	DefaultProps: map[string]interface{}{
		"bucket":        "",
		"force_destroy": false,
	},
	PropSchema: []PropDef{
		{Key: "bucket", Label: "Bucket Name", Type: PropTypeString, Required: true},
		{Key: "force_destroy", Label: "Force Destroy", Type: PropTypeBool},
	},
}
```

```go
// internal/aws/resources/alb.go
package resources

// ALB has no ParentRefAttr: subnet association uses a list (not supported
// by the single-value cross-reference generator in V1). Connect ALB to
// a subnet visually but the HCL cross-reference is not auto-generated.
var ALB = &ResourceDef{
	TFType:       "aws_lb",
	DisplayName:  "Application Load Balancer",
	Category:     "Load Balancing",
	TFRefAttr:    "id",
	TFOutputAttr: "id",
	DefaultProps: map[string]interface{}{
		"internal":           false,
		"load_balancer_type": "application",
	},
	PropSchema: []PropDef{
		{Key: "internal", Label: "Internal", Type: PropTypeBool},
		{Key: "load_balancer_type", Label: "Type", Type: PropTypeString},
	},
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/aws/resources/rds.go internal/aws/resources/dynamodb.go internal/aws/resources/s3.go internal/aws/resources/alb.go
git commit -m "feat: add database, storage, and load balancing resource definitions"
```

---

### Task 9: Catalog package

**Files:**
- Create: `internal/catalog/catalog.go`

- [ ] **Step 1: Create catalog.go**

```go
// internal/catalog/catalog.go
package catalog

import "cloudblocks-tui/internal/aws/resources"

// All returns all resource definitions in catalog display order.
func All() []*resources.ResourceDef {
	return all
}

// ByCategory returns a stable category order and a map of category → resources.
func ByCategory() ([]string, map[string][]*resources.ResourceDef) {
	order := []string{"Networking", "Compute", "Databases", "Storage", "Load Balancing"}
	m := make(map[string][]*resources.ResourceDef)
	for _, r := range all {
		m[r.Category] = append(m[r.Category], r)
	}
	return order, m
}

// ByTFType returns the ResourceDef for the given Terraform type, or nil.
func ByTFType(tfType string) *resources.ResourceDef {
	for _, r := range all {
		if r.TFType == tfType {
			return r
		}
	}
	return nil
}

var all = []*resources.ResourceDef{
	resources.VPC,
	resources.Subnet,
	resources.IGW,
	resources.NatGW,
	resources.SecurityGroup,
	resources.EC2,
	resources.ECS,
	resources.Lambda,
	resources.RDS,
	resources.DynamoDB,
	resources.S3,
	resources.ALB,
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/catalog/catalog.go
git commit -m "feat: add catalog package"
```

---

## Chunk 3: ASCII Renderer + Terraform Generator

### Task 10: ASCII Renderer

**Files:**
- Create: `internal/renderer/ascii.go`
- Create: `internal/renderer/ascii_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
	// child lines must be indented with └─
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/renderer/...
```

Expected: FAIL — `renderer.Render` undefined.

- [ ] **Step 3: Implement Render**

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/renderer/...
```

Expected: PASS — 4 tests.

- [ ] **Step 5: Commit**

```bash
git add internal/renderer/
git commit -m "feat: add ASCII tree renderer"
```

---

### Task 11: Terraform Generator

**Files:**
- Create: `internal/terraform/generator.go`
- Create: `internal/terraform/generator_test.go`

- [ ] **Step 1: Write failing tests**

```go
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
	// VPC has TFOutputAttr = "id" — should get an output block
	arch.AddNode(&graph.Node{
		ID:         "vpc-1",
		Type:       "aws_vpc",
		Name:       "vpc",
		Properties: map[string]interface{}{"cidr_block": "10.0.0.0/16", "enable_dns_hostnames": true},
	})
	// ECS has TFOutputAttr = "" — should NOT get an output block
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
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/terraform/...
```

Expected: FAIL — `tfgen.Generate` undefined.

- [ ] **Step 3: Implement generator.go**

```go
// internal/terraform/generator.go
package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/graph"
)

// Generate writes main.tf, variables.tf, and outputs.tf to outDir.
// lookup resolves a Terraform resource type string to its ResourceDef.
func Generate(arch *graph.Architecture, lookup func(string) *resources.ResourceDef, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}
	main, vars, outputs, err := buildHCL(arch, lookup)
	if err != nil {
		return err
	}
	files := map[string]string{
		"main.tf":      main,
		"variables.tf": vars,
		"outputs.tf":   outputs,
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(content), 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

func buildHCL(arch *graph.Architecture, lookup func(string) *resources.ResourceDef) (mainOut, varsOut, outputsOut string, err error) {
	var main, outputs strings.Builder

	main.WriteString(`terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region  = var.aws_region
  profile = "default"
}

`)

	// Build parent index: for each "to" node, list its parent IDs.
	parents := make(map[string][]string)
	for _, e := range arch.Edges {
		parents[e.To] = append(parents[e.To], e.From)
	}

	for _, id := range arch.NodeOrder {
		n, ok := arch.Nodes[id]
		if !ok {
			continue
		}
		def := lookup(n.Type)

		main.WriteString(fmt.Sprintf("resource %q %q {\n", n.Type, n.ID))

		// Build prop-type index for this resource.
		propTypes := make(map[string]resources.PropType)
		if def != nil {
			for _, pd := range def.PropSchema {
				propTypes[pd.Key] = pd.Type
			}
		}

		// Write properties in sorted order for determinism.
		keys := make([]string, 0, len(n.Properties))
		for k := range n.Properties {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			main.WriteString(fmt.Sprintf("  %s = %s\n", k, formatHCL(n.Properties[k], propTypes[k])))
		}

		// Write cross-references from parent edges.
		if def != nil && def.ParentRefAttr != "" {
			for _, parentID := range parents[id] {
				parent, ok := arch.Nodes[parentID]
				if !ok {
					continue
				}
				parentDef := lookup(parent.Type)
				refAttr := "id"
				if parentDef != nil && parentDef.TFRefAttr != "" {
					refAttr = parentDef.TFRefAttr
				}
				main.WriteString(fmt.Sprintf("  %s = %s.%s.%s\n",
					def.ParentRefAttr, parent.Type, parent.ID, refAttr))
			}
		}

		main.WriteString("}\n\n")

		// Emit output block if this resource type declares a TFOutputAttr.
		if def != nil && def.TFOutputAttr != "" {
			outputs.WriteString(fmt.Sprintf("output %q {\n  value = %s.%s.%s\n}\n\n",
				n.ID, n.Type, n.ID, def.TFOutputAttr))
		}
	}

	varsOut = `variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}`
	return main.String(), varsOut, outputs.String(), nil
}

// formatHCL formats v as an HCL literal.
// PropTypeString → quoted; PropTypeInt / PropTypeBool → unquoted.
func formatHCL(v interface{}, pt resources.PropType) string {
	switch pt {
	case resources.PropTypeInt, resources.PropTypeBool:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/terraform/...
```

Expected: PASS — 4 tests.

- [ ] **Step 5: Run all tests to confirm nothing is broken**

```bash
go test ./...
```

Expected: PASS for all packages that have tests.

- [ ] **Step 6: Commit**

```bash
git add internal/terraform/
git commit -m "feat: add Terraform HCL generator"
```

---

## Chunk 4: Deploy Runner + TUI Foundation

### Task 12: Deploy runner

**Files:**
- Create: `internal/deploy/runner.go`

No unit tests — requires a real Terraform binary. Verified manually in Task 20, Step 4 (manual smoke test).

- [ ] **Step 1: Create runner.go**

```go
// internal/deploy/runner.go
package deploy

import (
	"bufio"
	"fmt"
	"os/exec"
	"sync"
)

// Result is the outcome of a deployment run.
type Result struct {
	ExitCode int
	Err      error
}

// Run executes `terraform init` then `terraform apply -auto-approve` in workDir.
// Each line of stdout/stderr is sent to lines. The final Result is sent to done.
// Both channels are closed when Run completes.
// workDir must contain valid .tf files before Run is called.
func Run(workDir string, lines chan<- string, done chan<- Result) {
	go func() {
		defer close(lines)
		defer close(done)

		if err := runCmd(workDir, lines, "terraform", "init"); err != nil {
			done <- Result{ExitCode: 1, Err: fmt.Errorf("terraform init: %w", err)}
			return
		}
		if err := runCmd(workDir, lines, "terraform", "apply", "-auto-approve"); err != nil {
			done <- Result{ExitCode: 1, Err: fmt.Errorf("terraform apply: %w", err)}
			return
		}
		done <- Result{ExitCode: 0}
	}()
}

func runCmd(workDir string, lines chan<- string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	// wg ensures scanner goroutines finish before runCmd returns.
	// Without this, a goroutine could send to lines after close(lines) fires,
	// causing a panic.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	go func() {
		defer wg.Done()
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			lines <- "ERR: " + sc.Text()
		}
	}()

	err = cmd.Wait()
	wg.Wait() // wait for scanners to finish draining pipes before returning
	return err
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 3: Commit**

```bash
git add internal/deploy/runner.go
git commit -m "feat: add Terraform deploy runner with output streaming"
```

---

### Task 13: TUI messages, keymap, and layout

**Files:**
- Create: `internal/tui/messages.go`
- Create: `internal/tui/keymap.go`
- Create: `internal/tui/layout.go`

- [ ] **Step 1: Create messages.go**

```go
// internal/tui/messages.go
package tui

import "cloudblocks-tui/internal/aws/resources"

// AddNodeMsg is emitted by CatalogModel when the user adds a resource.
type AddNodeMsg struct{ Def *resources.ResourceDef }

// SelectNodeMsg is emitted by ArchModel when the user moves the cursor.
type SelectNodeMsg struct{ NodeID string }

// ConnectNodesMsg is emitted by ArchModel when connect mode completes.
type ConnectNodesMsg struct{ From, To string }

// DeleteNodeMsg is emitted by ArchModel when the user deletes a node.
type DeleteNodeMsg struct{ NodeID string }

// RenameNodeMsg is emitted by ArchModel when a rename is confirmed.
type RenameNodeMsg struct{ NodeID, Name string }

// UpdatePropMsg is emitted by PropsModel when a property is edited.
type UpdatePropMsg struct {
	NodeID string
	Key    string
	Value  interface{}
}

// DeployLineMsg carries one line of terraform output.
type DeployLineMsg struct{ Line string }

// DeployDoneMsg signals that the deploy subprocess has exited.
type DeployDoneMsg struct{ ExitCode int }

// StatusMsg sets a transient status bar message.
type StatusMsg struct{ Text string }
```

- [ ] **Step 2: Create keymap.go**

```go
// internal/tui/keymap.go
package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Tab     key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Add     key.Binding
	Connect key.Binding
	Delete  key.Binding
	Rename  key.Binding
	Edit    key.Binding
	Save    key.Binding
	Export  key.Binding
	Deploy  key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Connect: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connect")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Rename:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit props")),
		Save:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save")),
		Export:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export TF")),
		Deploy:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "deploy")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
```

- [ ] **Step 3: Create layout.go**

```go
// internal/tui/layout.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	actionKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))
)

// panelSizes returns (catalogW, archW, propsW) from total terminal width.
// Each panel width excludes its border and padding (the caller passes the
// inner width to the sub-model).
func panelSizes(totalWidth int) (int, int, int) {
	// 3 panels × (2 border cols + 2 padding cols) = 12 chars overhead
	available := totalWidth - 12
	if available < 60 {
		available = 60
	}
	catalogW := available * 20 / 100
	propsW := available * 28 / 100
	archW := available - catalogW - propsW
	return catalogW, archW, propsW
}
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/messages.go internal/tui/keymap.go internal/tui/layout.go
git commit -m "feat: add TUI messages, keymap, and layout styles"
```

---

### Task 14: TUI root model (app.go)

**Files:**
- Create: `internal/tui/app.go`

- [ ] **Step 1: Create app.go**

```go
// internal/tui/app.go
package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/catalog"
	"cloudblocks-tui/internal/deploy"
	"cloudblocks-tui/internal/graph"
	tfgen "cloudblocks-tui/internal/terraform"
	"cloudblocks-tui/internal/tui/views"
)

// Panel identifies which panel currently has focus.
type Panel int

const (
	PanelCatalog Panel = iota
	PanelArchitecture
	PanelProperties
)

const savePath = "cloudblocks.json"
const generatedDir = "generated"

// Model is the root Bubble Tea model. It owns the Architecture and routes
// messages to the focused sub-model.
type Model struct {
	arch    *graph.Architecture
	keys    KeyMap
	focused Panel
	dirty   bool
	tfOK    bool // whether terraform binary is found

	catalogV    views.CatalogModel
	archV       views.ArchModel
	propsV      views.PropsModel
	deployV     views.DeployModel
	deployActive bool

	// deploy channels (set during deploy initiation)
	deployLines chan string
	deployDone  chan deploy.Result

	// startup load prompt
	loadPrompt bool
	// quit confirmation prompt
	quitPrompt bool

	statusMsg   string
	statusTimer int // counts Update ticks until status clears

	width, height int
}

// tickMsg is used to clear the status bar after a delay.
type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// New creates a new root Model.
// If showLoadPrompt is true, the model will ask the user whether to load
// cloudblocks.json before accepting other input.
func New(showLoadPrompt bool) Model {
	_, err := exec.LookPath("terraform")
	tfOK := err == nil

	m := Model{
		arch:       graph.New(),
		keys:       DefaultKeyMap(),
		focused:    PanelCatalog,
		tfOK:       tfOK,
		loadPrompt: showLoadPrompt,
	}
	m.catalogV = views.NewCatalog()
	m.archV = views.NewArch(m.arch)
	m.propsV = views.NewProps()
	m.deployV = views.NewDeploy()

	if !tfOK {
		m.statusMsg = "Terraform not found — deploy disabled"
		m.statusTimer = 0 // don't auto-clear
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return tick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		catalogW, archW, propsW := panelSizes(m.width)
		innerH := m.height - 4 // header + footer
		m.catalogV.SetSize(catalogW, innerH)
		m.archV.SetSize(archW, innerH)
		m.propsV.SetSize(propsW, innerH/2)
		m.deployV.SetSize(propsW, innerH/2)

	case tickMsg:
		if m.statusTimer > 0 {
			m.statusTimer--
			if m.statusTimer == 0 && m.tfOK {
				m.statusMsg = ""
			}
		}
		cmds = append(cmds, tick())

	case StatusMsg:
		m.statusMsg = msg.Text
		m.statusTimer = 3

	// --- load prompt handling ---
	case tea.KeyMsg:
		if m.loadPrompt {
			switch msg.String() {
			case "y", "Y":
				m.loadPrompt = false
				if err := m.arch.Load(savePath); err != nil {
					m.statusMsg = "Load failed: " + err.Error()
				} else {
					m.dirty = false
					m.archV = views.NewArch(m.arch)
					m.statusMsg = "Loaded cloudblocks.json"
					m.statusTimer = 3
				}
			case "n", "N", "esc":
				m.loadPrompt = false
			}
			return m, nil
		}

		if m.quitPrompt {
			switch msg.String() {
			case "y", "Y":
				return m, tea.Quit
			case "n", "N", "esc":
				m.quitPrompt = false
				m.statusMsg = ""
			}
			return m, nil
		}

		// Global keys (work regardless of focused panel)
		switch {
		case key.Matches(msg, m.keys.Escape):
			if m.deployActive {
				m.deployActive = false
				return m, nil
			}
			// If not in deploy mode, let Esc fall through to sub-model dispatch
			// (handles connect-mode cancel and property-edit cancel).

		case key.Matches(msg, m.keys.Quit):
			if m.dirty {
				m.quitPrompt = true
				m.statusMsg = "Unsaved changes. Quit? [Y/N]"
			} else {
				return m, tea.Quit
			}
			return m, nil

		case key.Matches(msg, m.keys.Tab):
			if m.focused == PanelCatalog {
				m.focused = PanelArchitecture
			} else if m.focused == PanelArchitecture {
				m.focused = PanelProperties
			} else {
				m.focused = PanelCatalog
			}
			return m, nil

		case key.Matches(msg, m.keys.Save):
			if err := m.arch.Save(savePath); err != nil {
				m.statusMsg = "Save failed: " + err.Error()
			} else {
				m.dirty = false
				m.statusMsg = "Saved to " + savePath
				m.statusTimer = 3
			}
			return m, nil

		case key.Matches(msg, m.keys.Export):
			if err := tfgen.Generate(m.arch, catalog.ByTFType, generatedDir); err != nil {
				m.statusMsg = "Export failed: " + err.Error()
			} else {
				abs, _ := filepath.Abs(generatedDir)
				m.statusMsg = "Exported to " + abs
				m.statusTimer = 3
			}
			return m, nil

		case key.Matches(msg, m.keys.Deploy):
			if !m.tfOK {
				m.statusMsg = "Terraform not found — cannot deploy"
				m.statusTimer = 3
				return m, nil
			}
			if err := tfgen.Generate(m.arch, catalog.ByTFType, generatedDir); err != nil {
				m.statusMsg = "TF generation failed: " + err.Error()
				m.statusTimer = 3
				return m, nil
			}
			m.deployLines = make(chan string, 1024) // large buffer prevents goroutine block on stdout
			m.deployDone = make(chan deploy.Result, 1)
			deploy.Run(generatedDir, m.deployLines, m.deployDone)
			m.deployActive = true
			m.deployV = views.NewDeploy()
			propsW := m.propsV.Width()
			innerH := m.height - 4
			m.deployV.SetSize(propsW, innerH/2)
			cmds = append(cmds, m.deployV.WaitForLine(m.deployLines, m.deployDone))
			return m, tea.Batch(cmds...)
		}
	}

	// Dispatch key events to the focused sub-model.
	if keyMsg, ok := msg.(tea.KeyMsg); ok && !m.loadPrompt && !m.quitPrompt {
		switch m.focused {
		case PanelCatalog:
			var cmd tea.Cmd
			m.catalogV, cmd = m.catalogV.Update(keyMsg)
			cmds = append(cmds, cmd)
		case PanelArchitecture:
			var cmd tea.Cmd
			m.archV, cmd = m.archV.Update(keyMsg)
			cmds = append(cmds, cmd)
		case PanelProperties:
			var cmd tea.Cmd
			m.propsV, cmd = m.propsV.Update(keyMsg)
			cmds = append(cmds, cmd)
		}
	}

	// Handle sub-model output messages.
	switch msg := msg.(type) {
	case AddNodeMsg:
		n := &graph.Node{
			ID:         fmt.Sprintf("%s-%d", msg.Def.TFType, len(m.arch.Nodes)+1),
			Type:       msg.Def.TFType,
			Name:       msg.Def.DisplayName,
			Properties: copyProps(msg.Def.DefaultProps),
		}
		m.arch.AddNode(n)
		m.dirty = true
		m.archV.Refresh(m.arch)

	case SelectNodeMsg:
		if n, ok := m.arch.Nodes[msg.NodeID]; ok {
			def := catalog.ByTFType(n.Type)
			m.propsV.SetNode(n, def)
		}

	case ConnectNodesMsg:
		// Reject self-loops (guard in ArchModel, but double-check here)
		if msg.From != msg.To {
			m.arch.Connect(msg.From, msg.To)
			m.dirty = true
			m.archV.Refresh(m.arch)
		}

	case DeleteNodeMsg:
		m.arch.RemoveNode(msg.NodeID)
		m.dirty = true
		m.archV.Refresh(m.arch)
		m.propsV.SetNode(nil, nil)

	case RenameNodeMsg:
		if n, ok := m.arch.Nodes[msg.NodeID]; ok {
			n.Name = msg.Name
			m.dirty = true
			m.archV.Refresh(m.arch)
		}

	case UpdatePropMsg:
		if n, ok := m.arch.Nodes[msg.NodeID]; ok {
			n.Properties[msg.Key] = msg.Value
			m.dirty = true
		}

	case DeployLineMsg:
		var cmd tea.Cmd
		m.deployV, cmd = m.deployV.Update(msg)
		cmds = append(cmds, cmd)
		// Re-dispatch to continue reading the next line from the channel.
		cmds = append(cmds, m.deployV.WaitForLine(m.deployLines, m.deployDone))

	case DeployDoneMsg:
		var cmd tea.Cmd
		m.deployV, cmd = m.deployV.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	catalogW, archW, propsW := panelSizes(m.width)

	// Header
	dirtyMark := ""
	if m.dirty {
		dirtyMark = "*"
	}
	nodeCount := len(m.arch.Nodes)
	savedStr := "saved"
	if m.dirty {
		savedStr = "unsaved"
	}
	headerLeft := titleStyle.Render("CloudBlocks TUI")
	headerMid := mutedStyle.Render(fmt.Sprintf("[%d nodes | %s%s]", nodeCount, savedStr, dirtyMark))
	headerRight := ""
	if m.statusMsg != "" {
		headerRight = m.statusMsg
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top,
		headerLeft+" ",
		headerMid,
		strings.Repeat(" ", max(0, m.width-lipgloss.Width(headerLeft)-lipgloss.Width(headerMid)-lipgloss.Width(headerRight)-2)),
		headerRight,
	)

	// Panels
	catStyle := panelStyle
	archStyle := panelStyle
	propsStyle := panelStyle
	if m.focused == PanelCatalog {
		catStyle = focusedPanelStyle
	} else if m.focused == PanelArchitecture {
		archStyle = focusedPanelStyle
	} else if m.focused == PanelProperties {
		propsStyle = focusedPanelStyle
	}

	catalogPanel := catStyle.Width(catalogW).Render(
		titleStyle.Render("CATALOG") + "\n" + m.catalogV.View(),
	)
	archLabel := "ARCHITECTURE"
	if m.archV.InConnectMode() {
		archLabel = "ARCHITECTURE [CONNECT]"
	}
	archPanel := archStyle.Width(archW).Render(
		titleStyle.Render(archLabel) + "\n" + m.archV.View(),
	)

	rightBottom := m.actionsView()
	if m.deployActive {
		rightBottom = m.deployV.View()
	}
	rightPanel := propsStyle.Width(propsW).Render(
		titleStyle.Render("PROPERTIES") + "\n" + m.propsV.View() +
			"\n" + titleStyle.Render("ACTIONS") + "\n" + rightBottom,
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, catalogPanel, archPanel, rightPanel)

	if m.loadPrompt {
		return header + "\n" + body + "\n" +
			m.statusMsg + "\n" +
			mutedStyle.Render("Found cloudblocks.json. Load it? [Y/N]")
	}

	return header + "\n" + body
}

func (m Model) actionsView() string {
	lines := []string{
		actionKeyStyle.Render("[S]") + " Save",
		actionKeyStyle.Render("[X]") + " Export TF",
	}
	if m.tfOK {
		lines = append(lines, actionKeyStyle.Render("[P]") + " Deploy")
	} else {
		lines = append(lines, mutedStyle.Render("[P] Deploy (terraform not found)"))
	}
	lines = append(lines, actionKeyStyle.Render("[Q]") + " Quit")
	return strings.Join(lines, "\n")
}

func copyProps(src map[string]interface{}) map[string]interface{} {
	dst := make(map[string]interface{}, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
```


- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0. (Views package doesn't exist yet — this step will fail until views are created in Chunk 5. That is expected. Come back to verify after Task 19.)

- [ ] **Step 3: Commit (deferred)**

`app.go` is committed as part of the combined `git add internal/tui/` commit in Task 19, Step 4 (after the full build passes). Do NOT commit app.go separately — it will not compile until all view files exist.

---

## Chunk 5: TUI Views + Final Integration

### Task 15: Views shared styles

**Files:**
- Create: `internal/tui/views/styles.go`

- [ ] **Step 1: Create styles.go**

```go
// internal/tui/views/styles.go
package views

import "github.com/charmbracelet/lipgloss"

// Shared lipgloss styles used across all view sub-models.
var (
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0 (other view files don't exist yet — this will compile in isolation).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/views/styles.go
git commit -m "feat: add shared view styles"
```

---

### Task 16: Catalog view

**Files:**
- Create: `internal/tui/views/catalog.go`

- [ ] **Step 1: Create catalog.go**

```go
// internal/tui/views/catalog.go
package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/catalog"
	"cloudblocks-tui/internal/tui"
)

// catalogItem is one row in the catalog list (either a category header or a resource).
type catalogItem struct {
	def      *resources.ResourceDef // nil if header
	label    string
	isHeader bool
}

// CatalogModel is the left-panel sub-model.
type CatalogModel struct {
	items  []catalogItem
	cursor int
	width  int
	height int
}

var (
	catHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	catItemStyle   = lipgloss.NewStyle().PaddingLeft(2)
	catSelectedStyle = lipgloss.NewStyle().PaddingLeft(2).
				Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
)

// NewCatalog returns an initialized CatalogModel.
func NewCatalog() CatalogModel {
	order, byCategory := catalog.ByCategory()
	var items []catalogItem
	for _, cat := range order {
		items = append(items, catalogItem{label: cat, isHeader: true})
		for _, def := range byCategory[cat] {
			items = append(items, catalogItem{def: def, label: def.DisplayName})
		}
	}
	// Start cursor on the first non-header item.
	cursor := 0
	for i, it := range items {
		if !it.isHeader {
			cursor = i
			break
		}
	}
	return CatalogModel{items: items, cursor: cursor}
}

func (m CatalogModel) SetSize(w, h int) CatalogModel {
	m.width = w
	m.height = h
	return m
}

func (m CatalogModel) Update(msg tea.Msg) (CatalogModel, tea.Cmd) {
	km := tui.DefaultKeyMap()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Up):
			m.cursor = m.prevResource(m.cursor)
		case key.Matches(msg, km.Down):
			m.cursor = m.nextResource(m.cursor)
		case key.Matches(msg, km.Add), key.Matches(msg, km.Enter):
			if m.cursor < len(m.items) && !m.items[m.cursor].isHeader {
				return m, func() tea.Msg {
					return tui.AddNodeMsg{Def: m.items[m.cursor].def}
				}
			}
		}
	}
	return m, nil
}

func (m CatalogModel) View() string {
	var sb strings.Builder
	for i, it := range m.items {
		if it.isHeader {
			sb.WriteString(catHeaderStyle.Render(it.label) + "\n")
		} else if i == m.cursor {
			sb.WriteString(catSelectedStyle.Render(it.label) + "\n")
		} else {
			sb.WriteString(catItemStyle.Render(it.label) + "\n")
		}
	}
	return sb.String()
}

// nextResource returns the index of the next non-header item after cursor.
func (m CatalogModel) nextResource(cursor int) int {
	for i := cursor + 1; i < len(m.items); i++ {
		if !m.items[i].isHeader {
			return i
		}
	}
	return cursor
}

// prevResource returns the index of the previous non-header item before cursor.
func (m CatalogModel) prevResource(cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if !m.items[i].isHeader {
			return i
		}
	}
	return cursor
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: May still fail (ArchModel, PropsModel, DeployModel not yet created). Continue to Task 17.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/views/catalog.go
git commit -m "feat: add catalog panel view"
```

---

### Task 17: Architecture view

**Files:**
- Create: `internal/tui/views/architecture.go`

- [ ] **Step 1: Create architecture.go**

```go
// internal/tui/views/architecture.go
package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/graph"
	"cloudblocks-tui/internal/renderer"
	"cloudblocks-tui/internal/tui"
)

// ArchModel is the center-panel sub-model for the architecture diagram.
type ArchModel struct {
	arch        *graph.Architecture
	flatList    []*graph.Node // DFS-ordered node list for cursor navigation
	cursor      int
	connectMode bool
	connectFrom string // node ID of the connect source
	renameMode  bool
	renameInput textinput.Model
	width       int
	height      int
}

var (
	archSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230"))
	archConnectSourceStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("214")).
				Foreground(lipgloss.Color("0"))
)

// NewArch creates a new ArchModel backed by arch.
func NewArch(arch *graph.Architecture) ArchModel {
	ti := textinput.New()
	ti.Placeholder = "new name"
	m := ArchModel{
		arch:        arch,
		renameInput: ti,
	}
	m.flatList = buildFlatList(arch)
	return m
}

// Refresh updates the model after the architecture has changed externally.
func (m ArchModel) Refresh(arch *graph.Architecture) ArchModel {
	m.arch = arch
	m.flatList = buildFlatList(arch)
	if m.cursor >= len(m.flatList) && len(m.flatList) > 0 {
		m.cursor = len(m.flatList) - 1
	}
	return m
}

// InConnectMode reports whether the model is in connect mode.
func (m ArchModel) InConnectMode() bool { return m.connectMode }

func (m ArchModel) SetSize(w, h int) ArchModel {
	m.width = w
	m.height = h
	return m
}

func (m ArchModel) Update(msg tea.Msg) (ArchModel, tea.Cmd) {
	km := tui.DefaultKeyMap()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Rename mode: capture text input.
		if m.renameMode {
			switch msg.String() {
			case "enter":
				newName := strings.TrimSpace(m.renameInput.Value())
				if newName == "" {
					newName = m.selectedNode().Name
				}
				m.renameMode = false
				nodeID := m.selectedNode().ID
				return m, func() tea.Msg {
					return tui.RenameNodeMsg{NodeID: nodeID, Name: newName}
				}
			case "esc":
				m.renameMode = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.renameInput, cmd = m.renameInput.Update(msg)
				return m, cmd
			}
		}

		switch {
		case key.Matches(msg, km.Up):
			if m.cursor > 0 {
				m.cursor--
			}
			return m, m.emitSelect()

		case key.Matches(msg, km.Down):
			if m.cursor < len(m.flatList)-1 {
				m.cursor++
			}
			return m, m.emitSelect()

		case key.Matches(msg, km.Connect):
			if !m.connectMode && m.selectedNode() != nil {
				m.connectMode = true
				m.connectFrom = m.selectedNode().ID
			}
			return m, func() tea.Msg {
				return tui.StatusMsg{Text: "Select target to connect (Esc to cancel)"}
			}

		case key.Matches(msg, km.Enter):
			if m.connectMode && m.selectedNode() != nil {
				target := m.selectedNode().ID
				if target == m.connectFrom {
					// Spec: self-loop rejected, connect mode stays active
					return m, func() tea.Msg {
						return tui.StatusMsg{Text: "Cannot connect a resource to itself"}
					}
				}
				from := m.connectFrom
				m.connectMode = false
				m.connectFrom = ""
				return m, func() tea.Msg {
					return tui.ConnectNodesMsg{From: from, To: target}
				}
			}

		case key.Matches(msg, km.Escape):
			if m.connectMode {
				m.connectMode = false
				m.connectFrom = ""
			}
			return m, nil

		case key.Matches(msg, km.Delete):
			if m.selectedNode() != nil && !m.connectMode {
				nodeID := m.selectedNode().ID
				return m, func() tea.Msg {
					return tui.DeleteNodeMsg{NodeID: nodeID}
				}
			}

		case key.Matches(msg, km.Rename):
			if m.selectedNode() != nil && !m.connectMode {
				m.renameMode = true
				m.renameInput.SetValue(m.selectedNode().Name)
				m.renameInput.Focus()
			}

		case key.Matches(msg, km.Edit):
			if m.selectedNode() != nil {
				return m, m.emitSelect()
			}
		}
	}
	return m, nil
}

func (m ArchModel) View() string {
	if m.renameMode && m.selectedNode() != nil {
		return "Rename: " + m.renameInput.View()
	}

	tree := renderer.Render(m.arch)
	if tree == "" {
		return mutedStyle.Render("(empty — press A in Catalog to add resources)")
	}

	// Highlight the selected node line.
	lines := strings.Split(tree, "\n")
	var sb strings.Builder
	for i, line := range lines {
		// Match selected node by looking for "(nodeID)" in the line.
		if m.selectedNode() != nil && strings.Contains(line, "("+m.selectedNode().ID+")") {
			if m.connectMode && m.selectedNode().ID == m.connectFrom {
				sb.WriteString(archConnectSourceStyle.Render(line))
			} else {
				sb.WriteString(archSelectedStyle.Render(line))
			}
		} else {
			sb.WriteString(line)
		}
		if i < len(lines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m ArchModel) selectedNode() *graph.Node {
	if len(m.flatList) == 0 || m.cursor < 0 || m.cursor >= len(m.flatList) {
		return nil
	}
	return m.flatList[m.cursor]
}

func (m ArchModel) emitSelect() tea.Cmd {
	n := m.selectedNode()
	if n == nil {
		return nil
	}
	return func() tea.Msg { return tui.SelectNodeMsg{NodeID: n.ID} }
}

// buildFlatList returns all nodes in DFS order starting from roots.
// This gives a consistent traversal order for cursor navigation.
func buildFlatList(arch *graph.Architecture) []*graph.Node {
	var result []*graph.Node
	visited := make(map[string]bool)
	var dfs func(n *graph.Node)
	dfs = func(n *graph.Node) {
		if visited[n.ID] {
			return
		}
		visited[n.ID] = true
		result = append(result, n)
		for _, child := range arch.Children(n.ID) {
			dfs(child)
		}
	}
	for _, root := range arch.Roots() {
		dfs(root)
	}
	// Also include any nodes not reachable via edges (isolated nodes).
	for _, id := range arch.NodeOrder {
		if !visited[id] {
			if n, ok := arch.Nodes[id]; ok {
				result = append(result, n)
			}
		}
	}
	return result
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: May still fail (PropsModel, DeployModel not yet created). Continue to Task 18.

- [ ] **Step 3: Commit (deferred)**

`architecture.go` is committed as part of the combined `git add internal/tui/` commit in Task 19, Step 4 (after the full build passes). Do NOT commit it separately.

---

### Task 18: Properties view

**Files:**
- Create: `internal/tui/views/properties.go`

- [ ] **Step 1: Create properties.go**

```go
// internal/tui/views/properties.go
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/graph"
	"cloudblocks-tui/internal/tui"
)

// PropsModel is the top-right panel for editing node properties.
type PropsModel struct {
	node    *graph.Node
	def     *resources.ResourceDef
	cursor  int
	editing bool
	input   textinput.Model
	width   int
	height  int
}

var (
	propKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	propValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("222"))
)

// NewProps returns an empty PropsModel (no node selected yet).
func NewProps() PropsModel {
	ti := textinput.New()
	return PropsModel{input: ti}
}

// SetNode updates the model with a new node and its ResourceDef.
// Passing nil clears the panel.
func (m PropsModel) SetNode(n *graph.Node, def *resources.ResourceDef) PropsModel {
	m.node = n
	m.def = def
	m.cursor = 0
	m.editing = false
	return m
}

func (m PropsModel) Width() int { return m.width }

func (m PropsModel) SetSize(w, h int) PropsModel {
	m.width = w
	m.height = h
	return m
}

func (m PropsModel) Update(msg tea.Msg) (PropsModel, tea.Cmd) {
	if m.node == nil || m.def == nil {
		return m, nil
	}
	km := tui.DefaultKeyMap()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				pd := m.def.PropSchema[m.cursor]
				raw := m.input.Value()
				val := parseValue(raw, pd.Type)
				nodeID := m.node.ID
				m.editing = false
				m.input.Blur()
				return m, func() tea.Msg {
					return tui.UpdatePropMsg{NodeID: nodeID, Key: pd.Key, Value: val}
				}
			case "esc":
				m.editing = false
				m.input.Blur()
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		} else {
			switch {
			case key.Matches(msg, km.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, km.Down):
				if m.cursor < len(m.def.PropSchema)-1 {
					m.cursor++
				}
			case key.Matches(msg, km.Enter):
				if len(m.def.PropSchema) > 0 {
					pd := m.def.PropSchema[m.cursor]
					current := fmt.Sprintf("%v", m.node.Properties[pd.Key])
					m.input.SetValue(current)
					m.input.Focus()
					m.editing = true
				}
			}
		}
	}
	return m, nil
}

func (m PropsModel) View() string {
	if m.node == nil {
		return mutedStyle.Render("(no resource selected)")
	}

	var sb strings.Builder
	sb.WriteString(propKeyStyle.Render(m.node.Name) + "\n")

	if m.def == nil || len(m.def.PropSchema) == 0 {
		sb.WriteString(mutedStyle.Render("(no configurable properties)"))
		return sb.String()
	}

	for i, pd := range m.def.PropSchema {
		val := m.node.Properties[pd.Key]
		if i == m.cursor && m.editing {
			sb.WriteString(propKeyStyle.Render(pd.Label+": ") + m.input.View() + "\n")
		} else if i == m.cursor {
			line := propKeyStyle.Render(pd.Label+": ") + propValueStyle.Render(fmt.Sprintf("%v", val))
			sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line) + "\n")
		} else {
			sb.WriteString(propKeyStyle.Render(pd.Label+": ") + propValueStyle.Render(fmt.Sprintf("%v", val)) + "\n")
		}
	}
	return sb.String()
}

// parseValue converts a string input to the appropriate Go type based on PropType.
func parseValue(raw string, pt resources.PropType) interface{} {
	switch pt {
	case resources.PropTypeInt:
		var n int
		fmt.Sscanf(raw, "%d", &n)
		return n
	case resources.PropTypeBool:
		return raw == "true" || raw == "1" || raw == "yes"
	default:
		return raw
	}
}
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```

Expected: exits 0 (deploy view not yet created — expected failure; continue to next task).

- [ ] **Step 3: Commit (deferred to Task 19)**

`properties.go` is committed as part of the combined `git add internal/tui/` commit in Task 19, Step 4.

---

### Task 19: Deploy view

**Files:**
- Create: `internal/tui/views/deploy.go`

- [ ] **Step 1: Create deploy.go**

```go
// internal/tui/views/deploy.go
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/deploy"
	"cloudblocks-tui/internal/tui"
)

// DeployModel is the bottom-right panel that shows live terraform output.
type DeployModel struct {
	lines    []string
	done     bool
	exitCode int
	vp       viewport.Model
	width    int
	height   int
}

var deployOutputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

// NewDeploy returns a fresh DeployModel (no output yet).
func NewDeploy() DeployModel {
	return DeployModel{}
}

func (m DeployModel) SetSize(w, h int) DeployModel {
	m.width = w
	m.height = h
	m.vp = viewport.New(w, h)
	return m
}

// WaitForLine returns a tea.Cmd that reads the next line from lines.
// When lines is closed (all output read), it reads the final Result from done
// and returns DeployDoneMsg. runner.go closes done before lines (LIFO defers),
// so by the time lines closes, done already has a value ready to read.
func (m DeployModel) WaitForLine(lines <-chan string, done <-chan deploy.Result) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if ok {
			return tui.DeployLineMsg{Line: line}
		}
		// lines closed — read the final result from done
		result := <-done
		return tui.DeployDoneMsg{ExitCode: result.ExitCode}
	}
}

func (m DeployModel) Update(msg tea.Msg) (DeployModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tui.DeployLineMsg:
		m.lines = append(m.lines, msg.Line)
		m.vp.SetContent(strings.Join(m.lines, "\n"))
		m.vp.GotoBottom()
		// Continue reading from the channel by re-issuing WaitForLine.
		// The channel references are not stored in DeployModel; the caller
		// (app.go) must re-dispatch WaitForLine when it receives a DeployLineMsg.
		// This is handled in app.go's Update by checking for DeployLineMsg and
		// calling m.deployV.WaitForLine again. See note below.

	case tui.DeployDoneMsg:
		m.done = true
		m.exitCode = msg.ExitCode
		status := "Deploy complete ✓"
		if msg.ExitCode != 0 {
			status = fmt.Sprintf("Deploy failed (exit %d)", msg.ExitCode)
		}
		m.lines = append(m.lines, "", status)
		m.vp.SetContent(strings.Join(m.lines, "\n"))
		m.vp.GotoBottom()
	}
	return m, nil
}

func (m DeployModel) View() string {
	if m.width == 0 {
		return ""
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Render("DEPLOY OUTPUT")
	return header + "\n" + deployOutputStyle.Render(m.vp.View())
}
```

Note: The channel streaming is already fully wired in `app.go` (Task 14). No additional changes to `app.go` are needed for streaming.

- [ ] **Step 2: Verify app.go is already complete**

No edits to `app.go` needed — the `deployLines`/`deployDone` struct fields and the `WaitForLine` re-dispatch are already in the Task 14 listing. Confirm `app.go` contains `deployLines chan string` in the Model struct and `m.deployV.WaitForLine(m.deployLines, m.deployDone)` in the `DeployLineMsg` case. Do NOT add them again.

- [ ] **Step 3: Verify build**

```bash
go build ./...
```

Expected: exits 0 for all packages.

- [ ] **Step 4: Commit**

```bash
git add internal/tui/
git commit -m "feat: add TUI views (catalog, architecture, properties, deploy)"
```

---

### Task 20: Wire main.go and final integration

**Files:**
- Modify: `cmd/cloudblocks/main.go`

- [ ] **Step 1: Update main.go**

```go
// cmd/cloudblocks/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cloudblocks-tui/internal/tui"
)

func main() {
	// Detect if cloudblocks.json exists to show the load prompt.
	_, err := os.Stat("cloudblocks.json")
	showLoadPrompt := err == nil

	m := tui.New(showLoadPrompt)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Final build**

```bash
go build ./cmd/cloudblocks/
```

Expected: exits 0. Binary `cloudblocks` (or `cloudblocks.exe` on Windows) created.

- [ ] **Step 3: Run all tests**

```bash
go test ./...
```

Expected: PASS for `internal/graph`, `internal/renderer`, `internal/terraform`.

- [ ] **Step 4: Manual smoke test**

Run: `./cloudblocks`

Perform the following sequence:
1. Press `A` in the Catalog on "VPC" → VPC appears in Architecture panel
2. Press `A` on "Subnet" → Subnet appears
3. Press `TAB` to focus Architecture panel
4. Navigate to VPC, press `C` (connect mode), navigate to Subnet, press `Enter` → edge created, ASCII shows `VPC └─ Subnet`
5. Press `TAB` to focus Properties, press `Enter` on `cidr_block`, change value, press `Enter` to confirm
6. Press `X` → `./generated/main.tf` is created
7. Inspect `./generated/main.tf` — verify `vpc_id = aws_vpc.aws_vpc-1.id` cross-reference in subnet block
8. Press `S` → "Saved to cloudblocks.json" in status bar
9. Press `Q` → exits

Expected: all steps work without crashes.

- [ ] **Step 5: Commit**

```bash
git add cmd/cloudblocks/main.go
git commit -m "feat: wire main.go; CloudBlocks TUI MVP complete"
```

---

## Build & Run Instructions

```bash
# Build
go build -o cloudblocks ./cmd/cloudblocks/

# Run
./cloudblocks

# Run tests
go test ./...

# Build for current platform
GOOS=linux GOARCH=amd64 go build -o cloudblocks-linux ./cmd/cloudblocks/
```

**Prerequisites:**
- Go 1.21+
- `terraform` in `$PATH` for deploy functionality (export works without it)
- AWS CLI configured (`~/.aws/credentials` or environment variables) for actual deployment
