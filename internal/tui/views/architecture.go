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
	// TODO(Task 3): uncomment when StartSmartPlacementMsg is added to tuicore
	// if spm, ok := msg.(tuicore.StartSmartPlacementMsg); ok {
	// 	return m.handleStartSmartPlacement(spm)
	// }

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

// TODO(Task 3): uncomment when StartSmartPlacementMsg is added to tuicore
// func (m ArchModel) handleStartSmartPlacement(msg tuicore.StartSmartPlacementMsg) (ArchModel, tea.Cmd) {
// 	return m, nil
// }

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
