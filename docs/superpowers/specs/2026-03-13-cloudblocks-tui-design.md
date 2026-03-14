# CloudBlocks TUI — Design Spec

**Date:** 2026-03-13
**Status:** Approved

---

## Overview

CloudBlocks TUI is a terminal-based AWS infrastructure builder written in Go. Users visually assemble cloud infrastructure diagrams in the terminal, configure resource properties, and deploy them via Terraform — or export the generated `.tf` files for manual deployment.

---

## Goals

1. Browse a catalog of AWS resources grouped by category
2. Add resources to an architecture and connect them to form a topology
3. Configure resource properties via an inline editor
4. View the architecture as an ASCII tree diagram
5. Save and load architectures to/from disk
6. Generate Terraform configuration files
7. Deploy infrastructure via `terraform init` + `terraform apply` with live output

---

## Non-Goals (V1)

- No code upload for Lambda (config only: runtime, handler, memory)
- No pre-validation of AWS credentials (terraform surfaces errors during deploy)
- No TUI unit tests (covered manually; graph/renderer/generator have unit tests)
- No undo/redo
- No multi-architecture workspace

---

## Supported AWS Resources

| Category | Resources |
|---|---|
| Networking | VPC, Subnet, Internet Gateway, NAT Gateway, Security Group |
| Compute | EC2 Instance, ECS Service, Lambda Function |
| Databases | RDS, DynamoDB |
| Storage | S3 |
| Load Balancing | Application Load Balancer |

---

## Tech Stack

- **Language:** Go
- **TUI framework:** Bubble Tea
- **Styling:** Lip Gloss
- **UI components:** Bubbles
- **Deployment:** Terraform CLI (detected at startup)

---

## Architecture

### Approach

Single Bubble Tea model with sub-models per panel. The root `app.go` model owns the shared `Architecture` struct and routes `tea.Msg` to whichever panel is focused. Sub-models emit typed messages; the root model mutates shared state and re-renders all panels.

### Package Structure

```
cmd/cloudblocks/main.go          — entry point

internal/
  graph/
    node.go                      — Node struct
    edge.go                      — Edge struct
    architecture.go              — Architecture (nodes, edges, save/load)

  catalog/
    resources.go                 — static catalog, all ResourceDefs grouped by category

  aws/resources/
    vpc.go, subnet.go, ec2.go,
    ecs.go, rds.go, s3.go,
    alb.go, igw.go, natgw.go,
    sg.go, dynamodb.go, lambda.go — one file per resource type

  renderer/
    ascii.go                     — renders Architecture as ASCII tree

  terraform/
    generator.go                 — walks graph, emits main.tf / variables.tf / outputs.tf

  deploy/
    runner.go                    — runs terraform init + apply, streams output via channel

  tui/
    app.go                       — root Model
    keymap.go                    — all keybindings
    layout.go                    — lipgloss panel layout
    views/
      catalog.go                 — catalog panel sub-model
      architecture.go            — architecture panel sub-model
      properties.go              — property editor sub-model
      deploy.go                  — deploy panel sub-model
```

---

## Data Model

### Node

```go
type Node struct {
    ID         string
    Type       string                 // e.g. "aws_vpc"
    Name       string
    Properties map[string]interface{}
}
```

### Edge

```go
type Edge struct {
    From string // Node ID
    To   string // Node ID
}
```

### Architecture

```go
type Architecture struct {
    Nodes     map[string]*Node
    Edges     []Edge
    NodeOrder []string // insertion order for consistent rendering
}
```

Methods: `AddNode`, `RemoveNode`, `Connect`, `Disconnect`, `Children(id)`, `Roots()`, `Save(path)`, `Load(path)`.

### ResourceDef

```go
type ResourceDef struct {
    TFType       string
    DisplayName  string
    Category     string
    DefaultProps map[string]interface{}
    PropSchema   []PropDef
}

type PropDef struct {
    Key      string
    Label    string
    Required bool
}
```

