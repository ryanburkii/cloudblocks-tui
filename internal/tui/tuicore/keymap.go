// internal/tui/tuicore/keymap.go
package tuicore

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for the application.
type KeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Tab     key.Binding
	Enter   key.Binding
	Escape  key.Binding
	Add     key.Binding
	Connect key.Binding
	Move    key.Binding
	Link    key.Binding
	Delete  key.Binding
	Rename  key.Binding
	Edit    key.Binding
	Save    key.Binding
	Export  key.Binding
	Deploy  key.Binding
	Quit    key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
		Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
		Left:    key.NewBinding(key.WithKeys("left"), key.WithHelp("←", "left")),
		Right:   key.NewBinding(key.WithKeys("right"), key.WithHelp("→", "right")),
		Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next panel")),
		Enter:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "select")),
		Escape:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
		Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
		Connect: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "connect")),
		Move:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "move block")),
		Link:    key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "link")),
		Delete:  key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		Rename:  key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rename")),
		Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit props")),
		Save:    key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "save")),
		Export:  key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "export TF")),
		Deploy:  key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "deploy")),
		Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	}
}
