// cmd/cloudblocks/main.go
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"cloudblocks-tui/internal/tui"
)

func main() {
	// Detect if cloudblocks.json exists to show the load prompt.
	_, err := os.Stat("cloudblocks.json")
	showLoadPrompt := err == nil

	m := tui.New(showLoadPrompt)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
