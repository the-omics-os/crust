package sequenceviewer

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func sampleDNA() string {
	return "ATGCGATCGATCGATCGATCGATCGATCGATCGATCGATCGATCG"
}

func sampleProtein() string {
	return "MKWVTFISLLFLFSSAYSRGVFRRDTHKSEIAHRFKDLGE"
}

func TestNewDefaults(t *testing.T) {
	m := New()
	if m.Type() != SequenceUnknown {
		t.Fatalf("expected unknown type, got %v", m.Type())
	}
	if m.width != defaultWidth {
		t.Fatalf("expected default width %d, got %d", defaultWidth, m.width)
	}
	if m.gcWindow != defaultGCWindow {
		t.Fatalf("expected default gc window %d, got %d", defaultGCWindow, m.gcWindow)
	}
	if m.ViewMode() != IdentityView {
		t.Fatalf("expected identity view, got %v", m.ViewMode())
	}
}

func TestWithSequenceDNA(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA))
	if m.Type() != DNA {
		t.Fatalf("expected DNA type, got %v", m.Type())
	}
	if m.Length() != len(sampleDNA()) {
		t.Fatalf("expected length %d, got %d", len(sampleDNA()), m.Length())
	}
	if m.GCContent() <= 0 {
		t.Fatal("expected positive GC content")
	}
}

func TestViewCyclingDNA(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got := updated.(Model)
	if got.ViewMode() != GCContentView {
		t.Fatalf("expected GC content view, got %v", got.ViewMode())
	}
	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	got = updated.(Model)
	if got.ViewMode() != IdentityView {
		t.Fatalf("expected identity view after cycling, got %v", got.ViewMode())
	}
}

func TestViewCyclingProtein(t *testing.T) {
	m := New(WithSequence(sampleProtein(), Protein))
	expected := []ViewMode{HydrophobicityView, ChargeView, MolWeightView, IdentityView}
	current := m
	for _, want := range expected {
		updated, _ := current.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		current = updated.(Model)
		if current.ViewMode() != want {
			t.Fatalf("expected %v, got %v", want, current.ViewMode())
		}
	}
}

func TestComplementToggleDNA(t *testing.T) {
	m := New(WithSequence("ATGCGT", DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "c", Code: 'c'})
	got := updated.(Model)
	plain := stripANSI(got.Render())
	if !strings.Contains(plain, "TAC GCA") {
		t.Fatalf("expected complement strand in render, got:\n%s", plain)
	}
}

func TestComplementToggleProteinNoop(t *testing.T) {
	m := New(WithSequence(sampleProtein(), Protein))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "c", Code: 'c'})
	got := updated.(Model)
	if got.showComplement {
		t.Fatal("protein viewer should not enable complement")
	}
}

func TestRenderVariousWidths(t *testing.T) {
	m := New(
		WithSequence(sampleDNA(), DNA),
		WithAnnotations([]Annotation{{Name: "FeatureA", Start: 2, End: 10, Direction: 1}}),
		WithComplement(true),
	)
	for _, width := range []int{40, 80, 120} {
		m.SetWidth(width)
		rendered := stripANSI(m.Render())
		if rendered == "" {
			t.Fatalf("empty render at width %d", width)
		}
		if !strings.Contains(rendered, "DNA Sequence") {
			t.Fatalf("expected header at width %d, got:\n%s", width, rendered)
		}
		if !strings.Contains(rendered, "Left/Right residue") {
			t.Fatalf("expected footer at width %d, got:\n%s", width, rendered)
		}
	}
}

