// internal/tui/messages.go
package tui

import "cloudblocks-tui/internal/tui/tuicore"

// Re-export message types from tuicore so callers can use either import path.
// Using type aliases ensures tui.AddNodeMsg and tuicore.AddNodeMsg are the same type.
type AddNodeMsg = tuicore.AddNodeMsg
type SelectNodeMsg = tuicore.SelectNodeMsg
type ConnectNodesMsg = tuicore.ConnectNodesMsg
type DeleteNodeMsg = tuicore.DeleteNodeMsg
type RenameNodeMsg = tuicore.RenameNodeMsg
type UpdatePropMsg = tuicore.UpdatePropMsg
type DeployLineMsg = tuicore.DeployLineMsg
type DeployDoneMsg = tuicore.DeployDoneMsg
type StatusMsg = tuicore.StatusMsg
