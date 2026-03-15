// internal/tui/app.go
package tui

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
		m.catalogV = m.catalogV.SetSize(catalogW, innerH)
		m.archV = m.archV.SetSize(archW, innerH)
		m.propsV = m.propsV.SetSize(propsW, innerH/2)
		m.deployV = m.deployV.SetSize(propsW, innerH/2)

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
			m.deployLines = make(chan string, 1024)
			m.deployDone = make(chan deploy.Result, 1)
			deploy.Run(generatedDir, m.deployLines, m.deployDone)
			m.deployActive = true
			m.deployV = views.NewDeploy()
			propsW := m.propsV.Width()
			innerH := m.height - 4
			m.deployV = m.deployV.SetSize(propsW, innerH/2)
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
	case MoveNodeMsg:
		// MoveNodeMsg signals that a block was moved or placed — set dirty.
		m.dirty = true

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

	case SelectNodeMsg:
		if n, ok := m.arch.Nodes[msg.NodeID]; ok {
			def := catalog.ByTFType(n.Type)
			m.propsV = m.propsV.SetNode(n, def)
			// E key in Architecture panel sets FocusProps: true; shift focus
			// to Properties panel as required by the spec.
			if msg.FocusProps {
				m.focused = PanelProperties
			}
		}

	case ConnectNodesMsg:
		if msg.From != msg.To {
			m.arch.Connect(msg.From, msg.To)
			m.dirty = true
			m.archV = m.archV.Refresh(m.arch)
		}

	case DeleteNodeMsg:
		m.arch.RemoveNode(msg.NodeID)
		m.dirty = true
		m.archV = m.archV.Refresh(m.arch)
		m.propsV = m.propsV.SetNode(nil, nil)

	case RenameNodeMsg:
		if n, ok := m.arch.Nodes[msg.NodeID]; ok {
			n.Name = msg.Name
			m.dirty = true
			m.archV = m.archV.Refresh(m.arch)
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
	} else if m.archV.InLinkMode() {
		archLabel = "ARCHITECTURE [LINK]"
	} else if m.archV.InSmartPlacementMode() {
		archLabel = "ARCHITECTURE [PLACING]"
	}
	archPanel := archStyle.Width(archW).Render(
		titleStyle.Render(archLabel) + "\n" + m.archV.View(),
	)

	rightBottom := m.actionsView()
	if m.deployActive {
		rightBottom = m.deployV.View()
	}
	var rightContent string
	if m.deployActive {
		// During deploy the Actions sub-panel is replaced by the deploy output
		// panel; deployV.View() already renders its own "DEPLOY OUTPUT" header.
		rightContent = titleStyle.Render("PROPERTIES") + "\n" + m.propsV.View() +
			"\n" + rightBottom
	} else {
		rightContent = titleStyle.Render("PROPERTIES") + "\n" + m.propsV.View() +
			"\n" + titleStyle.Render("ACTIONS") + "\n" + rightBottom
	}
	rightPanel := propsStyle.Width(propsW).Render(rightContent)

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
		lines = append(lines, actionKeyStyle.Render("[P]")+" Deploy")
	} else {
		lines = append(lines, mutedStyle.Render("[P] Deploy (terraform not found)"))
	}
	lines = append(lines, actionKeyStyle.Render("[Q]")+" Quit")
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
