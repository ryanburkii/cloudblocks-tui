// internal/tui/keymap.go
package tui

import "cloudblocks-tui/internal/tui/tuicore"

// Re-export KeyMap and DefaultKeyMap from tuicore.
// Using a type alias ensures tui.KeyMap and tuicore.KeyMap are the same type.
type KeyMap = tuicore.KeyMap

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return tuicore.DefaultKeyMap()
}
