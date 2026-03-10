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

func TestEnterEmitsSubmitMsgAndExpandsDetail(t *testing.T) {
	m := New(WithPlate(samplePlate()), WithCursor(1, 2))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd on enter")
	}

	got := updated.(Model)
	if !got.detailExpanded {
		t.Fatal("expected detail footer to expand on enter")
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
	if cmd == nil || !got.detailExpanded {
		t.Fatal("expected enter to expand detail and emit submit")
	}

	updated, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if cmd != nil {
		t.Fatal("expected esc to collapse detail without canceling")
	}
	if got.detailExpanded {
		t.Fatal("expected detail to collapse on esc")
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
	for _, want := range []string{"Primary Screen Plate", "Cursor: B3", "[B3] Hit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected render to contain %q", want)
		}
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	view = updated.(Model).Render()
	if !strings.Contains(view, "Glyphs: + positive ctrl") {
		t.Fatal("expected help text to render when help is visible")
	}

	updated, _ = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	view = updated.(Model).Render()
	if !strings.Contains(view, "Replicates by reagent (BRCA1-siRNA)") {
		t.Fatal("expected replicate summary in expanded footer")
	}
	if !strings.Contains(view, "Metadata: assay=olink, batch=B-17") {
		t.Fatal("expected metadata summary in expanded footer")
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
