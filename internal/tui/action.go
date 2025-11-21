package tui

// ActionType defines supported UI actions triggered by key bindings.
type ActionType string

const (
	// ActionToggleDebugger toggles the debugger mode on the dev container.
	ActionToggleDebugger ActionType = "toggle_debugger"
	// ActionQuit requests the entire program to stop.
	ActionQuit ActionType = "quit"
)

// Action represents a single shortcut invocation from the UI.
type Action struct {
	Type    ActionType
	Enabled bool
}
