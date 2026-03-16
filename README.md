# CloudBlocks TUI

A terminal-based AWS infrastructure builder. Design cloud architectures visually in your terminal, then deploy them with Terraform.

## Overview

CloudBlocks TUI lets you assemble AWS infrastructure diagrams using a keyboard-driven interface. Add resources, connect them, configure properties, generate Terraform, and deploy — all without leaving the terminal.

## Features

- **Visual architecture builder** — compose AWS resources into a topology diagram
- **Resource catalog** — browse 12 AWS resource types across 5 categories
- **Property editor** — configure resource parameters inline
- **Terraform generation** — export to `./generated/main.tf`, `variables.tf`, `outputs.tf`
- **Live deployment** — run `terraform init` + `terraform apply` with streaming output in the TUI
- **Persistence** — save and load architectures

## Supported Resources

| Category | Resources |
|---|---|
| Networking | VPC, Subnet, Internet Gateway, NAT Gateway, Security Group |
| Compute | EC2 Instance, ECS Service, Lambda Function |
| Databases | RDS, DynamoDB |
| Storage | S3 |
| Load Balancing | Application Load Balancer |

## Requirements

- Go 1.21+
- [Terraform](https://developer.hashicorp.com/terraform/install) (for deploy)
- AWS credentials configured via `~/.aws/credentials` or environment variables

## Build & Run

```bash
go build -o cloudblocks ./cmd/cloudblocks
./cloudblocks
```

Or run directly:

```bash
go run ./cmd/cloudblocks
```

## Keyboard Shortcuts

| Key | Action |
|---|---|
| `Tab` | Switch panel |
| `↑` / `↓` / `k` / `j` | Navigate |
| `Enter` | Select |
| `Esc` | Cancel / back |
| `a` | Add resource from catalog |
| `c` | Connect mode (link two resources) |
| `l` | Link resources |
| `m` | Move a resource block |
| `d` | Delete selected resource |
| `r` | Rename resource |
| `e` | Edit resource properties |
| `s` | Save architecture |
| `x` | Export Terraform files |
| `p` | Deploy (terraform init + apply) |
| `q` / `Ctrl+C` | Quit |

## Project Structure

```
cmd/cloudblocks/       # Entry point
internal/
  tui/                 # Bubble Tea UI (app, layout, keymap, views)
  graph/               # Architecture graph (nodes, edges)
  catalog/             # Resource catalog
  aws/resources/       # AWS resource definitions
  terraform/           # Terraform file generator
  deploy/              # Terraform runner (streaming output)
generated/             # Terraform output (git-ignored)
```

## Tech Stack

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — UI components
- Terraform CLI — infrastructure deployment
