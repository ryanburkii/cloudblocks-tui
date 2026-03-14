// internal/tui/tuicore/messages.go
package tuicore

import "cloudblocks-tui/internal/aws/resources"

// AddNodeMsg is emitted by CatalogModel when the user adds a resource.
type AddNodeMsg struct{ Def *resources.ResourceDef }

// SelectNodeMsg is emitted by ArchModel when the user moves the cursor.
// FocusProps is true when emitted by the E key — the spec requires focus to
// shift to the Properties panel in that case.
type SelectNodeMsg struct {
	NodeID     string
	FocusProps bool
}

// ConnectNodesMsg is emitted by ArchModel when connect mode completes.
type ConnectNodesMsg struct{ From, To string }

// DeleteNodeMsg is emitted by ArchModel when the user deletes a node.
type DeleteNodeMsg struct{ NodeID string }

// RenameNodeMsg is emitted by ArchModel when a rename is confirmed.
type RenameNodeMsg struct{ NodeID, Name string }

// UpdatePropMsg is emitted by PropsModel when a property is edited.
type UpdatePropMsg struct {
	NodeID string
	Key    string
	Value  interface{}
}

// DeployLineMsg carries one line of terraform output.
type DeployLineMsg struct{ Line string }

// DeployDoneMsg signals that the deploy subprocess has exited.
type DeployDoneMsg struct{ ExitCode int }

// StatusMsg sets a transient status bar message.
type StatusMsg struct{ Text string }
