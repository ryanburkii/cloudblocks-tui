# Canvas Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the ASCII tree view in the Architecture panel with a free-form 2D block diagram canvas where resource blocks are positioned with arrow keys and connected via three interaction modes.

**Architecture:** The existing `ArchModel` in `internal/tui/views/architecture.go` is fully rewritten. It maintains a virtual 200×60 canvas; each Node has (X, Y) position; a viewport clips the visible area. Block rendering uses a cell grid (rune + lipgloss color per cell). Connection lines are drawn as orthogonal routes into the cell grid. All interaction modes (normal, move, connect, link, smart-placement) are state-machine branches in `Update`.

**Tech Stack:** Go 1.26.1, Bubble Tea, Lip Gloss, Bubbles (textinput). No new dependencies.

---

## Chunk 1: Foundational changes

### Task 1: Add X, Y position fields to Node

**Files:**
- Modify: `internal/graph/node.go`
- Modify: `internal/graph/persistence_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/graph/persistence_test.go`:

```go
func TestNodeXY_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "arch.json")

	arch := graph.New()
	arch.AddNode(&graph.Node{
		ID: "vpc-1", Type: "aws_vpc", Name: "v",
		Properties: map[string]interface{}{},
		X: 42, Y: 18,
	})

	if err := arch.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}
	arch2 := graph.New()
	if err := arch2.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	n := arch2.Nodes["vpc-1"]
	if n.X != 42 || n.Y != 18 {
		t.Errorf("expected X=42 Y=18, got X=%d Y=%d", n.X, n.Y)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
cd /home/burkii/cloudblocks-tui
go test ./internal/graph/... -run TestNodeXY_RoundTrip -v
```

Expected: compile error (`X` undefined on `Node`) or FAIL.

- [ ] **Step 3: Add X, Y to Node**

Replace `internal/graph/node.go`:

```go
// internal/graph/node.go
package graph

// Node represents a single AWS resource in the architecture.
type Node struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	X          int                    `json:"x"`
	Y          int                    `json:"y"`
}
```

- [ ] **Step 4: Run the test to verify it passes**

```bash
go test ./internal/graph/... -v
```

Expected: all tests PASS (existing tests still pass; new test passes).

- [ ] **Step 5: Commit**

```bash
git add internal/graph/node.go internal/graph/persistence_test.go
git commit -m "feat: add X, Y position fields to Node"
```

---

### Task 2: Delete renderer package and write architecture.go skeleton

This task deletes the renderer package (which the old architecture.go imported) and replaces architecture.go with a skeleton that compiles. The skeleton has the correct struct and all method signatures but renders an empty canvas placeholder. Subsequent tasks fill in the rendering and interaction logic.

**Files:**
- Delete: `internal/renderer/ascii.go`
- Delete: `internal/renderer/ascii_test.go`
- Rewrite: `internal/tui/views/architecture.go`

- [ ] **Step 1: Delete the renderer package**

```bash
rm internal/renderer/ascii.go internal/renderer/ascii_test.go
rmdir internal/renderer 2>/dev/null || true
```

- [ ] **Step 2: Write the architecture.go skeleton**

Replace `internal/tui/views/architecture.go` entirely:

