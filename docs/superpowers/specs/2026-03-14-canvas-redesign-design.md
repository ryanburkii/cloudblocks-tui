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
4. Three connection modes: smart placement (auto), link mode (`L`), manual connect (`C`)
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

X, Y serialize to JSON with the node. `Save`/`Load` require no other changes.

**Default stagger for new nodes:** Computed by `ArchModel`, not by `graph.AddNode`. `ArchModel` calculates the next stagger position and passes explicit X, Y to `arch.AddNode(node)`. The stagger formula: first node at (2, 2); each subsequent node at (2, prevY + 6); when Y would exceed 54, X increments by 20 and Y resets to 2. Stagger index is `len(arch.Nodes)` before the new node is added. `graph.AddNode` simply stores whatever X, Y the node already has — it does not compute positions.

**Nodes loaded from file** retain their saved X, Y — no stagger is applied on load.

### ArchModel — add canvas state

```go
// internal/tui/views/architecture.go
type ArchModel struct {
    arch       *graph.Architecture  // shared reference, existing
    width      int                  // visible panel width in terminal cells (set via SetSize)
    height     int                  // visible panel height in terminal cells (set via SetSize)
    viewportX  int                  // top-left X of visible window into canvas
    viewportY  int                  // top-left Y of visible window into canvas
    selectedID string               // currently selected block ID; "" if canvas empty
    moveMode   bool                 // arrow keys reposition selected block
    moveOriginX, moveOriginY int    // saved position when move mode entered (for Esc restore)
    connectMode bool                // manual C-key connect in progress
    portMode    bool                // drag-to-connect in progress
    connectSourceID string          // source node ID for connectMode and portMode
    portTargetID    string          // currently hovered target in portMode; "" if none
    smartPlacementMode    bool     // waiting for parent selection after add
    smartPlacementOptions []string // node IDs of compatible parents shown in prompt
    smartPlacementIdx     int      // currently highlighted index in options list
    pendingNode           *graph.Node // node being placed, held until parent chosen
    // rename mode fields preserved from current implementation
}
```

`moveMode`, `connectMode`, `portMode` are mutually exclusive. Entering one clears the others and resets their associated fields. `portTargetID` is cleared to `""` when `portMode` goes false.

**Viewport dimensions** are derived from the terminal window: the root `app.go` calls `archV.SetSize(w, h)` when it receives `tea.WindowSizeMsg`. `SetSize` stores the architecture panel's allocated width and height in `ArchModel`.

### Changes summary

| Package | File | Change |
|---|---|---|
| `internal/graph` | `node.go` | Add `X, Y int` fields |
| `internal/graph` | `architecture.go` | `AddNode` stores X, Y from node as-is (no stagger logic) |
| `internal/graph` | `architecture_test.go` | Add test: X/Y round-trips through `Save`/`Load` correctly |
| `internal/renderer` | `ascii.go` | **Deleted** |
| `internal/renderer` | `ascii_test.go` | **Deleted** |
| `internal/tui/tuicore` | `messages.go` | Add `MoveNodeMsg{ID string, X, Y int}` and `StartSmartPlacementMsg{Node *graph.Node}`. Existing `ConnectNodesMsg`, `RenameNodeMsg`, `DeleteNodeMsg` unchanged. |
| `internal/tui` | `messages.go` | Add type aliases for `MoveNodeMsg` and `StartSmartPlacementMsg` (following the existing alias pattern for all other message types). |
| `internal/tui` | `app.go` | Handle `MoveNodeMsg` (set dirty); handle `AddNodeMsg`: if resource has non-empty `ParentRefAttr`, allocate node but emit `StartSmartPlacementMsg` to `ArchModel` instead of calling `arch.AddNode`. If `ParentRefAttr` empty, existing flow unchanged. Call `archV.SetSize` on `WindowSizeMsg`. |
| `internal/tui/views` | `architecture.go` | Full rewrite of render + input handling |

`internal/tui/layout.go` is unchanged — panel size calculations are already there and `app.go` derives the architecture panel dimensions from them before calling `SetSize`.

No other files change.

---

## Canvas

### Virtual canvas size

200 columns × 60 rows (terminal cells). Viewport auto-scrolls to keep the selected block in view with a margin of 4 cells on each side.

### Block dimensions

Each block is **16 columns wide × 4 rows tall** (including border). Contents:

