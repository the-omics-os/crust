package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/the-omics-os/crust"
	"github.com/the-omics-os/crust/periodictable"
)

type app struct {
	table periodictable.Model
	done  bool
}

func (m app) Init() tea.Cmd { return m.table.Init() }

func (m app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case crust.SubmitMsg, crust.CancelMsg:
		m.done = true
		return m, tea.Quit
	}

	updated, cmd := m.table.Update(msg)
	m.table = updated.(periodictable.Model)
	return m, cmd
}

func (m app) View() tea.View {
	if m.done {
		return tea.NewView("Closed.\n")
	}

	view := tea.NewView(m.table.Render() + "\n\nq: quit demo wrapper\n")
	view.AltScreen = true
	return view
}

func main() {
	model := periodictable.New(
		periodictable.WithWidth(118),
		periodictable.WithSelected("Fe"),
		periodictable.WithHighlights("C", "N", "O", "S"),
	)

	program := tea.NewProgram(app{table: model})
	if _, err := program.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
