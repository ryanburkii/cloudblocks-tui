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

// truncatePad truncates s to at most n runes, padding with spaces to exactly n.
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

// runeLen returns the number of runes in s.
func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

// Update processes input events.
func (m ArchModel) Update(msg tea.Msg) (ArchModel, tea.Cmd) {
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
	case "right":
		// Horizontal-first L-shape, travelling rightward
		if ey == ny {
			drawHLine(grid, ex, nx, ey, fg)
		} else {
			drawHLine(grid, ex, nx, ey, fg)
			drawVLine(grid, nx, ey, ny, fg)
			// Corner: arrived from left, bending down=┐ or up=┘
			if ny > ey {
				setCell(grid, nx, ey, '┐', fg, "")
			} else {
				setCell(grid, nx, ey, '┘', fg, "")
			}
		}
	case "left":
		// Horizontal-first L-shape, travelling leftward (nx < ex)
		if ey == ny {
			drawHLine(grid, ex, nx, ey, fg)
		} else {
			drawHLine(grid, ex, nx, ey, fg)
			drawVLine(grid, nx, ey, ny, fg)
			// Corner: arrived from right, bending down=┌ or up=└
			if ny > ey {
				setCell(grid, nx, ey, '┌', fg, "")
			} else {
				setCell(grid, nx, ey, '└', fg, "")
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
	case "right":
		if ey == ny {
			drawDashedH(ex, nx, ey)
		} else {
			drawDashedH(ex, nx, ey)
			drawDashedV(nx, ey, ny)
			if ny > ey {
				setCell(grid, nx, ey, '┐', fg, "")
			} else {
				setCell(grid, nx, ey, '┘', fg, "")
			}
		}
	case "left":
		if ey == ny {
			drawDashedH(ex, nx, ey)
		} else {
			drawDashedH(ex, nx, ey)
			drawDashedV(nx, ey, ny)
			if ny > ey {
				setCell(grid, nx, ey, '┌', fg, "")
			} else {
				setCell(grid, nx, ey, '└', fg, "")
			}
		}
	case "bottom", "top":
		if ex == nx {
			drawDashedV(ex, ey, ny)
		} else {
			drawDashedV(ex, ey, ny)
			drawDashedH(ex, nx, ny)
			// Corner at (ex, ny): same logic as drawEdge bottom/top
			if dir == "bottom" {
				if nx > ex {
					setCell(grid, ex, ny, '┌', fg, "")
				} else {
					setCell(grid, ex, ny, '┐', fg, "")
				}
			} else {
				if nx > ex {
					setCell(grid, ex, ny, '└', fg, "")
				} else {
					setCell(grid, ex, ny, '┘', fg, "")
				}
			}
		}
	}
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
	return m.renderCanvas()
}

// renderCanvas builds the full canvas view string.
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
