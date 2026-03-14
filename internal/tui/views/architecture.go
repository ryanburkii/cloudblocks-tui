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
	"cloudblocks-tui/internal/tui/tuicore"
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
	km := tuicore.DefaultKeyMap()
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
				return tuicore.StatusMsg{Text: "Select target to connect (Esc to cancel)"}
			}

		case key.Matches(msg, km.Enter):
			if m.connectMode && m.selectedNode() != nil {
				target := m.selectedNode().ID
				if target == m.connectFrom {
					// Spec: self-loop rejected, connect mode stays active
					return m, func() tea.Msg {
						return tuicore.StatusMsg{Text: "Cannot connect a resource to itself"}
					}
				}
				from := m.connectFrom
				m.connectMode = false
				m.connectFrom = ""
				return m, func() tea.Msg {
					return tuicore.ConnectNodesMsg{From: from, To: target}
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
					return tuicore.DeleteNodeMsg{NodeID: nodeID}
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
	return func() tea.Msg { return tuicore.SelectNodeMsg{NodeID: n.ID} }
}

// buildFlatList returns all nodes in DFS order starting from roots.
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
