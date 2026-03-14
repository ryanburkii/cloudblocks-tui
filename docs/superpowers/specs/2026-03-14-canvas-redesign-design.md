# CloudBlocks TUI — Canvas Redesign Design Spec

**Date:** 2026-03-14
**Status:** Approved

---

## Overview

Replace the current ASCII tree view in the Architecture panel with a free-form block diagram canvas. Users place resource blocks on a 2D virtual canvas, move them with arrow keys, and connect them via three interaction modes. Connections route at 90° angles (orthogonal). The Catalog, Properties, and Actions panels are unchanged.

---

## Goals

1. Replace ASCII tree with a scrollable 2D canvas of resource blocks
2. Each block displays resource type and name; selected block is highlighted
3. Blocks are movable on the canvas via move mode (`M`)
4. Three connection modes: smart placement (auto), port mode (`P`), manual connect (`C`)
5. Connections render as orthogonal (90°) lines with arrowheads
6. Save/load preserves block positions

---

## Non-Goals

- No mouse support
- No zoom or pan beyond viewport auto-scroll
- No collision detection or automatic layout (user positions blocks manually)
- No changes to Terraform generator, deploy runner, catalog, or properties panel

---

## Data Model Changes

### Node — add position fields

```go
type Node struct {
    ID         string
    Type       string
    Name       string
    Properties map[string]interface{}
    X, Y       int  // canvas grid position (top-left corner of block, in terminal cells)
}
```

X, Y serialize to JSON with the node. `Save`/`Load` require no other changes. New nodes are placed at a staggered default position to avoid overlap (e.g. offset by 4 rows per node added).

### ArchModel — add canvas state

```go
// internal/tui/views/architecture.go
type ArchModel struct {
    // existing fields preserved
    viewportX, viewportY int     // top-left corner of visible window into canvas
    selectedID           string
    moveMode             bool    // arrow keys reposition selected block
    connectMode          bool    // manual C-key connect (existing)
    portMode             bool    // drag-to-connect in progress
    portTargetID         string  // block currently hovered in port mode
    connectSourceID      string  // source for both connectMode and portMode
}
```

No changes to `Architecture`, `Edge`, or any other package.

---

## Canvas

### Virtual canvas size

200 columns × 60 rows (terminal cells). Much larger than the visible panel. Viewport auto-scrolls to keep the selected block in view with a margin of 4 cells on each side.

### Block dimensions

Each block is **16 columns wide × 4 rows tall** (including border). Contents:
- Row 1: `┌──────────────┐`
- Row 2: `│ <type>       │` — resource type, muted orange, truncated to 14 chars
- Row 3: `│ <name>       │` — node name, white, truncated to 14 chars
- Row 4: `└──────────────┘`

### Block states

| State | Border | Background |
|---|---|---|
| Normal | `#30363d` dim | `#161b22` |
| Selected | `#58a6ff` blue, 2px | `#161b22` + blue glow |
| Move mode | `#3fb950` green, 2px | `#1a2a1a` + `✥` suffix on name |
| Port mode source | `#3fb950` green, 2px | `#1a2a1a` + `⇥` suffix on name |
| Port hover target | `#f0883e` orange, 2px | `#161b22` |

### Connection rendering

Connections are rendered after all blocks. For each `Edge (from → to)`:

1. Compute exit point: right-center of `from` block
2. Compute entry point: left-center or top-center of `to` block (whichever requires fewer segments)
3. Route: horizontal-first if `to` is to the right of `from`; vertical-first if `to` is below
4. Draw segments using `─`, `│`, `┐`, `└`, `┘`, `┌`, `▶`, `▼`
5. Connections involving the selected block render in blue (`#58a6ff`); all others in dim grey (`#30363d`)

In port mode, a dashed green line (`- - -`) renders from the source block's exit point to the current port target block (or floating cursor position if no block hovered).

### Viewport auto-scroll