```
┌──────────────┐     ← row 1: top border (14 dashes)
│ <type>       │     ← row 2: resource type label (leading space + 13 chars, truncated)
│ <name>       │     ← row 3: node name (leading space + 13 chars, truncated)
└──────────────┘     ← row 4: bottom border
```

Text is truncated to **13 characters** (the usable content width after the leading space). `│` + ` ` + 13 chars + `│` = 16 total.

### Block visual states

Block borders are rendered using Lip Gloss `border` styles. There are no pixel widths — "border" means the single-line box-drawing characters (`┌─┐│└─┘`). The selected/active states use different Lip Gloss foreground colors on the border characters.

| State | Border color | Background | Name suffix |
|---|---|---|---|
| Normal | `#30363d` (dim grey) | `#161b22` | none |
| Selected | `#58a6ff` (blue) | `#161b22` | none |
| Move mode | `#3fb950` (green) | `#1a2a1a` | ` ✥` |
| Port mode source | `#3fb950` (green) | `#1a2a1a` | ` ⇥` |
| Port hover target | `#f0883e` (orange) | `#161b22` | none |

No glow effects. Color difference alone distinguishes states.

### Connection rendering

Connections are rendered as a separate pass after all blocks are drawn. Lines are drawn directly into the string buffer at (x, y) positions; they overdraw blank canvas cells. Lines that pass through a block position are a known visual artifact — no special handling.

For each `Edge (from → to)`:

**Exit point selection** (based on relative position of `to` to `from`):
- `to` is to the right or right-and-below: exit from **right-center** of `from` (x = fromX + 16, y = fromY + 2)
- `to` is directly below: exit from **bottom-center** of `from` (x = fromX + 8, y = fromY + 4)
- `to` is to the left: exit from **left-center** of `from` (x = fromX, y = fromY + 2)
- `to` is above: exit from **top-center** of `from` (x = fromX + 8, y = fromY)

**Entry point** matches the exit direction:
- Exit right → entry **left-center** of `to` (x = toX, y = toY + 2)
- Exit bottom → entry **top-center** of `to` (x = toX + 8, y = toY)
- Exit left → entry **right-center** of `to` (x = toX + 16, y = toY + 2)
- Exit top → entry **bottom-center** of `to` (x = toX + 8, y = toY + 4)

**Routing by case:**
- **Right exit → left entry** (most common): horizontal segment from (exitX, exitY) to (entryX, exitY), then vertical from there to (entryX, entryY). Corner char at bend if rows differ; straight horizontal if same row.
- **Bottom exit → top entry**: vertical segment from (exitX, exitY) to (exitX, entryY), then horizontal from there to (entryX, entryY). Corner char at bend if columns differ; straight vertical if same column.
- **Left exit → right entry**: horizontal segment going left from exit to entryX, then vertical to entryY. Same L-shape logic as right case, direction reversed.
- **Top exit → bottom entry**: vertical segment going up from exit to entryY, then horizontal to entryX.

In all cases, the arrowhead character (`▶`, `▼`, `◀`, `▲`) is placed at the entry point, direction matching the final segment's direction.

**Characters used:** `─` (horizontal), `│` (vertical), `┐` (right-then-down corner), `└` (up-then-right), `┘` (left-then-up), `┌` (down-then-right), `▶` (rightward arrowhead), `▼` (downward arrowhead), `◀` (leftward arrowhead), `▲` (upward arrowhead).

**Color:** Connections where `from.ID == selectedID` or `to.ID == selectedID` render in `#58a6ff` (blue). All others render in `#30363d` (dim grey).

**Port mode preview:** While `portMode` is true, draw a dashed green line (`-`) from the source exit point to the entry point of `portTargetID`. If `portTargetID == ""`, no preview line is drawn.

### Viewport auto-scroll

After any `selectedID` change, move, or `SetSize` call: check if the selected block's bounding box `(node.X, node.Y, node.X+16, node.Y+4)` is within the safe zone `[viewportX+4, viewportY+4, viewportX+width-20, viewportY+height-8]`. If not, recompute:

```
viewportX = clamp(node.X + 8 - width/2,  0, max(0, 200 - width))
viewportY = clamp(node.Y + 2 - height/2, 0, max(0, 60  - height))
```

This centers the viewport on the block's center point `(node.X+8, node.Y+2)`. Integer division is fine — off-by-one in the viewport offset is invisible to the user.

