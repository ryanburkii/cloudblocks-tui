// internal/tui/views/styles.go
package views

import "github.com/charmbracelet/lipgloss"

// Shared lipgloss styles used across all view sub-models.
var (
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)