func TestLeftRightMovesFocus(t *testing.T) {
	m := New(WithSequence("ATGC", DNA))
	if m.FocusPosition() != 1 {
		t.Fatalf("expected initial focus at 1, got %d", m.FocusPosition())
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	got := updated.(Model)
	if got.FocusPosition() != 2 {
		t.Fatalf("expected focus at 2 after right, got %d", got.FocusPosition())
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	got = updated.(Model)
	if got.FocusPosition() != 1 {
		t.Fatalf("expected focus back at 1 after left, got %d", got.FocusPosition())
	}
}

func TestShiftRightSelectsRange(t *testing.T) {
	m := New(WithSequence("ATGC", DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got := updated.(Model)

	start, end, ok := got.SelectionRange()
	if !ok {
		t.Fatal("expected active selection after shift+right")
	}
	if start != 1 || end != 2 {
		t.Fatalf("expected selection 1-2, got %d-%d", start, end)
	}
	if got.FocusPosition() != 2 {
		t.Fatalf("expected focus at 2 after shift+right, got %d", got.FocusPosition())
	}
}

func TestPlainMoveClearsSelection(t *testing.T) {
	m := New(WithSequence("ATGC", DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got := updated.(Model)

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	got = updated.(Model)
	if _, _, ok := got.SelectionRange(); ok {
		t.Fatal("expected plain movement to clear selection")
	}
	if got.FocusPosition() != 3 {
		t.Fatalf("expected focus at 3 after plain right, got %d", got.FocusPosition())
	}
}

func TestShiftDownExtendsSelectionByRow(t *testing.T) {
	m := New(
		WithSequence(strings.Repeat("ATGC", 20), DNA),
		WithResiduesPerLine(8),
	)
	m.SetFocus(9)

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift})
	got := updated.(Model)

	start, end, ok := got.SelectionRange()
	if !ok {
		t.Fatal("expected active selection after shift+down")
	}
	if start != 9 || end != 17 {
		t.Fatalf("expected selection 9-17 after shift+down, got %d-%d", start, end)
	}
	if got.FocusPosition() != 17 {
		t.Fatalf("expected focus at 17 after shift+down, got %d", got.FocusPosition())
	}
}

func TestEscClearsSelection(t *testing.T) {
	m := New(WithSequence("ATGC", DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got := updated.(Model)

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if _, _, ok := got.SelectionRange(); ok {
		t.Fatal("expected esc to clear selection")
	}
	if got.FocusPosition() != 2 {
		t.Fatalf("expected focus to remain at 2 after clearing selection, got %d", got.FocusPosition())
	}
}

func TestUpDownMovesByRow(t *testing.T) {
	m := New(
		WithSequence(strings.Repeat("ATGC", 20), DNA),
		WithResiduesPerLine(8),
	)
	m.SetFocus(9)
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	got := updated.(Model)
	if got.FocusPosition() != 17 {
		t.Fatalf("expected focus at 17 after down, got %d", got.FocusPosition())
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	got = updated.(Model)
	if got.FocusPosition() != 9 {
		t.Fatalf("expected focus back at 9 after up, got %d", got.FocusPosition())
	}
}

func TestAutoExpandResiduesPerLineUsesAvailableWidth(t *testing.T) {
	m := New(
		WithSequence(strings.Repeat("ATGC", 20), DNA),
		WithWidth(40),
	)
	narrow := m.effectiveResiduesPerLine()

	m.SetWidth(120)
	wide := m.effectiveResiduesPerLine()

	if wide <= narrow {
		t.Fatalf("expected wider view to fit more residues, got narrow=%d wide=%d", narrow, wide)
	}
}

func TestExplicitResiduesPerLinePinsLayout(t *testing.T) {
	m := New(
		WithSequence(strings.Repeat("ATGC", 20), DNA),
		WithWidth(120),
		WithResiduesPerLine(8),
	)
	if got := m.effectiveResiduesPerLine(); got != 8 {
		t.Fatalf("expected explicit residues per line to stay pinned at 8, got %d", got)
	}
}

func TestFeatureJumpMovesFocus(t *testing.T) {
	m := New(
		WithSequence(strings.Repeat("ATGC", 20), DNA),
		WithAnnotations([]Annotation{
			{Name: "A", Start: 10, End: 20},
			{Name: "B", Start: 40, End: 50},
		}),
	)
	updated, _ := m.Update(tea.KeyPressMsg{Text: "]", Code: ']'})
	got := updated.(Model)
	if got.FocusPosition() != 10 {
		t.Fatalf("expected jump to first feature at 10, got %d", got.FocusPosition())
	}

	updated, _ = got.Update(tea.KeyPressMsg{Text: "]", Code: ']'})
	got = updated.(Model)
	if got.FocusPosition() != 40 {
		t.Fatalf("expected jump to second feature at 40, got %d", got.FocusPosition())
	}
}

func TestSetSequenceSwitchesTypeAndView(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA), WithView(GCContentView))
	m.SetSequence(sampleProtein(), Protein)
	if m.Type() != Protein {
		t.Fatalf("expected protein type, got %v", m.Type())
	}
	if m.ViewMode() != IdentityView {
		t.Fatalf("expected identity view after switching to protein, got %v", m.ViewMode())
	}
	if m.IsoelectricPoint() <= 0 {
		t.Fatal("expected positive pI for protein sequence")
	}
}

func TestResidueProperties(t *testing.T) {
	dna := New(WithSequence("ATGCGC", DNA), WithGCWindow(3))
	if dna.Residues()[2].Properties.GCWindow <= 0 {
		t.Fatal("expected GC window to be computed for DNA")
	}

	protein := New(WithSequence("ILKR", Protein))
	residues := protein.Residues()
	if residues[0].Properties.Hydrophobicity != 4.5 {
		t.Fatalf("expected isoleucine hydrophobicity 4.5, got %f", residues[0].Properties.Hydrophobicity)
	}
	if residues[2].Properties.Charge <= 0 {
		t.Fatalf("expected lysine positive charge, got %f", residues[2].Properties.Charge)
	}
}

func TestSetResiduesDefensiveCopy(t *testing.T) {
	residues := []Residue{
		{Position: 1, Code: 'A'},
		{Position: 2, Code: 'T'},
	}
	m := New(WithResidues(residues))
	residues[0].Code = 'Z'
	if m.Sequence() != "AT" {
		t.Fatalf("expected internal residue copy to remain unchanged, got %q", m.Sequence())
	}
}

func TestHelpToggle(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "?", Code: '?'})
	got := updated.(Model)
	rendered := stripANSI(got.Render())
	if !strings.Contains(rendered, "Help") {
		t.Fatalf("expected help block in render, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "highlighted residue is the focus") {
		t.Fatalf("expected focus guidance in help, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Legend") {
		t.Fatalf("expected legend block in help, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Features: > forward start") {
		t.Fatalf("expected feature symbol legend in help, got:\n%s", rendered)
	}
}

func TestHelpLegendIncludesPropertyRampOutsideIdentityView(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA), WithView(GCContentView))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "?", Code: '?'})
	got := updated.(Model)
	rendered := stripANSI(got.Render())
	if !strings.Contains(rendered, "Property bar:") {
		t.Fatalf("expected property legend in help for property view, got:\n%s", rendered)
	}
}

func TestWindowResize(t *testing.T) {
	m := New(WithSequence(sampleDNA(), DNA))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 96, Height: 24})
	got := updated.(Model)
	if got.width != 96 {
		t.Fatalf("expected width 96, got %d", got.width)
	}
	if got.height != 24 {
		t.Fatalf("expected height 24, got %d", got.height)
	}
}
