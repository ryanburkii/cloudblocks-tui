// internal/tui/views/catalog.go
package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/aws/resources"
	"cloudblocks-tui/internal/catalog"
	"cloudblocks-tui/internal/tui/tuicore"
)

// catalogItem is one row in the catalog list (either a category header or a resource).
type catalogItem struct {
	def      *resources.ResourceDef // nil if header
	label    string
	isHeader bool
}

// CatalogModel is the left-panel sub-model.
type CatalogModel struct {
	items  []catalogItem
	cursor int
	width  int
	height int
}

var (
	catHeaderStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))
	catItemStyle     = lipgloss.NewStyle().PaddingLeft(2)
	catSelectedStyle = lipgloss.NewStyle().PaddingLeft(2).
				Background(lipgloss.Color("62")).Foreground(lipgloss.Color("230"))
)

// NewCatalog returns an initialized CatalogModel.
func NewCatalog() CatalogModel {
	order, byCategory := catalog.ByCategory()
	var items []catalogItem
	for _, cat := range order {
		items = append(items, catalogItem{label: cat, isHeader: true})
		for _, def := range byCategory[cat] {
			items = append(items, catalogItem{def: def, label: def.DisplayName})
		}
	}
	// Start cursor on the first non-header item.
	cursor := 0
	for i, it := range items {
		if !it.isHeader {
			cursor = i
			break
		}
	}
	return CatalogModel{items: items, cursor: cursor}
}

func (m CatalogModel) SetSize(w, h int) CatalogModel {
	m.width = w
	m.height = h
	return m
}

func (m CatalogModel) Update(msg tea.Msg) (CatalogModel, tea.Cmd) {
	km := tuicore.DefaultKeyMap()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, km.Up):
			m.cursor = m.prevResource(m.cursor)
		case key.Matches(msg, km.Down):
			m.cursor = m.nextResource(m.cursor)
		case key.Matches(msg, km.Add), key.Matches(msg, km.Enter):
			if m.cursor < len(m.items) && !m.items[m.cursor].isHeader {
				return m, func() tea.Msg {
					return tuicore.AddNodeMsg{Def: m.items[m.cursor].def}
				}
			}
		}
	}
	return m, nil
}

func (m CatalogModel) View() string {
	var sb strings.Builder
	for i, it := range m.items {
		if it.isHeader {
			sb.WriteString(catHeaderStyle.Render(it.label) + "\n")
		} else if i == m.cursor {
			sb.WriteString(catSelectedStyle.Render(it.label) + "\n")
		} else {
			sb.WriteString(catItemStyle.Render(it.label) + "\n")
		}
	}
	return sb.String()
}

// nextResource returns the index of the next non-header item after cursor.
func (m CatalogModel) nextResource(cursor int) int {
	for i := cursor + 1; i < len(m.items); i++ {
		if !m.items[i].isHeader {
			return i
		}
	}
	return cursor
}

// prevResource returns the index of the previous non-header item before cursor.
func (m CatalogModel) prevResource(cursor int) int {
	for i := cursor - 1; i >= 0; i-- {
		if !m.items[i].isHeader {
			return i
		}
	}
	return cursor
}