---

## Interaction Model

### Zero-block and initial state

- Canvas empty: `selectedID = ""`. Arrow keys are no-ops. All connection/move/port keys are no-ops.
- First block added: it is auto-selected (`selectedID = newNode.ID`).
- On load: `selectedID` is set to `arch.NodeOrder[0]` if any nodes exist, `""` otherwise.

### Normal mode (canvas focused)

| Key | Action |
|---|---|
| `↑` `↓` `←` `→` | Cycle to nearest block in that direction (see algorithm below) |
| `M` | Enter move mode |
| `L` | Enter port mode (link) |
| `C` | Enter connect mode |
| `D` | Delete selected block + all its edges |
| `R` | Enter rename mode (textinput in status bar, pre-filled with current name) |
| `E` | Focus Properties panel |
| `Tab` | Cycle panel focus |

**Adjacent block selection algorithm:**
1. Define a directional half-plane: e.g., for `→`, all blocks whose center X > selectedBlock.centerX.
2. From those, filter to blocks within a 45° cone: `abs(dy) <= abs(dx)` for horizontal directions, `abs(dx) <= abs(dy)` for vertical.
3. Select the one with the smallest Euclidean distance from the selected block's center.
4. If no blocks exist in the 45° cone, widen to the full half-plane and pick the nearest.
5. If no blocks exist in the half-plane, no-op.

### Move mode

1. Press `M` → save `moveOriginX = node.X`, `moveOriginY = node.Y`; `moveMode = true`; block turns green + `✥`
2. Status bar: `"Moving <name> — arrows to reposition, Enter/M to drop, Esc to cancel"`
3. Arrow keys shift `node.X`/`node.Y` by 2 cells per press (always even-number positions)
4. `M` or `Enter` → emit `MoveNodeMsg{ID, X, Y}` to confirm; root `app.go` updates the node; `moveMode = false`
5. `Esc` → restore `node.X = moveOriginX`, `node.Y = moveOriginY`; emit `MoveNodeMsg` with original coords; `moveMode = false`

**Data flow during move mode:** `ArchModel` holds a shared reference to `*graph.Architecture`, so it mutates `arch.Nodes[id].X/Y` directly on each arrow press — providing live visual feedback at no extra cost. `MoveNodeMsg` is emitted only on confirm (`M`/`Enter`) or cancel (`Esc`) solely to signal `app.go` to set `m.dirty = true`. `app.go` does not re-apply the X/Y values from the message — it only sets the dirty flag.

### Port mode (drag-to-connect)

1. Press `L` → `portMode = true`, `connectSourceID = selectedID`, `portTargetID = ""`; source block turns green + `⇥`
2. Status bar: `"Link mode — navigate to target, Enter to connect, Esc to cancel"`
3. Arrow keys cycle `portTargetID` using the adjacent block algorithm. Initial pivot (when `portTargetID == ""`) is `connectSourceID`. Once `portTargetID` is set, pivot is `portTargetID`. The source block (`connectSourceID`) is excluded from candidates at all times. Hovered target highlights orange.
4. `Enter` (with `portTargetID != ""`) → emit `ConnectNodesMsg{From: connectSourceID, To: portTargetID}`; `portMode = false`; `selectedID = portTargetID`
5. `Enter` (with `portTargetID == ""`) → no-op
6. `Esc` → `portMode = false`, `portTargetID = ""`
7. Self-loop (`portTargetID == connectSourceID`): cannot occur (source excluded from cycling)
8. Duplicate edge: `ConnectNodesMsg` is handled by `arch.Connect()` which is a no-op on duplicates; status bar shows `"Connection already exists"` for 3 seconds (same transient message mechanism used across the app)

### Manual connect mode (C key)

1. Press `C` → `connectMode = true`, `connectSourceID = selectedID`; panel label changes to `ARCHITECTURE [CONNECT]`
2. Status bar: `"Select target to connect (Esc to cancel)"`
3. Arrow keys cycle `selectedID` normally (source node can be re-selected; self-loop rejection happens on Enter)
4. `Enter` → if `selectedID == connectSourceID`: status `"Cannot connect a resource to itself"`, mode stays active. Otherwise: emit `ConnectNodesMsg{From: connectSourceID, To: selectedID}`; `connectMode = false`
5. `Esc` → `connectMode = false`

