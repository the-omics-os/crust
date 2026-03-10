package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust/sequenceviewer"
)

type demoModel struct {
	viewer   sequenceviewer.Model
	mode     string
	lastSize tea.WindowSizeMsg
}

func newDNAViewer() sequenceviewer.Model {
	sequence := strings.Repeat(
		"ATGGCCATTGTAATGGGCCGCTGAAAGGGTGCCCGATAGGATCCGAATTCGCTAGCATGCGTACGTAGCTA",
		4,
	)
	return sequenceviewer.New(
		sequenceviewer.WithSequence(
			sequence,
			sequenceviewer.DNA,
		),
		sequenceviewer.WithComplement(true),
		sequenceviewer.WithGCWindow(12),
		sequenceviewer.WithAnnotations([]sequenceviewer.Annotation{
			{Name: "Promoter", Start: 1, End: 42, Direction: 1, Color: lipgloss.Color("81")},
			{Name: "ORF", Start: 43, End: 210, Direction: 1, Color: lipgloss.Color("42")},
			{Name: "EcoRI/BamHI zone", Start: 175, End: 236, Direction: 0, Color: lipgloss.Color("214")},
			{Name: "Terminator", Start: 250, End: 284, Direction: -1, Color: lipgloss.Color("205")},
		}),
	)
}

func newProteinViewer() sequenceviewer.Model {
	sequence := strings.Repeat(
		"MKWVTFISLLFLFSSAYSRGVFRRDTHKSEIAHRFKDLGEENFKALVLIAFAQYLQQCPFDEHVKLVNEVTEFAKTCVADESAENCDKSLHTLFGDELCKVASLRETYGEMADCCAKQEPERNECFLSHKDDSPDLPKLKPDPN",
		3,
	)
	return sequenceviewer.New(
		sequenceviewer.WithSequence(
			sequence,
			sequenceviewer.Protein,
		),
		sequenceviewer.WithAnnotations([]sequenceviewer.Annotation{
			{Name: "Signal peptide", Start: 1, End: 24, Direction: 1, Color: lipgloss.Color("81")},
			{Name: "Binding region", Start: 42, End: 120, Direction: 0, Color: lipgloss.Color("205")},
			{Name: "Cys cluster", Start: 120, End: 210, Direction: 0, Color: lipgloss.Color("42")},
			{Name: "Flexible tail", Start: 260, End: 360, Direction: -1, Color: lipgloss.Color("214")},
		}),
	)
}

func newDemoModel() demoModel {
	return demoModel{
		viewer: newDNAViewer(),
		mode:   "DNA",
	}
}

func (m demoModel) Init() tea.Cmd {
	return nil
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.lastSize = msg
	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.viewer = newDNAViewer()
			m.mode = "DNA"
			if m.lastSize.Width > 0 {
				updated, cmd := m.viewer.Update(m.lastSize)
				m.viewer = updated.(sequenceviewer.Model)
				return m, cmd
			}
			return m, nil
		case "2":
			m.viewer = newProteinViewer()
			m.mode = "Protein"
			if m.lastSize.Width > 0 {
				updated, cmd := m.viewer.Update(m.lastSize)
				m.viewer = updated.(sequenceviewer.Model)
				return m, cmd
			}
			return m, nil
		}
	}

	updated, cmd := m.viewer.Update(msg)
	m.viewer = updated.(sequenceviewer.Model)
	return m, cmd
}

func (m demoModel) View() tea.View {
	header := fmt.Sprintf(
		"SequenceViewer demo | mode: %s | 1 DNA | 2 Protein | Arrows move | Shift+Arrows select | [ ] feature | Tab lens | ? help+legend | q quit",
		m.mode,
	)
	return tea.NewView(header + "\n\n" + m.viewer.Render())
}

func main() {
	program := tea.NewProgram(newDemoModel())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
