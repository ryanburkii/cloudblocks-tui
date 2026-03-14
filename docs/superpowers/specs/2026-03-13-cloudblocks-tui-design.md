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
    NodeOrder []string // insertion order; used to display unconnected nodes in a consistent list order
}
```

Methods: `AddNode`, `RemoveNode`, `Connect`, `Children(id)`, `Roots()`, `Save(path)`, `Load(path)`.

Note: `Disconnect` (removing individual edges) is not exposed in V1 — users remove edges by deleting and re-adding nodes.

**Roots()** returns all nodes that have no incoming edges (i.e. no other node has an edge pointing to them). These are the top-level nodes rendered at the root of the ASCII tree. `NodeOrder` determines the display order of these root nodes.

### ResourceDef

```go
type ResourceDef struct {
    TFType        string
    DisplayName   string
    Category      string
    DefaultProps  map[string]interface{}
    PropSchema    []PropDef
    ParentRefAttr string // HCL attribute name on this resource that references its parent
                         // e.g. "vpc_id" for Subnet, "subnet_id" for EC2, "" if no parent ref needed
    TFRefAttr     string // HCL attribute on this resource used as the RHS of cross-references
                         // i.e. the trailing attribute when another resource references this one
                         // e.g. "id" for VPC/Subnet/EC2, "arn" for Lambda
                         // Always "id" unless the resource uses a different primary reference attribute
    TFOutputAttr  string // HCL attribute exposed in outputs.tf (e.g. "id", "arn", "bucket")
                         // if empty, no output block is emitted for this resource type
}

type PropDef struct {
    Key      string
    Label    string
    Type     PropType  // String | Int | Bool
    Required bool
}

type PropType string

const (
    PropTypeString PropType = "string"  // renders text input; HCL value is quoted
    PropTypeInt    PropType = "int"     // renders text input (numeric); HCL value is unquoted
    PropTypeBool   PropType = "bool"    // renders text input (true/false); HCL value is unquoted
)
```

`PropSchema` is the single source of truth for:
- The property editor UI (which input type to render)
- Terraform HCL generation (whether to quote the value)

`ParentRefAttr` tells the generator which HCL attribute to populate with the parent node's resource reference when an edge exists (e.g. a Subnet node with `ParentRefAttr = "vpc_id"` and an edge from a VPC will emit `vpc_id = aws_vpc.<id>.id`). If `ParentRefAttr` is empty, the edge creates no HCL cross-reference.

`TFOutputAttr` tells the generator which HCL attribute to expose in `outputs.tf` (e.g. `"id"` for VPC/Subnet/EC2, `"arn"` for Lambda, `"bucket"` for S3). If empty, no output block is emitted for that resource type.

### Persistence Serialization

The in-memory `Architecture` uses a `map[string]*Node` for O(1) lookups, but JSON serialization uses a flat struct:

```go
type ArchitectureJSON struct {
    Nodes []*Node `json:"nodes"`
    Edges []Edge  `json:"edges"`
}
```

`Save` converts the map to a slice ordered by `NodeOrder` before marshalling. `Load` rebuilds the map from the slice and reconstructs `NodeOrder` from the slice's positional order (i.e. `NodeOrder[i] = nodes[i].ID`). No separate `NodeOrder` field is stored in JSON — slice position is the canonical order. Loading clears the dirty flag.

---

## TUI Layout

Classic 3-column layout:

```
┌─────────────────────────────────────────────────────────────┐
│  CloudBlocks TUI  [N nodes | unsaved*]   [status message]   │
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

### Status Bar

The header status bar displays: `[N nodes | saved]` or `[N nodes | unsaved*]` on the left, and a transient status message on the right (e.g. `"Saved to cloudblocks.json"`, `"Terraform not found — deploy disabled"`, `"Exported to ./generated/"`). Transient messages clear after 3 seconds.

### Deploy Panel Expansion

When deployment is triggered (`P`), the Actions sub-panel (bottom-right) is replaced by a scrollable deploy output panel that streams `terraform init` and `terraform apply` output. The Catalog, Architecture, and Properties panels remain visible. The panel label changes to `DEPLOY OUTPUT`. When deploy completes (success or failure), the panel shows the exit status and waits. Pressing `Esc` closes the deploy panel and restores the normal Actions panel. Pressing `Q` while the deploy panel is active triggers the normal quit flow (status bar prompt `"Unsaved changes. Quit? [Y/N]"` if dirty, or immediate exit if clean) — it does NOT close the deploy panel.

---

## Key Bindings

