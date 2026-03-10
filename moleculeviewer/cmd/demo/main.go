package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
	"github.com/the-omics-os/crust/moleculeviewer"
)

type model struct {
	viewer moleculeviewer.Model
	active int
	status string
}

func newModel(index int) model {
	m := model{}
	m.load(index)
	return m
}

func (m *model) load(index int) {
	index = ((index % len(samples)) + len(samples)) % len(samples)
	s := samples[index]
	m.active = index
	m.viewer = moleculeviewer.New(
		moleculeviewer.WithName(s.name),
		moleculeviewer.WithMolecule(loadSample(index)),
		moleculeviewer.WithWidth(96),
		moleculeviewer.WithHeight(26),
	)
	m.status = fmt.Sprintf("Loaded %s", s.name)
}

func (m model) Init() tea.Cmd { return m.viewer.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.load(0)
			return m, nil
		case "2":
			m.load(1)
			return m, nil
		case "3":
			m.load(2)
			return m, nil
		}
	case crust.SubmitMsg:
		m.status = fmt.Sprintf("Selected %v", msg.Data["atom_label"])
		return m, nil
	case crust.CancelMsg:
		m.status = "Viewer cancel triggered"
		return m, nil
	}

	updated, cmd := m.viewer.Update(msg)
	m.viewer = updated.(moleculeviewer.Model)
	return m, cmd
}

func (m model) View() tea.View {
	footer := fmt.Sprintf(
		"\n\n  1 Caffeine  2 Aspirin  3 Nicotine  |  Arrows move  Tab plane  / search  ? help  Enter select  q quit\n  %s\n",
		m.status,
	)
	v := tea.NewView(m.viewer.Render() + footer)
	v.AltScreen = true
	return v
}

func main() {
	p := tea.NewProgram(newModel(0))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
