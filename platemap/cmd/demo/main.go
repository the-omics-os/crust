package main

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust/platemap"
)

func main() {
	model := demoModel{
		inner: platemap.New(
			platemap.WithPlate(samplePlate()),
			platemap.WithWidth(92),
			platemap.WithHeight(20),
			platemap.WithCursor(1, 2),
		),
	}

	program := tea.NewProgram(model)
	if _, err := program.Run(); err != nil {
		println(err.Error())
		os.Exit(1)
	}
}

type demoModel struct {
	inner platemap.Model
}

func (m demoModel) Init() tea.Cmd {
	return m.inner.Init()
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	updated, cmd := m.inner.Update(msg)
	m.inner = updated.(platemap.Model)
	return m, cmd
}

func (m demoModel) View() tea.View {
	view := m.inner.View()
	view.AltScreen = true
	return view
}

func samplePlate() platemap.PlateData {
	return platemap.PlateData{
		Format: platemap.Plate96,
		Title:  "Primary Screen Plate",
		Metadata: map[string]string{
			"assay": "olink",
			"batch": "B-17",
			"run":   "screen-01",
		},
		Wells: []platemap.Well{
			{Row: 0, Col: 0, Signal: 1.90, Normalized: 1.10, ZScore: 2.40, Control: platemap.ControlPositive, SampleID: "POS-1"},
			{Row: 0, Col: 11, Signal: 0.12, Normalized: 0.15, ZScore: -2.10, Control: platemap.ControlNegative, SampleID: "NEG-1"},
			{Row: 1, Col: 2, Signal: 0.42, Normalized: 0.81, ZScore: -2.30, Control: platemap.ControlSample, SampleID: "BRCA1-01", Reagent: "BRCA1-siRNA", Hit: true},
			{Row: 1, Col: 3, Signal: 0.39, Normalized: 0.77, ZScore: -2.05, Control: platemap.ControlSample, SampleID: "BRCA1-02", Reagent: "BRCA1-siRNA"},
			{Row: 1, Col: 4, Signal: 0.31, Normalized: 0.70, ZScore: -1.88, Control: platemap.ControlSample, SampleID: "BRCA1-03", Reagent: "BRCA1-siRNA"},
			{Row: 2, Col: 7, Signal: 1.35, Normalized: 1.04, ZScore: 1.52, Control: platemap.ControlSample, SampleID: "STAT3-01", Reagent: "STAT3-siRNA"},
			{Row: 3, Col: 5, Signal: 0.00, Normalized: 0.00, ZScore: 0.00, Control: platemap.ControlEmpty},
			{Row: 4, Col: 1, Signal: 0.00, Normalized: 0.00, ZScore: 0.00, Control: platemap.ControlSample, SampleID: "MISSING-01", Missing: true},
			{Row: 5, Col: 9, Signal: 1.58, Normalized: 1.09, ZScore: 2.12, Control: platemap.ControlSample, SampleID: "MYC-01", Reagent: "MYC-siRNA", Hit: true},
			{Row: 7, Col: 11, Signal: 1.03, Normalized: 1.02, ZScore: 1.40, Control: platemap.ControlSample, SampleID: "TP53-01", Reagent: "TP53-siRNA"},
		},
	}
}