```go
// internal/tui/views/architecture.go
package views

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/graph"
	"cloudblocks-tui/internal/tui/tuicore"
)

const (
	blockW  = 16
	blockH  = 4
	canvasW = 200
	canvasH = 60
)

// canvasCell is one terminal cell in the canvas grid.
type canvasCell struct {
	ch rune
	fg lipgloss.Color
	bg lipgloss.Color
}

// canvas color palette
var (
	colNormalBorder = lipgloss.Color("#30363d")
	colSelected     = lipgloss.Color("#58a6ff")
	colMove         = lipgloss.Color("#3fb950")
	colPortTarget   = lipgloss.Color("#f0883e")
	colNormalBg     = lipgloss.Color("#161b22")
	colMoveBg       = lipgloss.Color("#1a2a1a")
	colDimBg        = lipgloss.Color("#0d1117")
	colDimFg        = lipgloss.Color("#21262d")
	colType         = lipgloss.Color("#f0883e")
	colName         = lipgloss.Color("#e6edf3")
	colConnActive   = lipgloss.Color("#58a6ff")
	colConnDim      = lipgloss.Color("#30363d")
	colConnLink     = lipgloss.Color("#3fb950")
)

// ArchModel is the architecture panel sub-model.
type ArchModel struct {
	arch      *graph.Architecture
	width     int
	height    int
	viewportX int
	viewportY int
	selectedID string

	moveMode              bool
	moveOriginX, moveOriginY int

	connectMode     bool
	portMode        bool
	connectSourceID string
	portTargetID    string

	smartPlacementMode    bool
	smartPlacementOptions []string // compatible node IDs + "none" sentinel at end
	smartPlacementIdx     int
	pendingNode           *graph.Node

	renameMode  bool
	renameInput textinput.Model
}

// NewArch creates a new ArchModel backed by arch.
func NewArch(arch *graph.Architecture) ArchModel {
	ti := textinput.New()
	ti.Placeholder = "new name"
	m := ArchModel{arch: arch, renameInput: ti}
	if len(arch.NodeOrder) > 0 {
		m.selectedID = arch.NodeOrder[0]
		m.scrollToSelected()
	}
	return m
}

// Refresh updates the model after the architecture changed externally.
func (m ArchModel) Refresh(arch *graph.Architecture) ArchModel {
	m.arch = arch
	if m.selectedID != "" {
		if _, ok := arch.Nodes[m.selectedID]; !ok {
			m.selectedID = ""
		}
	}
	if m.selectedID == "" && len(arch.NodeOrder) > 0 {
		m.selectedID = arch.NodeOrder[0]
	}
	m.scrollToSelected()
	return m
}

// SetSize stores panel dimensions and re-scrolls to selected block.
func (m ArchModel) SetSize(w, h int) ArchModel {
	m.width = w
	m.height = h
	m.scrollToSelected()
	return m
}

// Width returns the panel width (used by app.go for deploy panel sizing).
func (m ArchModel) Width() int { return m.width }

// InConnectMode reports whether manual connect mode is active.
func (m ArchModel) InConnectMode() bool { return m.connectMode }

// InLinkMode reports whether link mode is active.
func (m ArchModel) InLinkMode() bool { return m.portMode }

// InSmartPlacementMode reports whether smart placement prompt is active.
func (m ArchModel) InSmartPlacementMode() bool { return m.smartPlacementMode }

// StaggerPosition computes the default canvas (X, Y) for new node index n (0-based).
// n = len(arch.Nodes) before the new node is added.
func StaggerPosition(n int) (int, int) {
	col := n / 9
	row := n % 9
	return 2 + col*20, 2 + row*6
}

// scrollToSelected adjusts viewportX/Y so the selected block is visible.
func (m *ArchModel) scrollToSelected() {
	if m.selectedID == "" || m.width == 0 || m.height == 0 {
		return
	}
	n, ok := m.arch.Nodes[m.selectedID]
	if !ok {
		return
	}
	// Check safe zone
	if n.X >= m.viewportX+4 && n.X+blockW <= m.viewportX+m.width-4 &&
		n.Y >= m.viewportY+4 && n.Y+blockH <= m.viewportY+m.height-4 {
		return
	}
	// Center on block's center point
	vx := n.X + 8 - m.width/2
	vy := n.Y + 2 - m.height/2
	maxVX := canvasW - m.width
	maxVY := canvasH - m.height
	if maxVX < 0 {
		maxVX = 0
	}
	if maxVY < 0 {
		maxVY = 0
	}
	m.viewportX = clampInt(vx, 0, maxVX)
	m.viewportY = clampInt(vy, 0, maxVY)
}

// clampInt clamps v to [lo, hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// truncate truncates s to at most n runes, padding with spaces to exactly n.
func truncatePad(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		runes = runes[:n]
	}
	for len(runes) < n {
		runes = append(runes, ' ')
	}
	return string(runes)
}

// runeWidth returns the number of runes in s (same as len([]rune(s))).
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

// Update processes input events.
func (m ArchModel) Update(msg tea.Msg) (ArchModel, tea.Cmd) {
	// Non-key messages
	if spm, ok := msg.(tuicore.StartSmartPlacementMsg); ok {
		return m.handleStartSmartPlacement(spm)
	}

	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		return m, nil
	}

	if m.renameMode {
		return m.handleRenameKey(keyMsg)
	}
	if m.smartPlacementMode {
		return m.handleSmartPlacementKey(keyMsg)
	}
	if m.moveMode {
		return m.handleMoveKey(keyMsg)
	}
	if m.portMode {
		return m.handleLinkKey(keyMsg)
	}
	if m.connectMode {
		return m.handleConnectKey(keyMsg)
	}
	return m.handleNormalKey(keyMsg)
}

// placeholder stubs — filled in by later tasks
func (m ArchModel) handleRenameKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := strings.TrimSpace(m.renameInput.Value())
		if newName == "" {
			if n, ok := m.arch.Nodes[m.selectedID]; ok {
				newName = n.Name
			}
		}
		m.renameMode = false
		nodeID := m.selectedID
		return m, func() tea.Msg {
			return tuicore.RenameNodeMsg{NodeID: nodeID, Name: newName}
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

func (m ArchModel) handleSmartPlacementKey(msg tea.KeyMsg) (ArchModel, tea.Cmd)  { return m, nil }
func (m ArchModel) handleMoveKey(msg tea.KeyMsg) (ArchModel, tea.Cmd)             { return m, nil }
func (m ArchModel) handleLinkKey(msg tea.KeyMsg) (ArchModel, tea.Cmd)             { return m, nil }
func (m ArchModel) handleConnectKey(msg tea.KeyMsg) (ArchModel, tea.Cmd)          { return m, nil }
func (m ArchModel) handleNormalKey(msg tea.KeyMsg) (ArchModel, tea.Cmd)           { return m, nil }
func (m ArchModel) handleStartSmartPlacement(msg tuicore.StartSmartPlacementMsg) (ArchModel, tea.Cmd) {
	return m, nil
}

// View renders the architecture panel.
func (m ArchModel) View() string {
	if m.renameMode {
		if n, ok := m.arch.Nodes[m.selectedID]; ok {
			return "Rename " + n.Name + ": " + m.renameInput.View()
		}
	}
	if len(m.arch.Nodes) == 0 {
		return mutedStyle.Render("(empty — press A in Catalog to add resources)")
	}
	return mutedStyle.Render("(canvas — coming in next task)")
}

// emitSelect emits SelectNodeMsg for the currently selected node.
func (m ArchModel) emitSelect() tea.Cmd {
	if m.selectedID == "" {
		return nil
	}
	id := m.selectedID
	return func() tea.Msg { return tuicore.SelectNodeMsg{NodeID: id} }
}

// emitSelectFocus is like emitSelect but sets FocusProps: true (E key).
func (m ArchModel) emitSelectFocus() (ArchModel, tea.Cmd) {
	if m.selectedID == "" {
		return m, nil
	}
	id := m.selectedID
	return m, func() tea.Msg { return tuicore.SelectNodeMsg{NodeID: id, FocusProps: true} }
}

// Ensure unused import is referenced (will be used in Task 5).
var _ = runeLen
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: success (no errors). Note: the existing `InConnectMode()` method is preserved so `app.go` still compiles.

- [ ] **Step 4: Commit**

```bash
git add internal/renderer internal/tui/views/architecture.go
git commit -m "feat: delete renderer, add canvas ArchModel skeleton"
```

---

### Task 3: New message types, key bindings, and type aliases

**Files:**
- Modify: `internal/tui/tuicore/messages.go`
- Modify: `internal/tui/tuicore/keymap.go`
- Modify: `internal/tui/messages.go`

- [ ] **Step 1: Add MoveNodeMsg and StartSmartPlacementMsg**

Append to `internal/tui/tuicore/messages.go`:

```go
// MoveNodeMsg is emitted by ArchModel when a block's position changes.
// app.go handles it by setting the dirty flag.
type MoveNodeMsg struct {
	ID   string
	X, Y int
}

// StartSmartPlacementMsg is emitted by app.go when a resource with a
// ParentRefAttr is added. ArchModel holds the pending node and shows
// the parent-selection prompt.
type StartSmartPlacementMsg struct {
	Node *graph.Node
}
```

Add `"cloudblocks-tui/internal/graph"` to the import in `tuicore/messages.go`.

Full updated `internal/tui/tuicore/messages.go`:

```go
// internal/tui/tuicore/messages.go
package tuicore

import (
	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/graph"
)

// AddNodeMsg is emitted by CatalogModel when the user adds a resource.
type AddNodeMsg struct{ Def *resources.ResourceDef }

