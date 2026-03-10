package platemap

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

func samplePlate() PlateData {
	return PlateData{
		Format: Plate96,
		Title:  "Primary Screen Plate",
		Metadata: map[string]string{
			"assay": "olink",
			"batch": "B-17",
		},
		Wells: []Well{
			{Row: 0, Col: 0, Signal: 1.90, Normalized: 1.10, ZScore: 2.40, Control: ControlPositive, SampleID: "POS-1"},
			{Row: 0, Col: 11, Signal: 0.12, Normalized: 0.15, ZScore: -2.10, Control: ControlNegative, SampleID: "NEG-1"},
			{Row: 1, Col: 2, Signal: 0.42, Normalized: 0.81, ZScore: -2.30, Control: ControlSample, SampleID: "BRCA1-01", Reagent: "BRCA1-siRNA", Hit: true},
			{Row: 1, Col: 3, Signal: 0.39, Normalized: 0.77, ZScore: -2.05, Control: ControlSample, SampleID: "BRCA1-02", Reagent: "BRCA1-siRNA"},
			{Row: 2, Col: 2, Signal: 0.36, Normalized: 0.74, ZScore: -1.80, Control: ControlSample, SampleID: "BRCA1-03", Reagent: "BRCA1-siRNA"},
			{Row: 3, Col: 5, Signal: 0.00, Normalized: 0.00, ZScore: 0.00, Control: ControlEmpty},
			{Row: 4, Col: 1, Signal: 0.00, Normalized: 0.00, ZScore: 0.00, Control: ControlSample, SampleID: "MISSING-01", Missing: true},
			{Row: 7, Col: 11, Signal: 1.03, Normalized: 1.02, ZScore: 1.40, Control: ControlSample, SampleID: "TP53-01", Reagent: "TP53-siRNA"},
		},
	}
}

func TestNewDefaults(t *testing.T) {
	m := New()
	if m.plate.Format != Plate96 {
		t.Fatalf("expected default format %v, got %v", Plate96, m.plate.Format)
	}
	if m.mode != ViewRawSignal {
		t.Fatalf("expected default mode %v, got %v", ViewRawSignal, m.mode)
	}
	if m.width != 80 || m.height != 18 {
		t.Fatalf("expected default size 80x18, got %dx%d", m.width, m.height)
	}
	if m.selectedRow != -1 || m.selectedCol != -1 {
		t.Fatalf("expected no active selection, got row=%d col=%d", m.selectedRow, m.selectedCol)
	}
}

func TestWithPlateDefensiveCopy(t *testing.T) {
	plate := samplePlate()
	m := New(WithPlate(plate))

	plate.Wells[0].SampleID = "MUTATED"
	plate.Metadata["assay"] = "changed"

	got := m.Plate()
	if got.Wells[0].SampleID == "MUTATED" {
		t.Fatal("WithPlate did not defensively copy wells")
	}
	if got.Metadata["assay"] == "changed" {
		t.Fatal("WithPlate did not defensively copy metadata")
	}
}

func TestPlateReturnsCopy(t *testing.T) {
	m := New(WithPlate(samplePlate()))
	got := m.Plate()
	got.Wells[0].SampleID = "MUTATED"
	got.Metadata["batch"] = "changed"

	again := m.Plate()
	if again.Wells[0].SampleID == "MUTATED" {
		t.Fatal("Plate() returned a shared well slice")
	}
	if again.Metadata["batch"] == "changed" {
		t.Fatal("Plate() returned shared metadata")
	}
}

func TestSetPlateDefensiveCopy(t *testing.T) {
	plate := samplePlate()
	m := New()
	m.SetPlate(plate)

	plate.Wells[1].SampleID = "MUTATED"
	plate.Metadata["batch"] = "changed"

	got := m.Plate()
	if got.Wells[1].SampleID == "MUTATED" {
		t.Fatal("SetPlate did not defensively copy wells")
	}
	if got.Metadata["batch"] == "changed" {
		t.Fatal("SetPlate did not defensively copy metadata")
	}
}

func TestUpsertWellReplacesAndUpsizesFormat(t *testing.T) {
	m := New(WithPlate(samplePlate()))
	m.UpsertWell(Well{Row: 1, Col: 2, Signal: 9.9, Normalized: 2.1, ZScore: 6.7, Control: ControlSample, SampleID: "BRCA1-01", Reagent: "BRCA1-siRNA"})

	well, ok := m.wellAt(1, 2)
	if !ok {
		t.Fatal("expected B3 well to exist after upsert")
	}
	if well.Signal != 9.9 {
		t.Fatalf("expected updated signal 9.9, got %.2f", well.Signal)
	}

	m.UpsertWell(Well{Row: 20, Col: 30, Signal: 1.2, Normalized: 1.1, ZScore: 0.2, SampleID: "ROW20", Control: ControlSample})
	if m.plate.Format != Plate1536 {
		t.Fatalf("expected format to upsize to 1536, got %v", m.plate.Format)
	}
	if _, ok := m.wellAt(20, 30); !ok {
		t.Fatal("expected streamed well at U31 to exist")
	}
}

