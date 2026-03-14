// internal/tui/layout.go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62")).
				Padding(0, 1)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("62"))

	selectedItemStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("62")).
				Foreground(lipgloss.Color("230"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	actionKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("39"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("82"))
)

// panelSizes returns (catalogW, archW, propsW) from total terminal width.
func panelSizes(totalWidth int) (int, int, int) {
	// 3 panels × (2 border cols + 2 padding cols) = 12 chars overhead
	available := totalWidth - 12
	if available < 60 {
		available = 60
	}
	catalogW := available * 20 / 100
	propsW := available * 28 / 100
	archW := available - catalogW - propsW
	return catalogW, archW, propsW
}