| Key | Context | Action |
|-----|---------|--------|
| `TAB` | Any | Cycle focus: Catalog → Architecture → Properties → (repeat) |
| `↑` / `↓` | Catalog, Architecture, Properties | Navigate items |
| `A` / `Enter` | Catalog | Add selected resource to architecture |
| `C` | Architecture (normal) | Enter connect mode |
| `Enter` | Architecture (normal) | No-op (selection only via ↑↓) |
| `Enter` | Architecture (connect mode) | Create edge from source to currently selected node |
| `Esc` | Architecture (connect mode) | Cancel connect mode |
| `D` | Architecture (normal) | Delete selected node |
| `R` | Architecture (normal) | Rename selected node (inline — see Rename section) |
| `E` | Architecture (normal) | Focus Properties panel |
| `Enter` | Properties | Edit focused field (inline text input) |
| `Esc` | Properties (editing) | Cancel edit, discard changes |
| `Enter` | Properties (editing) | Confirm edit, update node property |
| `S` | Any | Save architecture to `cloudblocks.json` |
| `X` | Any | Export Terraform to `./generated/` |
| `P` | Any | Deploy (generate + terraform init + apply) |
| `Q` | Any | Quit (see Quit section) |

---

## Connect Mode

Two-step interaction:
1. Select source node in Architecture panel, press `C` → status bar shows `"Select target to connect (Esc to cancel)"` and the panel label changes to `ARCHITECTURE [CONNECT]`
2. Navigate to target node with `↑`/`↓` → press `Enter` to create edge and return to normal mode

Pressing `Esc` at any point cancels connect mode and returns to normal mode.

If the user navigates back to the source node and presses `Enter` (self-loop), the connection is silently rejected and the status bar shows `"Cannot connect a resource to itself"`. Connect mode remains active.

---

## Rename

Pressing `R` in the Architecture panel (normal mode) on a selected node opens an inline text input within the Architecture panel, pre-filled with the node's current name. The user edits the name and presses `Enter` to confirm or `Esc` to cancel. The ASCII tree updates immediately on confirmation.

---

## Quit

Pressing `Q` from any context:
- If there are no unsaved changes: exit immediately
- If there are unsaved changes: the status bar shows `"Unsaved changes. Quit? [Y/N]"`. Pressing `Y` exits; pressing `N` or `Esc` cancels and returns to normal operation.

---

## Terraform Generation

- Each node → one HCL resource block in `main.tf`
- `PropDef.Type` determines HCL value quoting: `string` values are quoted, `int` and `bool` values are unquoted
- Edges drive cross-references: for each edge `(from → to)`, the generator looks up `ResourceDef.ParentRefAttr` for the `to` node's resource type and emits `<to.ParentRefAttr> = <from_tf_type>.<from_id>.<from.TFRefAttr>` inside the `to` block. If `ParentRefAttr` is empty for the `to` node, no cross-reference is emitted. `TFRefAttr` (from the `from` node's `ResourceDef`) is the trailing HCL attribute (e.g. `id`, `arn`) and is always "id" unless the resource explicitly defines otherwise.
- `variables.tf` — AWS region variable
- `outputs.tf` — for each node whose `ResourceDef.TFOutputAttr` is non-empty, emits an output block exposing that attribute (e.g. `id` for VPC/Subnet/EC2, `bucket` for S3, `arn` for Lambda). Nodes with empty `TFOutputAttr` are skipped.
- AWS provider block uses `profile = "default"` (reads `~/.aws/credentials`)
- Output directory: `./generated/`

### Export vs Deploy

- **Export (`X`):** Writes `./generated/*.tf`, shows `"Exported to ./generated/"` in status bar
- **Deploy (`P`):** Writes `./generated/*.tf`, then runs `terraform init` followed by `terraform apply -auto-approve`; deploy panel (replaces Actions sub-panel) shows live streamed output. `--auto-approve` is always passed — no interactive confirmation is expected from the user inside the TUI.

---

## Persistence

- **Save (`S`):** Serializes `Architecture` to `cloudblocks.json` via `ArchitectureJSON` (nodes as ordered array, edges as array). Clears the dirty flag.
- **Load:** On startup, if `cloudblocks.json` exists in the current directory, the status bar shows `"Found cloudblocks.json. Load it? [Y/N]"` before any other interaction is possible. Pressing `Y` loads the file, clears the dirty flag, and starts normal operation. Pressing `N` (or `Esc`) starts with an empty architecture. No other keys are processed until the prompt is answered.
- Unsaved changes are tracked by a dirty flag on the root model; set on any mutation, cleared on save or load.

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Terraform not installed | Detected at startup via `exec.LookPath("terraform")`; status bar warning; Export still works; Deploy key (`P`) shows `"Terraform not found"` message |
| `terraform apply` fails | Non-zero exit code shown in deploy panel (`"Deploy failed (exit 1)"`); output preserved in panel for review |
| Save/load errors | Shown in status bar as transient message; non-fatal |
| AWS credentials missing | Not pre-validated; terraform surfaces errors during deploy via streamed output |

---

## Testing Strategy

- **Unit tests:** `internal/graph` (add/remove/connect operations), `internal/renderer` (ASCII output for known topologies), `internal/terraform` (HCL generation — verify quoting behavior per `PropType`, cross-references, outputs)
- **No TUI unit tests in MVP** — covered by manual smoke test
- **Manual smoke test:** Build the `VPC → Subnet → ALB → ECS → RDS` example from CLAUDE.md, export TF, verify valid HCL