func TestArrowNavigationAndSelection(t *testing.T) {
	m := New(WithPlate(samplePlate()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	got := updated.(Model)
	if got.cursorCol != 1 {
		t.Fatalf("expected cursor col 1, got %d", got.cursorCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	got = updated.(Model)
	if got.cursorRow != 1 || got.selectedRow != 1 || got.selectedCol != -1 {
		t.Fatalf("expected row selection on row 1, got cursor=(%d,%d) selectedRow=%d selectedCol=%d", got.cursorRow, got.cursorCol, got.selectedRow, got.selectedCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got = updated.(Model)
	if got.cursorCol != 2 || got.selectedCol != 2 || got.selectedRow != -1 {
		t.Fatalf("expected column selection on col 2, got cursor=(%d,%d) selectedRow=%d selectedCol=%d", got.cursorRow, got.cursorCol, got.selectedRow, got.selectedCol)
	}
}

func TestIntuitiveSweepAndInspectControls(t *testing.T) {
	m := New(WithPlate(samplePlate()), WithCursor(1, 2))

	updated, _ := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	got := updated.(Model)
	if got.selectedRow != 1 || got.selectedCol != -1 {
		t.Fatalf("expected row sweep on row 1, got row=%d col=%d", got.selectedRow, got.selectedCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	got = updated.(Model)
	if got.selectedRow != -1 || got.selectedCol != -1 {
		t.Fatalf("expected row sweep toggle off, got row=%d col=%d", got.selectedRow, got.selectedCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: 'c', Text: "c"})
	got = updated.(Model)
	if got.selectedCol != 2 || got.selectedRow != -1 {
		t.Fatalf("expected column sweep on col 2, got row=%d col=%d", got.selectedRow, got.selectedCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got = updated.(Model)
	if !got.inspectorVisible {
		t.Fatal("expected inspector to open on space")
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: 'i', Text: "i"})
	got = updated.(Model)
	if got.inspectorVisible {
		t.Fatal("expected inspector to toggle closed on i")
	}
}

func TestTabCyclesModes(t *testing.T) {
	m := New()

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := updated.(Model)
	if got.mode != ViewNormalized {
		t.Fatalf("expected normalized mode after tab, got %v", got.mode)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	got = updated.(Model)
	if got.mode != ViewRawSignal {
		t.Fatalf("expected raw signal mode after shift+tab, got %v", got.mode)
	}
}

func TestHomeEndAndPageNavigation(t *testing.T) {
	m := New(
		WithFormat(Plate384),
		WithCursor(8, 10),
		WithWidth(60),
		WithHeight(12),
	)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyHome})
	got := updated.(Model)
	if got.cursorCol != 0 {
		t.Fatalf("expected home to move to first column, got %d", got.cursorCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEnd})
	got = updated.(Model)
	if got.cursorCol != 23 {
		t.Fatalf("expected end to move to last column, got %d", got.cursorCol)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyPgUp})
	got = updated.(Model)
	if got.cursorRow >= 8 {
		t.Fatalf("expected pgup to move upward, got row %d", got.cursorRow)
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyPgDown})
	got = updated.(Model)
	if got.cursorRow <= 0 {
		t.Fatalf("expected pgdown to move downward, got row %d", got.cursorRow)
	}
}

func TestEnterEmitsSubmitMsgAndExpandsDetail(t *testing.T) {
	m := New(WithPlate(samplePlate()), WithCursor(1, 2))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd on enter")
	}

	got := updated.(Model)
	if !got.inspectorVisible {
		t.Fatal("expected inspector to open on enter")
	}

	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != plateMapComponent {
		t.Fatalf("expected component %q, got %q", plateMapComponent, submit.Component)
	}
	if submit.Data["coordinate"] != "B3" {
		t.Fatalf("expected coordinate B3, got %v", submit.Data["coordinate"])
	}
	if submit.Data["present"] != true {
		t.Fatalf("expected present=true, got %v", submit.Data["present"])
	}
}

func TestEscClearsContextBeforeCancel(t *testing.T) {
	m := New(WithPlate(samplePlate()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	got := updated.(Model)
	if !got.helpVisible {
		t.Fatal("expected help to be visible after ?")
	}

	updated, cmd := got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if cmd != nil {
		t.Fatal("expected esc to only close help first")
	}
	if got.helpVisible {
		t.Fatal("expected help to close on esc")
	}

	updated, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got = updated.(Model)
	if cmd == nil || !got.inspectorVisible {
		t.Fatal("expected enter to open inspector and emit submit")
	}

	updated, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if cmd != nil {
		t.Fatal("expected esc to close inspector without canceling")
	}
	if got.inspectorVisible {
		t.Fatal("expected inspector to close on esc")
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got = updated.(Model)
	if got.selectedCol != 1 {
		t.Fatalf("expected selected column 1, got %d", got.selectedCol)
	}

	updated, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if cmd != nil {
		t.Fatal("expected esc to clear selection before canceling")
	}
	if got.selectedCol != -1 || got.selectedRow != -1 {
		t.Fatal("expected selection to be cleared on esc")
	}

	_, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel cmd after esc with no active context")
	}
	cancel, ok := cmd().(crust.CancelMsg)
	if !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", cmd())
	}
	if cancel.Component != plateMapComponent {
		t.Fatalf("expected component %q, got %q", plateMapComponent, cancel.Component)
	}
}

func TestRenderContainsSummaryHelpAndReplicates(t *testing.T) {
	m := New(
		WithPlate(samplePlate()),
		WithCursor(1, 2),
		WithWidth(72),
		WithHeight(14),
	)

	view := m.Render()
	for _, want := range []string{"Primary Screen Plate", "Focus: B3", "Lenses:", "Legend:", "Raw Signal view", "Always:", "positive ctrl", "Signal:", "lowest", "highest", "space/i", "inspect"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected render to contain %q", want)
		}
	}
	for _, unwanted := range []string{"┌", "└"} {
		if strings.Contains(view, unwanted) {
			t.Fatalf("expected collapsed lower band to stay single-line per segment, found %q", unwanted)
		}
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	view = updated.(Model).Render()
	if !strings.Contains(view, "toggle row sweep") {
		t.Fatal("expected help text to render when help is visible")
	}

	updated, _ = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeySpace})
	view = updated.(Model).Render()
	for _, want := range []string{"Replicates", "BRCA1-siRNA", "Mean Z", "Metadata", "assay", "B-17"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected inspector to contain %q", want)
		}
	}
	if strings.Contains(view, "┘ ┌") {
		t.Fatal("expected inspector metric cards to be laid out as blocks, not concatenated raw strings")
	}
}

func TestLegendTracksActiveLens(t *testing.T) {
	m := New(
		WithPlate(samplePlate()),
		WithCursor(1, 2),
		WithWidth(72),
		WithHeight(16),
		WithViewMode(ViewZScore),
	)

	view := m.Render()
	for _, want := range []string{"Z-Score view", "Magnitude:", "near 0", "extreme", "Color:", "negative", "positive"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected z-score legend to contain %q", want)
		}
	}

	m.SetViewMode(ViewHitClass)
	view = m.Render()
	for _, want := range []string{"Hit Class view", "Samples:", "#", "hit", "non-hit sample"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected hit-class legend to contain %q", want)
		}
	}
}

func TestRenderAcrossFormats(t *testing.T) {
	tests := []struct {
		format PlateFormat
		row    int
		col    int
		want   string
	}{
		{format: Plate96, row: 7, col: 11, want: "H12"},
		{format: Plate384, row: 15, col: 23, want: "P24"},
		{format: Plate1536, row: 31, col: 47, want: "AF48"},
	}

	for _, tt := range tests {
		m := New(
			WithFormat(tt.format),
			WithCursor(tt.row, tt.col),
			WithWidth(60),
			WithHeight(12),
			WithTitle(tt.format.String()),
		)
		view := m.Render()
		if view == "" {
			t.Fatalf("empty view for format %v", tt.format)
		}
		if !strings.Contains(view, tt.want) {
			t.Fatalf("expected render for format %v to contain %q", tt.format, tt.want)
		}
	}
}

func TestInitWindowResizeAndTheme(t *testing.T) {
	theme := DefaultTheme()
	theme.Hit = lipgloss.Color("196")

	m := New(WithTheme(theme))
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected nil init cmd")
	}
	if m.theme.Hit != lipgloss.Color("196") {
		t.Fatal("expected custom theme to be applied")
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 101, Height: 23})
	got := updated.(Model)
	if got.Width() != 101 || got.Height() != 23 {
		t.Fatalf("expected resized model 101x23, got %dx%d", got.Width(), got.Height())
	}
}