`PropSchema` is the single source of truth for both the property editor UI and Terraform HCL generation.

---

## TUI Layout

Classic 3-column layout:

```
┌─────────────────────────────────────────────────────────────┐
│  CloudBlocks TUI                          [status bar]       │
├──────────────┬──────────────────────┬───────────────────────┤
│   CATALOG    │    ARCHITECTURE      │     PROPERTIES        │
│              │                      │                       │
│  Networking  │  VPC (vpc-1)         │  ECS (ecs-1)          │
│    VPC       │  └─ Subnet (sub-1)   │  cpu:    512          │
│    Subnet    │     └─ ALB (alb-1)   │  memory: 1024         │
│    IGW       │        └─ ECS (ecs-1)│                       │
│  Compute     │           └─ RDS     ├───────────────────────┤
│    EC2       │                      │     ACTIONS           │
│    ECS       │                      │  [S] Save             │
│    Lambda    │                      │  [X] Export TF        │
│  Databases   │                      │  [P] Deploy           │
│    RDS       │                      │  [Q] Quit             │
│    DynamoDB  │                      │                       │
└──────────────┴──────────────────────┴───────────────────────┘
```

---

## Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `TAB` | Any | Cycle focus between panels |
| `↑` / `↓` | Catalog, Architecture, Properties | Navigate items |
| `A` / `Enter` | Catalog | Add selected resource to architecture |
| `C` | Architecture | Enter connect mode |
| `Enter` | Architecture (connect mode) | Connect to selected node |
| `Esc` | Architecture (connect mode) | Cancel connect |
| `D` | Architecture | Delete selected node |
| `R` | Architecture | Rename selected node |
| `E` / `Tab` | Architecture | Focus Properties panel |
| `Enter` | Properties | Edit focused field |
| `Esc` | Properties | Cancel edit |
| `S` | Any | Save architecture to `cloudblocks.json` |
| `X` | Any | Export Terraform to `./generated/` |
| `P` | Any | Deploy (generate + terraform init + apply) |
| `Q` | Any | Quit (prompt if unsaved changes) |

---

## Connect Mode

Two-step interaction:
1. Select source node in Architecture panel, press `C` → status bar shows `"Select target node to connect (Esc to cancel)"`
2. Navigate to target node → press `Enter` to create edge

---

## Terraform Generation

- Each node → one HCL resource block in `main.tf`
- Property map keys → HCL attributes
- Edges drive cross-references (e.g. `vpc_id = aws_vpc.vpc-1.id`)
- `variables.tf` — AWS region variable
- `outputs.tf` — IDs of all root-level resources
- AWS provider block uses `profile = "default"` (reads `~/.aws/credentials`)
- Output directory: `./generated/`

### Export vs Deploy

- **Export (`X`):** Writes `./generated/*.tf`, shows output path in status bar
- **Deploy (`P`):** Writes `./generated/*.tf`, then runs `terraform init` + `terraform apply`; deploy panel expands to show live streamed output

---

## Persistence

- **Save (`S`):** Serializes `Architecture` to `cloudblocks.json` as JSON
- **Load:** On startup, if `cloudblocks.json` exists in current directory, offer to load it
- Format: JSON with nodes array and edges array

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Terraform not installed | Detected at startup; status bar warning; Export still works; Deploy disabled |
| `terraform apply` fails | Exit code shown in deploy panel; output preserved for review |
| Save/load errors | Shown in status bar; non-fatal |
| AWS credentials missing | Not pre-validated; terraform surfaces errors during deploy |

---

## Testing Strategy

- **Unit tests:** `internal/graph` (add/remove/connect operations), `internal/renderer` (ASCII output), `internal/terraform` (HCL generation correctness)
- **No TUI unit tests in MVP** — covered by manual smoke test
- **Manual smoke test:** Build the `VPC → Subnet → ALB → ECS → RDS` example from CLAUDE.md, export TF, verify valid HCL
