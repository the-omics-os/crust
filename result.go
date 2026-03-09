// Package crust provides life sciences TUI components for Bubble Tea.
//
// Crust components are standalone tea.Model implementations with typed
// Go constructors and functional options. They signal completion via
// tea.Cmd returning SubmitMsg or CancelMsg.
package crust

// SubmitMsg signals that a component has completed with user-confirmed data.
// Returned via tea.Cmd from Update().
type SubmitMsg struct {
	Component string
	Data      map[string]any
}

// CancelMsg signals that a component interaction was cancelled.
type CancelMsg struct {
	Component string
	Reason    string
}
