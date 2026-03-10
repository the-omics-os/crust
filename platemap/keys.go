package platemap

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Up         key.Binding
	Down       key.Binding
	Left       key.Binding
	Right      key.Binding
	ShiftUp    key.Binding
	ShiftDown  key.Binding
	ShiftLeft  key.Binding
	ShiftRight key.Binding
	Home       key.Binding
	End        key.Binding
	PageUp     key.Binding
	PageDown   key.Binding
	NextLens   key.Binding
	PrevLens   key.Binding
	RowSweep   key.Binding
	ColSweep   key.Binding
	Inspect    key.Binding
	Confirm    key.Binding
	Back       key.Binding
	Help       key.Binding
}

var defaultKeyMap = keyMap{
	Up:         key.NewBinding(key.WithKeys("up")),
	Down:       key.NewBinding(key.WithKeys("down")),
	Left:       key.NewBinding(key.WithKeys("left")),
	Right:      key.NewBinding(key.WithKeys("right")),
	ShiftUp:    key.NewBinding(key.WithKeys("shift+up")),
	ShiftDown:  key.NewBinding(key.WithKeys("shift+down")),
	ShiftLeft:  key.NewBinding(key.WithKeys("shift+left")),
	ShiftRight: key.NewBinding(key.WithKeys("shift+right")),
	Home:       key.NewBinding(key.WithKeys("home")),
	End:        key.NewBinding(key.WithKeys("end")),
	PageUp:     key.NewBinding(key.WithKeys("pgup")),
	PageDown:   key.NewBinding(key.WithKeys("pgdown")),
	NextLens:   key.NewBinding(key.WithKeys("tab")),
	PrevLens:   key.NewBinding(key.WithKeys("shift+tab")),
	RowSweep:   key.NewBinding(key.WithKeys("r")),
	ColSweep:   key.NewBinding(key.WithKeys("c")),
	Inspect:    key.NewBinding(key.WithKeys(" ", "space", "i")),
	Confirm:    key.NewBinding(key.WithKeys("enter")),
	Back:       key.NewBinding(key.WithKeys("esc")),
	Help:       key.NewBinding(key.WithKeys("?")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		key.NewBinding(key.WithKeys("arrows"), key.WithHelp("arrows", "move")),
		key.NewBinding(key.WithKeys("space/i"), key.WithHelp("space/i", "inspect")),
		key.NewBinding(key.WithKeys("r/c"), key.WithHelp("r/c", "sweep")),
		key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "lens")),
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm")),
		key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			key.NewBinding(key.WithKeys("↑/↓/←/→"), key.WithHelp("↑/↓/←/→", "move focus")),
			key.NewBinding(key.WithKeys("home/end"), key.WithHelp("home/end", "row edge")),
			key.NewBinding(key.WithKeys("pgup/pgdn"), key.WithHelp("pgup/pgdn", "jump viewport")),
			key.NewBinding(key.WithKeys("shift+arrows"), key.WithHelp("shift+arrows", "sweep while moving")),
		},
		{
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "toggle row sweep")),
			key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "toggle column sweep")),
			key.NewBinding(key.WithKeys("space/i"), key.WithHelp("space/i", "inspect focused well")),
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "confirm focused well")),
		},
		{
			key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next lens")),
			key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "previous lens")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back / close")),
			key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle full help")),
		},
	}
}