The navigation mechanism changes from list-based to canvas-based, but the user-visible behavior (C → navigate → Enter) is identical. This is a rewrite of the internal mechanics, not the UX contract.

### Smart placement (on add)

Triggered when the user presses `A` or `Enter` in the Catalog panel for a resource whose `ResourceDef.ParentRefAttr` is non-empty.

**Compatible parent resolution table:**

| Resource being added | ParentRefAttr | Compatible parent TFType |
|---|---|---|
| Subnet | `vpc_id` | `aws_vpc` |
| EC2 Instance | `subnet_id` | `aws_subnet` |
| ECS Service | `subnet_id` | `aws_subnet` |
| RDS | `subnet_id` | `aws_subnet` |
| NAT Gateway | `subnet_id` | `aws_subnet` |
| Lambda | `subnet_id` | `aws_subnet` |
| Security Group | `vpc_id` | `aws_vpc` |

Resources with empty `ParentRefAttr` (VPC, IGW, S3, DynamoDB, ALB) skip smart placement entirely.

**Flow:**

1. `CatalogModel` emits `AddNodeMsg{Def}` as usual.
2. `app.go` receives `AddNodeMsg`. Since `Def.ParentRefAttr` is non-empty, it allocates a `*graph.Node` (generates ID, copies default props) but does NOT call `arch.AddNode`. Instead it emits `StartSmartPlacementMsg{Node: node}` which is dispatched to `ArchModel`.
3. `ArchModel` receives `StartSmartPlacementMsg`, stores the node as `pendingNode`. Look up compatible parent nodes from the current architecture. If none exist, skip prompt — jump to step 9 with `[none]` behaviour.
4. Focus shifts to the canvas. All blocks and connections dim (muted style). Compatible parent nodes remain full brightness. Set `smartPlacementMode = true`, `smartPlacementOptions = [compatibleIDs..., "none"]`, `smartPlacementIdx = 0`.
5. Status bar: `"Connect <DisplayName> to: <name1>  <name2>  [none]  (↑↓ select, Enter confirm)"`
6. `↑`/`↓` cycle `smartPlacementIdx`. `←`/`→` are no-ops. `Enter` confirms the highlighted option. `Esc` acts as `[none]`.
7. **If a parent is chosen:**
   - Count existing children: `childCount = number of edges where edge.From == chosenParentID`
   - Place `pendingNode` at `(parentX + 20 * childCount, parentY + 6)`
   - Call `arch.AddNode(pendingNode)` then `arch.Connect(chosenParentID, pendingNode.ID)`
8. **If `[none]`:**
   - Stagger index = `len(arch.Nodes)` at this moment (pendingNode is not yet in arch.Nodes)
   - Compute stagger position, set `pendingNode.X, pendingNode.Y`
   - Call `arch.AddNode(pendingNode)` with no edge
9. Set `selectedID = pendingNode.ID`; `pendingNode = nil`; `smartPlacementMode = false`; canvas un-dims; emit `MoveNodeMsg` with the node's final X/Y to `app.go` (so dirty flag is set via the existing MoveNodeMsg handler).

### Rename mode

Press `R` in normal mode: the existing rename textinput is reused from the current implementation. It renders **in the header status bar area** (the top bar of the TUI, which already shows transient status messages). The textinput replaces the status message slot. The canvas remains visible behind it and the selected block's name updates immediately on `Enter`. `Esc` cancels. Emits `RenameNodeMsg`.

### Delete

`D` deletes the selected block. All edges where `edge.From == selectedID` or `edge.To == selectedID` are automatically removed. No confirmation prompt. `selectedID` moves to the next block in `NodeOrder` (or `""` if canvas becomes empty). Emits `DeleteNodeMsg`.

---

## Testing Strategy

- **Unit tests — graph:** `Save`/`Load` round-trips X/Y correctly (X, Y on Node serialize to JSON and are restored on load).
- **Unit tests — renderer:** `internal/renderer` package is deleted entirely (`ascii.go` + `ascii_test.go`). No other package imports `internal/renderer` so no cascading test breakage.
- **No unit tests for canvas render/input:** TUI canvas logic is tightly coupled to terminal state and Lip Gloss rendering; covered by manual smoke test instead.
- **Manual smoke test:** Add VPC → Subnet → ALB → ECS → RDS; connect via all three methods; move blocks; verify connections re-route; save and reload; verify positions persist.