// SelectNodeMsg is emitted by ArchModel when the user moves the cursor.
type SelectNodeMsg struct {
	NodeID     string
	FocusProps bool
}

// ConnectNodesMsg is emitted by ArchModel when connect/link mode completes.
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

// MoveNodeMsg is emitted by ArchModel when a block's canvas position changes.
// app.go handles it solely by setting the dirty flag.
type MoveNodeMsg struct {
	ID   string
	X, Y int
}

// StartSmartPlacementMsg is emitted by app.go when a resource with a
// non-empty ParentRefAttr is added. ArchModel holds the pending node and
// shows the parent-selection prompt before calling arch.AddNode.
type StartSmartPlacementMsg struct {
	Node *graph.Node
}
```

- [ ] **Step 2: Add Left, Right, Move, Link bindings to KeyMap**

Replace `internal/tui/tuicore/keymap.go`:

```go
// internal/tui/tuicore/keymap.go
package tuicore

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Tab     key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Add     key.Binding
	Connect key.Binding
	Move    key.Binding
	Link    key.Binding
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
		Left:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		Right:   key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Connect: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connect")),
		Move:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move block")),
		Link:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link")),
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

- [ ] **Step 3: Add type aliases in tui/messages.go**

Replace `internal/tui/messages.go`:

```go
// internal/tui/messages.go
package tui

import "cloudblocks-tui/internal/tui/tuicore"

// Re-export message types from tuicore so callers can use either import path.
type AddNodeMsg = tuicore.AddNodeMsg
type SelectNodeMsg = tuicore.SelectNodeMsg
type ConnectNodesMsg = tuicore.ConnectNodesMsg
type DeleteNodeMsg = tuicore.DeleteNodeMsg
type RenameNodeMsg = tuicore.RenameNodeMsg
type UpdatePropMsg = tuicore.UpdatePropMsg
type DeployLineMsg = tuicore.DeployLineMsg
type DeployDoneMsg = tuicore.DeployDoneMsg
type StatusMsg = tuicore.StatusMsg
type MoveNodeMsg = tuicore.MoveNodeMsg
type StartSmartPlacementMsg = tuicore.StartSmartPlacementMsg
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tuicore/messages.go internal/tui/tuicore/keymap.go internal/tui/messages.go
git commit -m "feat: add MoveNodeMsg, StartSmartPlacementMsg, Left/Right/Move/Link key bindings"
```

---

## Chunk 2: Canvas rendering

### Task 4: Canvas block rendering

Add the cell grid, `renderCanvas`, and `drawBlock` to `architecture.go`. After this task, running the app will show actual blocks on the canvas.

**Files:**
- Modify: `internal/tui/views/architecture.go`

- [ ] **Step 1: Add makeGrid, setCellFg, renderGrid helpers and drawBlock**

Add these functions to `internal/tui/views/architecture.go` (after the existing helper functions, before `View`):

```go
// makeGrid creates a width×height cell grid filled with spaces.
func makeGrid(w, h int) [][]canvasCell {
	grid := make([][]canvasCell, h)
	for y := range grid {
		grid[y] = make([]canvasCell, w)
		for x := range grid[y] {
			grid[y][x] = canvasCell{ch: ' ', fg: colNormalBorder, bg: ""}
		}
	}
	return grid
}

// setCell writes a rune+colors to grid[y][x] if in bounds.
func setCell(grid [][]canvasCell, x, y int, ch rune, fg, bg lipgloss.Color) {
	if y < 0 || y >= len(grid) || x < 0 || x >= len(grid[0]) {
		return
	}
	grid[y][x] = canvasCell{ch: ch, fg: fg, bg: bg}
}

// renderGrid converts the cell grid to a displayable string.
func renderGrid(grid [][]canvasCell) string {
	var sb strings.Builder
	for y, row := range grid {
		for _, c := range row {
			style := lipgloss.NewStyle().Foreground(c.fg)
			if c.bg != "" {
				style = style.Background(c.bg)
			}
			sb.WriteString(style.Render(string(c.ch)))
		}
		if y < len(grid)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// blockBorderColor returns the border color for a given node based on current mode.
func (m ArchModel) blockBorderColor(nodeID string) lipgloss.Color {
	switch {
	case nodeID == m.selectedID && m.moveMode:
		return colMove
	case nodeID == m.selectedID && m.portMode:
		return colMove
	case nodeID == m.selectedID:
		return colSelected
	case nodeID == m.portTargetID && m.portMode:
		return colPortTarget
	default:
		return colNormalBorder
	}
}

// blockBg returns the background color for a given node.
func (m ArchModel) blockBg(nodeID string) lipgloss.Color {
	if (nodeID == m.selectedID) && (m.moveMode || m.portMode) {
		return colMoveBg
	}
	if m.smartPlacementMode {
		for _, opt := range m.smartPlacementOptions {
			if opt == nodeID {
				return colNormalBg // compatible: full brightness
			}
		}
		return colDimBg // incompatible: dim
	}
	return colNormalBg
}

// drawBlock renders a node's block into the grid.
func (m ArchModel) drawBlock(grid [][]canvasCell, n *graph.Node) {
	// Translate canvas coords to viewport coords
	bx := n.X - m.viewportX
	by := n.Y - m.viewportY

	// Skip if completely off-screen
	if bx+blockW < 0 || bx >= m.width || by+blockH < 0 || by >= m.height {
		return
	}

	borderCol := m.blockBorderColor(n.ID)
	bg := m.blockBg(n.ID)

	nameSuffix := ""
	if n.ID == m.selectedID && m.moveMode {
		nameSuffix = " \u2725" // ✥
	} else if n.ID == m.selectedID && m.portMode {
		nameSuffix = " \u21e5" // ⇥
	}

	// Dim fg for incompatible nodes during smart placement
	fgType := colType
	fgName := colName
	if m.smartPlacementMode {
		isCompatible := false
		for _, opt := range m.smartPlacementOptions {
			if opt == n.ID {
				isCompatible = true
				break
			}
		}
		if !isCompatible {
			fgType = colDimFg
			fgName = colDimFg
		}
	}

	typeStr := truncatePad(n.Type, 13)
	nameStr := truncatePad(n.Name+nameSuffix, 13)

	// Row 0: top border  ┌──────────────┐
	// Row 1: type row    │ <type>       │
	// Row 2: name row    │ <name>       │
	// Row 3: bot border  └──────────────┘
	borderChars := []rune("┌" + strings.Repeat("─", 14) + "┐")
	botChars := []rune("└" + strings.Repeat("─", 14) + "┘")
	typeChars := []rune("│ " + typeStr + "│")
	nameChars := []rune("│ " + nameStr + "│")

	rows := [4][]rune{borderChars, typeChars, nameChars, botChars}
	rowFG := [4]lipgloss.Color{borderCol, fgType, fgName, borderCol}
	isBorder := [4]bool{true, false, false, true}

	for dy, row := range rows {
		y := by + dy
		for dx, ch := range row {
			x := bx + dx
			fg := rowFG[dy]
			// Side border chars on content rows
			if !isBorder[dy] && (dx == 0 || dx == blockW-1) {
				fg = borderCol
			}
			setCell(grid, x, y, ch, fg, bg)
		}
	}
}
```

