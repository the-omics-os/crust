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
		if !strings.Contains(rendered, "Tab: view") {
			t.Fatalf("expected footer at width %d, got:\n%s", width, rendered)
		}
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
