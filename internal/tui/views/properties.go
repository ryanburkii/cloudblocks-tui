// internal/tui/views/properties.go
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/graph"
	"cloudblocks-tui/internal/tui/tuicore"
)

// PropsModel is the top-right panel for editing node properties.
type PropsModel struct {
	node    *graph.Node
	def     *resources.ResourceDef
	cursor  int
	editing bool
	input   textinput.Model
	width   int
	height  int
}

var (
	propKeyStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	propValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("222"))
)

// NewProps returns an empty PropsModel (no node selected yet).
func NewProps() PropsModel {
	ti := textinput.New()
	return PropsModel{input: ti}
}

// SetNode updates the model with a new node and its ResourceDef.
// Passing nil clears the panel.
func (m PropsModel) SetNode(n *graph.Node, def *resources.ResourceDef) PropsModel {
	m.node = n
	m.def = def
	m.cursor = 0
	m.editing = false
	return m
}

func (m PropsModel) Width() int { return m.width }

func (m PropsModel) SetSize(w, h int) PropsModel {
	m.width = w
	m.height = h
	return m
}

func (m PropsModel) Update(msg tea.Msg) (PropsModel, tea.Cmd) {
	if m.node == nil || m.def == nil {
		return m, nil
	}
	km := tuicore.DefaultKeyMap()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			switch msg.String() {
			case "enter":
				pd := m.def.PropSchema[m.cursor]
				raw := m.input.Value()
				val := parseValue(raw, pd.Type)
				nodeID := m.node.ID
				m.editing = false
				m.input.Blur()
				return m, func() tea.Msg {
					return tuicore.UpdatePropMsg{NodeID: nodeID, Key: pd.Key, Value: val}
				}
			case "esc":
				m.editing = false
				m.input.Blur()
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				return m, cmd
			}
		} else {
			switch {
			case key.Matches(msg, km.Up):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, km.Down):
				if m.cursor < len(m.def.PropSchema)-1 {
					m.cursor++
				}
			case key.Matches(msg, km.Enter):
				if len(m.def.PropSchema) > 0 {
					pd := m.def.PropSchema[m.cursor]
					current := fmt.Sprintf("%v", m.node.Properties[pd.Key])
					m.input.SetValue(current)
					m.input.Focus()
					m.editing = true
				}
			}
		}
	}
	return m, nil
}

func (m PropsModel) View() string {
	if m.node == nil {
		return mutedStyle.Render("(no resource selected)")
	}

	var sb strings.Builder
	sb.WriteString(propKeyStyle.Render(m.node.Name) + "\n")

	if m.def == nil || len(m.def.PropSchema) == 0 {
		sb.WriteString(mutedStyle.Render("(no configurable properties)"))
		return sb.String()
	}

	for i, pd := range m.def.PropSchema {
		val := m.node.Properties[pd.Key]
		if i == m.cursor && m.editing {
			sb.WriteString(propKeyStyle.Render(pd.Label+": ") + m.input.View() + "\n")
		} else if i == m.cursor {
			line := propKeyStyle.Render(pd.Label+": ") + propValueStyle.Render(fmt.Sprintf("%v", val))
			sb.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("237")).Render(line) + "\n")
		} else {
			sb.WriteString(propKeyStyle.Render(pd.Label+": ") + propValueStyle.Render(fmt.Sprintf("%v", val)) + "\n")
		}
	}
	return sb.String()
}

// parseValue converts a string input to the appropriate Go type based on PropType.
func parseValue(raw string, pt resources.PropType) interface{} {
	switch pt {
	case resources.PropTypeInt:
		var n int
		fmt.Sscanf(raw, "%d", &n)
		return n
	case resources.PropTypeBool:
		return raw == "true" || raw == "1" || raw == "yes"
	default:
		return raw
	}
}