- [ ] **Step 2: Update View() to call renderCanvas**

Replace the `View` method:

```go
func (m ArchModel) View() string {
	if m.renameMode {
		if n, ok := m.arch.Nodes[m.selectedID]; ok {
			return "Rename " + n.Name + ": " + m.renameInput.View()
		}
	}
	if len(m.arch.Nodes) == 0 {
		return mutedStyle.Render("(empty — press A in Catalog to add resources)")
	}
	return m.renderCanvas()
}

// renderCanvas builds the full canvas view string.
func (m ArchModel) renderCanvas() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	grid := makeGrid(m.width, m.height)

	// Draw blocks (connections drawn in Task 5)
	for _, id := range m.arch.NodeOrder {
		if n, ok := m.arch.Nodes[id]; ok {
			m.drawBlock(grid, n)
		}
	}

	return renderGrid(grid)
}
```

- [ ] **Step 3: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 4: Verify blocks render**

```bash
./cloudblocks
```

Add a VPC via Catalog. The Architecture panel should show a bordered block (16×4 chars) with `aws_vpc` and `VPC` inside it.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/views/architecture.go
git commit -m "feat: canvas block rendering with cell grid"
```

---

### Task 5: Connection routing

Add `drawConnections` to `renderCanvas`. After this task, edges between blocks render as orthogonal lines with arrowheads.

**Files:**
- Modify: `internal/tui/views/architecture.go`

- [ ] **Step 1: Add exit/entry point helpers**

Add to `internal/tui/views/architecture.go`:

```go
// exitDir returns which side the connection exits from, based on relative position of to vs from.
// Returns "right", "bottom", "left", or "top".
func exitDir(from, to *graph.Node) string {
	// to is to the left of from
	if to.X+blockW <= from.X {
		return "left"
	}
	// to is above from
	if to.Y+blockH <= from.Y {
		return "top"
	}
	// to is directly below (center X close)
	if to.Y >= from.Y+blockH && abs(to.X+8-(from.X+8)) < blockW {
		return "bottom"
	}
	return "right"
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// exitPoint returns the canvas coordinates of the connection exit point.
func exitPoint(n *graph.Node, dir string) (int, int) {
	switch dir {
	case "right":
		return n.X + blockW, n.Y + 2
	case "bottom":
		return n.X + 8, n.Y + blockH
	case "left":
		return n.X, n.Y + 2
	case "top":
		return n.X + 8, n.Y
	}
	return n.X + blockW, n.Y + 2
}

// entryPoint returns the canvas coordinates of the connection entry point.
func entryPoint(n *graph.Node, dir string) (int, int) {
	switch dir {
	case "right":
		return n.X, n.Y + 2
	case "bottom":
		return n.X + 8, n.Y
	case "left":
		return n.X + blockW, n.Y + 2
	case "top":
		return n.X + 8, n.Y + blockH
	}
	return n.X, n.Y + 2
}

// arrowChar returns the arrowhead character for the final segment direction.
func arrowChar(dir string) rune {
	switch dir {
	case "right":
		return '▶'
	case "bottom":
		return '▼'
	case "left":
		return '◀'
	case "top":
		return '▲'
	}
	return '▶'
}

// drawHLine draws a horizontal line segment from (x1,y) to (x2,y).
// Coordinates are viewport-relative.
func drawHLine(grid [][]canvasCell, x1, x2, y int, fg lipgloss.Color) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		setCell(grid, x, y, '─', fg, "")
	}
}

// drawVLine draws a vertical line segment from (x,y1) to (x,y2).
// Coordinates are viewport-relative.
func drawVLine(grid [][]canvasCell, x, y1, y2 int, fg lipgloss.Color) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		setCell(grid, x, y, '│', fg, "")
	}
}
```

- [ ] **Step 2: Add drawEdge and drawConnections**

```go
// drawEdge draws a single orthogonal connection from node `from` to node `to`.
func (m ArchModel) drawEdge(grid [][]canvasCell, from, to *graph.Node, fg lipgloss.Color) {
	dir := exitDir(from, to)
	ex, ey := exitPoint(from, dir)
	nx, ny := entryPoint(to, dir)

	// Translate canvas → viewport
	ex -= m.viewportX
	ey -= m.viewportY
	nx -= m.viewportX
	ny -= m.viewportY

	arrow := arrowChar(dir)

	switch dir {
	case "right", "left":
		// Horizontal-first L-shape
		if ey == ny {
			// Straight horizontal
			drawHLine(grid, ex, nx, ey, fg)
		} else {
			// L-shape: horizontal to nx, then vertical to ny
			drawHLine(grid, ex, nx, ey, fg)
			drawVLine(grid, nx, ey, ny, fg)
			// Corner character at bend
			if ny > ey {
				setCell(grid, nx, ey, '┐', fg, "")
			} else {
				setCell(grid, nx, ey, '┘', fg, "")
			}
		}
	case "bottom":
		// Vertical-first L-shape going downward
		if ex == nx {
			drawVLine(grid, ex, ey, ny, fg)
		} else {
			drawVLine(grid, ex, ey, ny, fg)
			drawHLine(grid, ex, nx, ny, fg)
			// Corner: came down, turning right=┌ or left=┐
			if nx > ex {
				setCell(grid, ex, ny, '┌', fg, "")
			} else {
				setCell(grid, ex, ny, '┐', fg, "")
			}
		}
	case "top":
		// Vertical-first L-shape going upward
		if ex == nx {
			drawVLine(grid, ex, ey, ny, fg)
		} else {
			drawVLine(grid, ex, ey, ny, fg)
			drawHLine(grid, ex, nx, ny, fg)
			// Corner: came up, turning right=└ or left=┘
			if nx > ex {
				setCell(grid, ex, ny, '└', fg, "")
			} else {
				setCell(grid, ex, ny, '┘', fg, "")
			}
		}
	}
	// Draw arrowhead at entry point
	setCell(grid, nx, ny, arrow, fg, "")
}

