// internal/tui/views/deploy.go
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"cloudblocks-tui/internal/deploy"
	"cloudblocks-tui/internal/tui/tuicore"
)

// DeployModel is the bottom-right panel that shows live terraform output.
type DeployModel struct {
	lines    []string
	done     bool
	exitCode int
	vp       viewport.Model
	width    int
	height   int
}

var deployOutputStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("250"))

// NewDeploy returns a fresh DeployModel (no output yet).
func NewDeploy() DeployModel {
	return DeployModel{}
}

func (m DeployModel) SetSize(w, h int) DeployModel {
	m.width = w
	m.height = h
	m.vp = viewport.New(w, h)
	return m
}

// WaitForLine returns a tea.Cmd that reads the next line from lines.
// When lines is closed (all output read), it reads the final Result from done
// and returns DeployDoneMsg.
func (m DeployModel) WaitForLine(lines <-chan string, done <-chan deploy.Result) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-lines
		if ok {
			return tuicore.DeployLineMsg{Line: line}
		}
		// lines closed — read the final result from done
		result := <-done
		return tuicore.DeployDoneMsg{ExitCode: result.ExitCode}
	}
}

func (m DeployModel) Update(msg tea.Msg) (DeployModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tuicore.DeployLineMsg:
		m.lines = append(m.lines, msg.Line)
		m.vp.SetContent(strings.Join(m.lines, "\n"))
		m.vp.GotoBottom()

	case tuicore.DeployDoneMsg:
		m.done = true
		m.exitCode = msg.ExitCode
		status := "Deploy complete ✓"
		if msg.ExitCode != 0 {
			status = fmt.Sprintf("Deploy failed (exit %d)", msg.ExitCode)
		}
		m.lines = append(m.lines, "", status)
		m.vp.SetContent(strings.Join(m.lines, "\n"))
		m.vp.GotoBottom()
	}
	return m, nil
}

func (m DeployModel) View() string {
	if m.width == 0 {
		return ""
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Render("DEPLOY OUTPUT")
	return header + "\n" + deployOutputStyle.Render(m.vp.View())
}