On every selection change or block move, check if the selected block is within the viewport bounds minus a 4-cell margin. If not, shift `viewportX`/`viewportY` to bring it into view.

---

## Interaction Model

### Normal mode (canvas focused)

| Key | Action |
|---|---|
| `↑` `↓` `←` `→` | Cycle to adjacent block (nearest in that direction) |
| `M` | Enter move mode |
| `P` | Enter port mode |
| `C` | Enter connect mode (existing behaviour) |
| `D` | Delete selected block |
| `R` | Rename selected block (inline, existing behaviour) |
| `E` | Focus Properties panel |
| `Tab` | Cycle panel focus |

### Move mode

1. Press `M` on selected block → block turns green, status bar: `"Moving <name> — arrows to reposition, Enter/M to drop, Esc to cancel"`
2. Arrow keys shift block position by 2 cells per press (snaps to even-cell grid)
3. Press `M` or `Enter` → drop block at new position, return to normal mode
4. Press `Esc` → restore original position, return to normal mode

### Port mode (drag-to-connect)

1. Press `P` on selected block → source block turns green with `⇥`, dashed line appears, status bar: `"Port mode — navigate to target, Enter to connect, Esc to cancel"`
2. Arrow keys move focus to the nearest block in that direction (same cycle logic as normal mode); hovered target highlights orange
3. Press `Enter` → create edge from source to target, return to normal mode
4. Press `Esc` → cancel, return to normal mode
5. Self-loop: silently rejected, status bar: `"Cannot connect a resource to itself"`
6. Duplicate edge: silently rejected, status bar: `"Connection already exists"`

### Manual connect mode (C key — unchanged)

Existing behaviour preserved exactly. Same self-loop and duplicate edge rejection.

### Smart placement (on add)

When the user adds a resource from the Catalog (`A` or `Enter`) whose `ResourceDef.ParentRefAttr` is non-empty:

1. Canvas dims (muted style applied to all blocks)
2. Status bar prompt: `"Connect <DisplayName> to parent: <node-name-1>  <node-name-2>  [none]  (↑↓ select, Enter confirm)"`
3. Only compatible parent nodes are listed — nodes whose `TFType` matches the expected parent for that `ParentRefAttr` (e.g. Subnet lists VPCs only). If no compatible nodes exist, the prompt is skipped and the block is placed unconnected.
4. `↑`/`↓` cycle through options including `[none]`; `Enter` confirms
5. New block is placed at a default position offset from the chosen parent (4 rows below, same column). If `[none]`, placed at next staggered default position.
6. If parent chosen: edge is created automatically (no need for C/P)
7. `Esc` acts as `[none]` — block placed unconnected

Resources with empty `ParentRefAttr` (VPC, IGW, S3, DynamoDB, Security Group, ALB) skip the prompt entirely and land on the canvas immediately.

---

## Package / File Changes

| File | Change |
|---|---|
| `internal/graph/node.go` | Add `X, Y int` fields |
| `internal/graph/architecture.go` | `AddNode` sets default staggered X, Y |
| `internal/tui/views/architecture.go` | Full rewrite of render + input handling |
| `internal/tui/tuicore/messages.go` | Add `MoveNodeMsg{ID string, X, Y int}` |

All other files unchanged.

---

## Testing Strategy

- **Unit tests:** `internal/graph` — `AddNode` sets non-zero X/Y; `Save`/`Load` round-trips X/Y correctly
- **Unit tests:** `internal/renderer` — removed (ASCII tree renderer is replaced; existing tests deleted)
- **Manual smoke test:** Add VPC → Subnet → ALB → ECS → RDS, connect them, move blocks around, verify connections re-route correctly, save and reload, verify positions persist

---

## Open Questions (resolved)

- **Arrow routing complexity:** Constrained to 90° orthogonal only. No diagonal lines.
- **Mouse support:** Explicitly out of scope for V2.
- **Auto-layout:** Not included — user positions blocks manually.