// drawConnections draws all edges into the grid.
func (m ArchModel) drawConnections(grid [][]canvasCell) {
	for _, edge := range m.arch.Edges {
		from, fromOK := m.arch.Nodes[edge.From]
		to, toOK := m.arch.Nodes[edge.To]
		if !fromOK || !toOK {
			continue
		}
		fg := colConnDim
		if edge.From == m.selectedID || edge.To == m.selectedID {
			fg = colConnActive
		}
		m.drawEdge(grid, from, to, fg)
	}

	// Link mode preview
	if m.portMode && m.portTargetID != "" {
		src, srcOK := m.arch.Nodes[m.connectSourceID]
		tgt, tgtOK := m.arch.Nodes[m.portTargetID]
		if srcOK && tgtOK {
			m.drawEdgeDashed(grid, src, tgt, colConnLink)
		}
	}
}

// drawEdgeDashed draws a dashed preview line (for link mode).
func (m ArchModel) drawEdgeDashed(grid [][]canvasCell, from, to *graph.Node, fg lipgloss.Color) {
	dir := exitDir(from, to)
	ex, ey := exitPoint(from, dir)
	nx, ny := entryPoint(to, dir)
	ex -= m.viewportX
	ey -= m.viewportY
	nx -= m.viewportX
	ny -= m.viewportY

	// Dashed: draw '-' every other cell
	drawDashedH := func(x1, x2, y int) {
		if x1 > x2 {
			x1, x2 = x2, x1
		}
		for x := x1; x <= x2; x++ {
			if (x-x1)%2 == 0 {
				setCell(grid, x, y, '-', fg, "")
			}
		}
	}
	drawDashedV := func(x, y1, y2 int) {
		if y1 > y2 {
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			if (y-y1)%2 == 0 {
				setCell(grid, x, y, '|', fg, "")
			}
		}
	}

	switch dir {
	case "right", "left":
		if ey == ny {
			drawDashedH(ex, nx, ey)
		} else {
			drawDashedH(ex, nx, ey)
			drawDashedV(nx, ey, ny)
		}
	case "bottom", "top":
		if ex == nx {
			drawDashedV(ex, ey, ny)
		} else {
			drawDashedV(ex, ey, ny)
			drawDashedH(ex, nx, ny)
		}
	}
}
```

- [ ] **Step 3: Wire drawConnections into renderCanvas**

Update `renderCanvas` to call `drawConnections` before blocks:

```go
func (m ArchModel) renderCanvas() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	grid := makeGrid(m.width, m.height)

	// Draw connections first (blocks overdraw connection lines at overlap points)
	m.drawConnections(grid)

	// Draw blocks
	for _, id := range m.arch.NodeOrder {
		if n, ok := m.arch.Nodes[id]; ok {
			m.drawBlock(grid, n)
		}
	}

	return renderGrid(grid)
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/views/architecture.go
git commit -m "feat: orthogonal connection routing in canvas"
```

---

## Chunk 3: Interaction model and app.go

### Task 6: Normal mode navigation (adjacent block selection)

**Files:**
- Modify: `internal/tui/views/architecture.go`

- [ ] **Step 1: Add adjacentBlock helper**

```go
// blockCenter returns the center point of a node's block.
func blockCenter(n *graph.Node) (float64, float64) {
	return float64(n.X) + float64(blockW)/2, float64(n.Y) + float64(blockH)/2
}

// adjacentBlock finds the nearest block in the given direction from selectedID.
// dir: "up", "down", "left", "right"
// Returns "" if no block found in that direction.
func (m ArchModel) adjacentBlock(dir string) string {
	src, ok := m.arch.Nodes[m.selectedID]
	if !ok {
		return ""
	}
	sx, sy := blockCenter(src)

	bestID := ""
	bestDist := math.MaxFloat64

	for _, id := range m.arch.NodeOrder {
		if id == m.selectedID {
			continue
		}
		n, ok := m.arch.Nodes[id]
		if !ok {
			continue
		}
		cx, cy := blockCenter(n)
		dx := cx - sx
		dy := cy - sy

		// Check half-plane
		inPlane := false
		switch dir {
		case "right":
			inPlane = dx > 0
		case "left":
			inPlane = dx < 0
		case "down":
			inPlane = dy > 0
		case "up":
			inPlane = dy < 0
		}
		if !inPlane {
			continue
		}

		// 45° cone filter: for horizontal dirs abs(dy)<=abs(dx), vertical abs(dx)<=abs(dy)
		inCone := false
		switch dir {
		case "right", "left":
			inCone = math.Abs(dy) <= math.Abs(dx)
		case "up", "down":
			inCone = math.Abs(dx) <= math.Abs(dy)
		}

		dist := math.Sqrt(dx*dx + dy*dy)
		if inCone && dist < bestDist {
			bestDist = dist
			bestID = id
		}
	}

	// If nothing in cone, widen to full half-plane
	if bestID == "" {
		for _, id := range m.arch.NodeOrder {
			if id == m.selectedID {
				continue
			}
			n, ok := m.arch.Nodes[id]
			if !ok {
				continue
			}
			cx, cy := blockCenter(n)
			dx := cx - sx
			dy := cy - sy
			inPlane := false
			switch dir {
			case "right":
				inPlane = dx > 0
			case "left":
				inPlane = dx < 0
			case "down":
				inPlane = dy > 0
			case "up":
				inPlane = dy < 0
			}
			if !inPlane {
				continue
			}
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist < bestDist {
				bestDist = dist
				bestID = id
			}
		}
	}
	return bestID
}
```

- [ ] **Step 2: Add `math` to the import block in architecture.go**

The `adjacentBlock` function uses `math.MaxFloat64` and `math.Abs`. Add `"math"` to the import block:

```go
import (
    "math"
    "strings"
    "unicode/utf8"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "cloudblocks-tui/internal/graph"
    "cloudblocks-tui/internal/tui/tuicore"
)
```

(Note: `key` import is also added here; it's needed for `handleNormalKey` in Step 3.)

```bash
go build ./...
```

Expected: success.

- [ ] **Step 3: Implement handleNormalKey**

Replace the stub `handleNormalKey`:

```go
func (m ArchModel) handleNormalKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()

	switch {
	case key.Matches(msg, km.Up):
		if id := m.adjacentBlock("up"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Down):
		if id := m.adjacentBlock("down"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Left):
		if id := m.adjacentBlock("left"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Right):
		if id := m.adjacentBlock("right"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Move):
		if m.selectedID != "" {
			return m.enterMoveMode()
		}
	case key.Matches(msg, km.Link):
		if m.selectedID != "" {
			return m.enterLinkMode()
		}
	case key.Matches(msg, km.Connect):
		if m.selectedID != "" {
			return m.enterConnectMode()
		}
	case key.Matches(msg, km.Delete):
		if m.selectedID != "" {
			return m.deleteSelected()
		}
	case key.Matches(msg, km.Rename):
		if m.selectedID != "" {
			return m.enterRenameMode()
		}
	case key.Matches(msg, km.Edit):
		if m.selectedID != "" {
			return m.emitSelectFocus()
		}
	}
	return m, nil
}
```

- [ ] **Step 4: Add mode entry stubs**

```go
func (m ArchModel) enterMoveMode() (ArchModel, tea.Cmd) {
	if n, ok := m.arch.Nodes[m.selectedID]; ok {
		m.moveOriginX, m.moveOriginY = n.X, n.Y
	}
	m.moveMode = true
	m.connectMode = false
	m.portMode = false
	m.smartPlacementMode = false
	return m, func() tea.Msg {
		n := m.arch.Nodes[m.selectedID]
		return tuicore.StatusMsg{Text: "Moving " + n.Name + " — arrows to reposition, Enter/M to drop, Esc to cancel"}
	}
}

