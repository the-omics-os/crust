package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
	"github.com/the-omics-os/crust/ontologybrowser"
)

type demoModel struct {
	browser ontologybrowser.Model
	done    bool
	result  string
}

func newDemoModel() demoModel {
	return demoModel{
		browser: ontologybrowser.New(
			ontologybrowser.WithRoots([]ontologybrowser.OntologyNode{
				{ID: "GO:0008150", Name: "biological_process", Description: "Processes carried out by integrated living units."},
				{ID: "GO:0003674", Name: "molecular_function", Description: "Activities performed at the molecular level."},
				{ID: "GO:0005575", Name: "cellular_component", Description: "Locations relative to cellular structures."},
			}),
		),
	}
}

func (m demoModel) Init() tea.Cmd {
	return m.browser.Init()
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case ontologybrowser.ExpandMsg:
		m.browser.SetChildren(msg.NodeID, demoChildren(msg.NodeID))
		return m, nil
	case crust.SubmitMsg:
		m.done = true
		m.result = fmt.Sprintf("Selected %s (%s)", msg.Data["name"], msg.Data["id"])
		return m, tea.Quit
	case crust.CancelMsg:
		m.done = true
		m.result = "Cancelled"
		return m, tea.Quit
	}

	updated, cmd := m.browser.Update(msg)
	m.browser = updated.(ontologybrowser.Model)
	return m, cmd
}

func (m demoModel) View() tea.View {
	if m.done {
		return tea.NewView(m.result + "\n")
	}

	return tea.NewView(m.browser.Render())
}

func demoChildren(nodeID string) []ontologybrowser.OntologyNode {
	switch nodeID {
	case "GO:0008150":
		return []ontologybrowser.OntologyNode{
			{ID: "GO:0009987", Name: "cellular process", Description: "Processes carried out at the cellular level."},
			{ID: "GO:0044237", Name: "cellular metabolic process", Description: "Chemical reactions and pathways of individual cells."},
			{ID: "GO:0050896", Name: "response to stimulus", Description: "Changes in state or activity due to a stimulus."},
		}
	case "GO:0044237":
		return []ontologybrowser.OntologyNode{
			{ID: "GO:0044260", Name: "cellular macromolecule metabolic process", Description: "Metabolism of macromolecules within cells.", Loaded: true},
			{ID: "GO:0009117", Name: "nucleotide metabolic process", Description: "Metabolism of nucleotides within cells.", Loaded: true},
		}
	case "GO:0050896":
		return []ontologybrowser.OntologyNode{
			{ID: "GO:0006950", Name: "response to stress", Description: "A response to a stress stimulus.", Loaded: true},
			{ID: "GO:0009605", Name: "response to external stimulus", Description: "A response triggered by factors outside the organism.", Loaded: true},
		}
	case "GO:0003674":
		return []ontologybrowser.OntologyNode{
			{ID: "GO:0005488", Name: "binding", Description: "Selective, non-covalent interaction with a molecule.", Loaded: true},
			{ID: "GO:0003824", Name: "catalytic activity", Description: "Catalysis of a biochemical reaction.", Loaded: true},
			{ID: "GO:0140096", Name: "catalytic activity, acting on a protein", Description: "Catalysis directed at protein substrates.", Loaded: true},
		}
	case "GO:0005575":
		return []ontologybrowser.OntologyNode{
			{ID: "GO:0044464", Name: "cell part", Description: "Any constituent part of a cell.", Loaded: true},
			{ID: "GO:0043226", Name: "organelle", Description: "An organized structure of distinctive morphology and function within a cell.", Loaded: true},
			{ID: "GO:0005623", Name: "cell", Description: "The basic structural and functional unit of organisms.", Loaded: true},
		}
	default:
		return nil
	}
}

func main() {
	program := tea.NewProgram(newDemoModel())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