func (m ArchModel) enterLinkMode() (ArchModel, tea.Cmd) {
	m.portMode = true
	m.connectMode = false
	m.moveMode = false
	m.smartPlacementMode = false
	m.connectSourceID = m.selectedID
	m.portTargetID = ""
	return m, func() tea.Msg {
		return tuicore.StatusMsg{Text: "Link mode — navigate to target, Enter to connect, Esc to cancel"}
	}
}

func (m ArchModel) enterConnectMode() (ArchModel, tea.Cmd) {
	m.connectMode = true
	m.portMode = false
	m.moveMode = false
	m.smartPlacementMode = false
	m.connectSourceID = m.selectedID
	return m, func() tea.Msg {
		return tuicore.StatusMsg{Text: "Select target to connect (Esc to cancel)"}
	}
}

func (m ArchModel) enterRenameMode() (ArchModel, tea.Cmd) {
	if n, ok := m.arch.Nodes[m.selectedID]; ok {
		m.renameMode = true
		m.renameInput.SetValue(n.Name)
		m.renameInput.Focus()
	}
	return m, nil
}

func (m ArchModel) deleteSelected() (ArchModel, tea.Cmd) {
	id := m.selectedID
	// Advance selectedID before emit (app.go will call Refresh which re-selects)
	m.selectedID = ""
	return m, func() tea.Msg { return tuicore.DeleteNodeMsg{NodeID: id} }
}
```

- [ ] **Step 5: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/views/architecture.go
git commit -m "feat: canvas normal mode navigation and mode entry"
```

---

### Task 7: Move mode, connect mode, link mode

**Files:**
- Modify: `internal/tui/views/architecture.go`

- [ ] **Step 1: Implement handleMoveKey**

Replace the stub:

```go
func (m ArchModel) handleMoveKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()
	n, ok := m.arch.Nodes[m.selectedID]
	if !ok {
		m.moveMode = false
		return m, nil
	}

	switch {
	case key.Matches(msg, km.Up):
		n.Y = clampInt(n.Y-2, 0, canvasH-blockH)
		m.scrollToSelected()
	case key.Matches(msg, km.Down):
		n.Y = clampInt(n.Y+2, 0, canvasH-blockH)
		m.scrollToSelected()
	case key.Matches(msg, km.Left):
		n.X = clampInt(n.X-2, 0, canvasW-blockW)
		m.scrollToSelected()
	case key.Matches(msg, km.Right):
		n.X = clampInt(n.X+2, 0, canvasW-blockW)
		m.scrollToSelected()
	case key.Matches(msg, km.Move), key.Matches(msg, km.Enter):
		// Drop: confirm and set dirty
		m.moveMode = false
		id, x, y := n.ID, n.X, n.Y
		return m, func() tea.Msg { return tuicore.MoveNodeMsg{ID: id, X: x, Y: y} }
	case key.Matches(msg, km.Escape):
		// Restore original position
		n.X, n.Y = m.moveOriginX, m.moveOriginY
		m.moveMode = false
		id, x, y := n.ID, n.X, n.Y
		return m, func() tea.Msg { return tuicore.MoveNodeMsg{ID: id, X: x, Y: y} }
	}
	return m, nil
}
```

- [ ] **Step 2: Implement handleConnectKey**

Replace the stub:

```go
func (m ArchModel) handleConnectKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()

	switch {
	case key.Matches(msg, km.Up):
		if id := m.adjacentBlock("up"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Down):
		if id := m.adjacentBlock("down"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Left):
		if id := m.adjacentBlock("left"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Right):
		if id := m.adjacentBlock("right"); id != "" {
			m.selectedID = id
			m.scrollToSelected()
			return m, m.emitSelect()
		}
	case key.Matches(msg, km.Enter):
		if m.selectedID == m.connectSourceID {
			return m, func() tea.Msg {
				return tuicore.StatusMsg{Text: "Cannot connect a resource to itself"}
			}
		}
		from, to := m.connectSourceID, m.selectedID
		m.connectMode = false
		m.connectSourceID = ""
		return m, func() tea.Msg { return tuicore.ConnectNodesMsg{From: from, To: to} }
	case key.Matches(msg, km.Escape):
		m.connectMode = false
		m.connectSourceID = ""
	}
	return m, nil
}
```

- [ ] **Step 3: Implement handleLinkKey**

Replace the stub:

```go
func (m ArchModel) handleLinkKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()

	// The pivot for navigation is portTargetID if set, else connectSourceID.
	pivot := m.connectSourceID
	if m.portTargetID != "" {
		pivot = m.portTargetID
	}

	navigate := func(dir string) (ArchModel, tea.Cmd) {
		// Temporarily set selectedID to pivot for adjacentBlock to work
		oldSel := m.selectedID
		m.selectedID = pivot
		id := m.adjacentBlock(dir)
		m.selectedID = oldSel
		if id != "" && id != m.connectSourceID {
			m.portTargetID = id
			m.scrollToBlock(id)
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, km.Up):
		return navigate("up")
	case key.Matches(msg, km.Down):
		return navigate("down")
	case key.Matches(msg, km.Left):
		return navigate("left")
	case key.Matches(msg, km.Right):
		return navigate("right")
	case key.Matches(msg, km.Enter):
		if m.portTargetID == "" {
			return m, nil
		}
		from, to := m.connectSourceID, m.portTargetID
		m.portMode = false
		m.connectSourceID = ""
		m.portTargetID = ""
		m.selectedID = to
		return m, func() tea.Msg { return tuicore.ConnectNodesMsg{From: from, To: to} }
	case key.Matches(msg, km.Escape):
		m.portMode = false
		m.connectSourceID = ""
		m.portTargetID = ""
	}
	return m, nil
}

// scrollToBlock scrolls the viewport to make the given node visible.
func (m *ArchModel) scrollToBlock(id string) {
	old := m.selectedID
	m.selectedID = id
	m.scrollToSelected()
	m.selectedID = old
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/views/architecture.go
git commit -m "feat: move mode, connect mode, link mode"
```

---

### Task 8: Smart placement

**Files:**
- Modify: `internal/tui/views/architecture.go`

- [ ] **Step 1: Add compatible parent lookup and required imports**

Add `"cloudblocks-tui/internal/catalog"` and `"cloudblocks-tui/internal/aws/resources"` to the import block in `architecture.go`. The `catalog` package is needed to look up `DisplayName` for the smart placement status bar prompt. The `resources` package is needed for the `*resources.ResourceDef` parameter type in `confirmSmartPlacement`.

Updated import block:

```go
import (
    "math"
    "strings"
    "unicode/utf8"

    "github.com/charmbracelet/bubbles/key"
    "github.com/charmbracelet/bubbles/textinput"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"

    "cloudblocks-tui/internal/aws/resources"
    "cloudblocks-tui/internal/catalog"
    "cloudblocks-tui/internal/graph"
    "cloudblocks-tui/internal/tui/tuicore"
)
```

Add to `internal/tui/views/architecture.go`:

```go
// parentTFType maps ParentRefAttr values to the TFType of the compatible parent.
var parentTFType = map[string]string{
	"vpc_id":    "aws_vpc",
	"subnet_id": "aws_subnet",
}

// compatibleParents returns node IDs of nodes whose TFType matches the
// expected parent for the given ParentRefAttr.
func (m ArchModel) compatibleParents(parentRefAttr string) []string {
	wantType, ok := parentTFType[parentRefAttr]
	if !ok {
		return nil
	}
	var result []string
	for _, id := range m.arch.NodeOrder {
		if n, ok := m.arch.Nodes[id]; ok && n.Type == wantType {
			result = append(result, id)
		}
	}
	return result
}

// childCount returns how many nodes have an edge from parentID.
func (m ArchModel) childCount(parentID string) int {
	count := 0
	for _, e := range m.arch.Edges {
		if e.From == parentID {
			count++
		}
	}
	return count
}
```

- [ ] **Step 2: Implement handleStartSmartPlacement**

Replace the stub:

```go
func (m ArchModel) handleStartSmartPlacement(msg tuicore.StartSmartPlacementMsg) (ArchModel, tea.Cmd) {
	m.pendingNode = msg.Node

	def := catalog.ByTFType(msg.Node.Type)
	if def == nil || def.ParentRefAttr == "" {
		// No parent needed — place directly
		return m.confirmSmartPlacement("none", def)
	}

	compatible := m.compatibleParents(def.ParentRefAttr)
	if len(compatible) == 0 {
		// No compatible parents exist — place directly
		return m.confirmSmartPlacement("none", def)
	}

	// Build options list: compatible IDs + "none" sentinel
	m.smartPlacementOptions = append(append([]string{}, compatible...), "none")
	m.smartPlacementIdx = 0
	m.smartPlacementMode = true
	m.connectMode = false
	m.portMode = false
	m.moveMode = false

	displayName := msg.Node.Type
	if def != nil {
		displayName = def.DisplayName
	}
	prompt := m.buildSmartPlacementPrompt(displayName)
	return m, func() tea.Msg { return tuicore.StatusMsg{Text: prompt} }
}

// buildSmartPlacementPrompt builds the status bar text for smart placement.
func (m ArchModel) buildSmartPlacementPrompt(displayName string) string {
	parts := []string{"Connect " + displayName + " to:"}
	for i, id := range m.smartPlacementOptions {
		label := "[none]"
		if id != "none" {
			if n, ok := m.arch.Nodes[id]; ok {
				label = n.Name
			}
		}
		if i == m.smartPlacementIdx {
			label = ">" + label + "<"
		}
		parts = append(parts, label)
	}
	parts = append(parts, "(↑↓ select, Enter confirm)")
	return strings.Join(parts, "  ")
}
```

- [ ] **Step 3: Implement handleSmartPlacementKey**

Replace the stub:

```go
func (m ArchModel) handleSmartPlacementKey(msg tea.KeyMsg) (ArchModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()
	def := catalog.ByTFType(m.pendingNode.Type)

	switch {
	case key.Matches(msg, km.Up):
		if m.smartPlacementIdx > 0 {
			m.smartPlacementIdx--
		}
		displayName := m.pendingNode.Type
		if def != nil {
			displayName = def.DisplayName
		}
		prompt := m.buildSmartPlacementPrompt(displayName)
		return m, func() tea.Msg { return tuicore.StatusMsg{Text: prompt} }

	case key.Matches(msg, km.Down):
		if m.smartPlacementIdx < len(m.smartPlacementOptions)-1 {
			m.smartPlacementIdx++
		}
		displayName := m.pendingNode.Type
		if def != nil {
			displayName = def.DisplayName
		}
		prompt := m.buildSmartPlacementPrompt(displayName)
		return m, func() tea.Msg { return tuicore.StatusMsg{Text: prompt} }

	case key.Matches(msg, km.Enter):
		chosen := m.smartPlacementOptions[m.smartPlacementIdx]
		return m.confirmSmartPlacement(chosen, def)

	case key.Matches(msg, km.Escape):
		return m.confirmSmartPlacement("none", def)
	}
	return m, nil
}

// confirmSmartPlacement finalises placement of pendingNode.
// chosenParentID is either a node ID or "none".
func (m ArchModel) confirmSmartPlacement(chosenParentID string, def *resources.ResourceDef) (ArchModel, tea.Cmd) {
	n := m.pendingNode
	m.pendingNode = nil
	m.smartPlacementMode = false
	m.smartPlacementOptions = nil
	m.smartPlacementIdx = 0

	if chosenParentID != "none" {
		parent, ok := m.arch.Nodes[chosenParentID]
		if ok {
			cc := m.childCount(chosenParentID)
			n.X = parent.X + 20*cc
			n.Y = parent.Y + 6
		}
		m.arch.AddNode(n)
		m.arch.Connect(chosenParentID, n.ID)
	} else {
		idx := len(m.arch.Nodes)
		n.X, n.Y = StaggerPosition(idx)
		m.arch.AddNode(n)
	}

	m.selectedID = n.ID
	m.scrollToSelected()

	// Emit MoveNodeMsg to signal dirty to app.go
	id, x, y := n.ID, n.X, n.Y
	return m, func() tea.Msg { return tuicore.MoveNodeMsg{ID: id, X: x, Y: y} }
}
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/views/architecture.go
git commit -m "feat: smart placement on resource add"
```

---

### Task 9: Update app.go

Wire up `MoveNodeMsg`, modify `AddNodeMsg` handling for smart placement, update `archLabel` in `View`, and confirm `SetSize` is called correctly.

**Files:**
- Modify: `internal/tui/app.go`

- [ ] **Step 1: Handle MoveNodeMsg**

In `app.go`, in the `// Handle sub-model output messages.` switch block, add:

```go
case MoveNodeMsg:
    // MoveNodeMsg signals that a block was moved or placed — set dirty.
    m.dirty = true
```

- [ ] **Step 2: Update AddNodeMsg handler to route smart placement**

Replace the existing `case AddNodeMsg:` block:

```go
case AddNodeMsg:
    n := &graph.Node{
        ID:         fmt.Sprintf("%s-%d", msg.Def.TFType, len(m.arch.Nodes)+1),
        Type:       msg.Def.TFType,
        Name:       msg.Def.DisplayName,
        Properties: copyProps(msg.Def.DefaultProps),
    }
    if msg.Def.ParentRefAttr != "" {
        // Smart placement: hand the pending node to ArchModel.
        // ArchModel calls arch.AddNode once the user confirms the parent.
        var cmd tea.Cmd
        m.archV, cmd = m.archV.Update(StartSmartPlacementMsg{Node: n})
        cmds = append(cmds, cmd)
    } else {
        // No parent ref: compute stagger position and add directly.
        n.X, n.Y = views.StaggerPosition(len(m.arch.Nodes))
        m.arch.AddNode(n)
        m.dirty = true
        m.archV = m.archV.Refresh(m.arch)
    }
    // (end of case AddNodeMsg)
```

- [ ] **Step 3: Update arch panel label in View to reflect link/smart placement modes**

In `View()`, find the `archLabel` section and extend it:

```go
archLabel := "ARCHITECTURE"
if m.archV.InConnectMode() {
    archLabel = "ARCHITECTURE [CONNECT]"
} else if m.archV.InLinkMode() {
    archLabel = "ARCHITECTURE [LINK]"
} else if m.archV.InSmartPlacementMode() {
    archLabel = "ARCHITECTURE [PLACING]"
}
```

- [ ] **Step 4: Confirm SetSize is already called for archV**

Check that `tea.WindowSizeMsg` handler already calls `m.archV = m.archV.SetSize(archW, innerH)`. Looking at the existing code at line 112: yes, it does. No change needed.

- [ ] **Step 5: Verify it compiles**

```bash
go build ./...
```

Expected: success.

- [ ] **Step 6: Run all tests**

```bash
go test ./...
```

Expected: all tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go
git commit -m "feat: wire MoveNodeMsg, smart placement routing, arch label modes"
```

---

### Task 10: Manual smoke test

Verify the full canvas workflow end-to-end.

**Files:** none

- [ ] **Step 1: Build the binary**

```bash
go build -o cloudblocks ./cmd/cloudblocks/
```

Expected: builds without error.

- [ ] **Step 2: Run and test the workflow**

```bash
./cloudblocks
```

Perform these steps manually:
1. Add VPC from Catalog (`A`). Verify block appears on canvas.
2. Add Subnet. Verify smart placement prompt appears asking which VPC to connect to. Select the VPC and press `Enter`. Verify Subnet block appears below VPC with a connection line.
3. Add ALB (no ParentRefAttr). Verify it appears at a staggered position.
4. Use `C` key (connect mode) to connect Subnet→ALB. Verify connection line appears.
5. Add ECS. Use `L` key (link mode) to connect ALB→ECS.
6. Select VPC. Press `M`. Use arrow keys to move it. Press `Enter` to drop.
7. Press `S` to save. Verify `cloudblocks.json` created.
8. Press `Q` to quit and re-launch. Verify load prompt appears and positions are restored.

- [ ] **Step 3: Export Terraform and verify**

```bash
./cloudblocks
# load cloudblocks.json, then press X to export
cat generated/main.tf
```

Expected: valid HCL with resource blocks and cross-references.

- [ ] **Step 4: Final commit (if any fixes were needed)**

```bash
git add internal/tui/views/architecture.go internal/tui/app.go
git commit -m "fix: smoke test fixes"
```

---

## Summary of files changed

| File | Change |
|---|---|
| `internal/graph/node.go` | Added `X, Y int` with JSON tags |
| `internal/graph/persistence_test.go` | Added `TestNodeXY_RoundTrip` |
| `internal/renderer/ascii.go` | **Deleted** |
| `internal/renderer/ascii_test.go` | **Deleted** |
| `internal/tui/tuicore/messages.go` | Added `MoveNodeMsg`, `StartSmartPlacementMsg` |
| `internal/tui/tuicore/keymap.go` | Added `Left`, `Right`, `Move`, `Link` bindings |
| `internal/tui/messages.go` | Added type aliases for new message types |
| `internal/tui/views/architecture.go` | Full rewrite: canvas render + all interaction modes |
| `internal/tui/app.go` | `AddNodeMsg` routing, `MoveNodeMsg` handler, arch label update |
