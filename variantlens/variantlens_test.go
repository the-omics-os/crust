package variantlens

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

func sampleContext() VariantContext {
	return VariantContext{
		RefSequence:    "GATTGCGATCCT",
		ReferenceStart: 178,
		ContextSize:    4,
		Variants: []Variant{
			{
				Position:    181,
				Ref:         "T",
				Alt:         "G",
				Type:        "SNV",
				Consequence: "missense",
				HGVS:        "c.181T>G | p.Cys61Gly",
				Gene:        "BRCA1",
				Impact:      "HIGH",
				Evidence:    "ClinVar: Pathogenic | gnomAD: 0.00002 | SIFT: Deleterious",
			},
			{
				Position:    186,
				Ref:         "T",
				Alt:         "C",
				Type:        "SNV",
				Consequence: "missense",
				HGVS:        "c.186T>C | p.Asp62Asn",
				Gene:        "BRCA1",
				Impact:      "MODERATE",
				Evidence:    "ClinVar: Uncertain significance | gnomAD: 0.0011",
			},
		},
		Features: []Feature{
			{Name: "Exon 5", Type: "exon", Start: 178, End: 189},
			{Name: "CDS 5", Type: "CDS", Start: 178, End: 189},
			{Name: "BRCT domain", Type: "domain", Start: 182, End: 188},
			{Name: "Primer A", Type: "primer", Start: 185, End: 189},
		},
	}
}

func keyText(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: text, Code: []rune(text)[0]}
}

func TestNewDefaultsAndOptions(t *testing.T) {
	theme := DefaultTheme()
	theme.Header = lipgloss.Color("45")

	m := New(
		WithContext(sampleContext()),
		WithWidth(72),
		WithTheme(theme),
		WithSelectedVariant(1),
		WithViewMode(ViewHGVS),
	)

	if m.Width() != 72 {
		t.Fatalf("expected width 72, got %d", m.Width())
	}
	if m.ViewMode() != ViewHGVS {
		t.Fatalf("expected HGVS view, got %v", m.ViewMode())
	}
	if m.SelectedIndex() != 1 {
		t.Fatalf("expected selected index 1, got %d", m.SelectedIndex())
	}
	if m.theme.Header != lipgloss.Color("45") {
		t.Fatal("expected custom theme to be applied")
	}
}

func TestContextGetterReturnsCopy(t *testing.T) {
	m := New(WithContext(sampleContext()))

	got := m.Context()
	got.RefSequence = "AAAA"
	got.Variants[0].Gene = "MUTATED"
	got.Features[0].Name = "changed"

	again := m.Context()
	if again.RefSequence == "AAAA" {
		t.Fatal("Context returned internal ref sequence")
	}
	if again.Variants[0].Gene == "MUTATED" {
		t.Fatal("Context returned internal variants")
	}
	if again.Features[0].Name == "changed" {
		t.Fatal("Context returned internal features")
	}
}

func TestSetContextMakesDefensiveCopy(t *testing.T) {
	ctx := sampleContext()
	m := New()
	m.SetContext(ctx)

	ctx.RefSequence = "AAAA"
	ctx.Variants[0].Impact = "LOW"
	ctx.Features[0].Name = "mutated"

	internal := m.Context()
	if internal.RefSequence == "AAAA" {
		t.Fatal("SetContext did not copy the reference sequence")
	}
	if internal.Variants[0].Impact == "LOW" {
		t.Fatal("SetContext did not copy variants")
	}
	if internal.Features[0].Name == "mutated" {
		t.Fatal("SetContext did not copy features")
	}
}

func TestArrowNavigationEmitsVariantChangedMsg(t *testing.T) {
	m := New(WithContext(sampleContext()))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("expected variant changed cmd")
	}

	msg := cmd()
	changed, ok := msg.(VariantChangedMsg)
	if !ok {
		t.Fatalf("expected VariantChangedMsg, got %T", msg)
	}
	if changed.Index != 1 || changed.Variant.Position != 186 {
		t.Fatalf("unexpected changed payload: %+v", changed)
	}

	got := updated.(Model)
	if got.SelectedIndex() != 1 {
		t.Fatalf("expected selected index 1, got %d", got.SelectedIndex())
	}

	updated, _ = got.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if updated.(Model).SelectedIndex() != 0 {
		t.Fatalf("expected to navigate back to index 0, got %d", updated.(Model).SelectedIndex())
	}
}

func TestContextResizeAndClamp(t *testing.T) {
	m := New(WithContext(sampleContext()))

	updated, cmd := m.Update(keyText("="))
	if cmd == nil {
		t.Fatal("expected context size changed cmd")
	}

	msg := cmd()
	sizeMsg, ok := msg.(ContextSizeChangedMsg)
	if !ok {
		t.Fatalf("expected ContextSizeChangedMsg, got %T", msg)
	}
	if sizeMsg.ContextSize != 7 {
		t.Fatalf("expected context size 7, got %d", sizeMsg.ContextSize)
	}

	got := updated.(Model)
	if got.ContextSize() != 7 {
		t.Fatalf("expected stored context size 7, got %d", got.ContextSize())
	}

	updated, _ = got.Update(keyText("l"))
	got = updated.(Model)
	if got.ContextSize() != 10 {
		t.Fatalf("expected compatibility alias l to widen context to 10, got %d", got.ContextSize())
	}

	for i := 0; i < 8; i++ {
		updated, _ = got.Update(keyText("-"))
		got = updated.(Model)
	}
	if got.ContextSize() != minContextSize {
		t.Fatalf("expected context size clamped to %d, got %d", minContextSize, got.ContextSize())
	}
}

func TestTabAndNumericShortcutsChangeView(t *testing.T) {
	m := New(WithContext(sampleContext()))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if cmd == nil {
		t.Fatal("expected view mode change cmd")
	}
	if updated.(Model).ViewMode() != ViewAnnotation {
		t.Fatalf("expected annotation view, got %v", updated.(Model).ViewMode())
	}

	updated, _ = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if updated.(Model).ViewMode() != ViewSummary {
		t.Fatalf("expected summary view after shift+tab, got %v", updated.(Model).ViewMode())
	}

	updated, _ = updated.(Model).Update(keyText("4"))
	if updated.(Model).ViewMode() != ViewEvidence {
		t.Fatalf("expected evidence view from numeric shortcut, got %v", updated.(Model).ViewMode())
	}
}

func TestEnterSubmitsFocusedVariant(t *testing.T) {
	m := New(WithContext(sampleContext()))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd on enter")
	}

	submit, ok := cmd().(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", cmd())
	}
	if submit.Component != "variant_lens" {
		t.Fatalf("expected component variant_lens, got %q", submit.Component)
	}
	if submit.Data["index"] != 0 {
		t.Fatalf("expected selected index 0, got %v", submit.Data["index"])
	}
	if submit.Data["context_size"] != 4 {
		t.Fatalf("expected context size 4, got %v", submit.Data["context_size"])
	}
}

func TestEscClosesHelpThenCancels(t *testing.T) {
	m := New(WithContext(sampleContext()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	got := updated.(Model)
	if !got.HelpVisible() {
		t.Fatal("expected help to be visible")
	}

	updated, cmd := got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	got = updated.(Model)
	if cmd != nil {
		t.Fatal("expected esc to only close help first")
	}
	if got.HelpVisible() {
		t.Fatal("expected help to close")
	}

	_, cmd = got.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel cmd after help is closed")
	}
	if _, ok := cmd().(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", cmd())
	}
}

func TestWindowSizeMsgSetsWidth(t *testing.T) {
	m := New(WithContext(sampleContext()))

	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if cmd != nil {
		t.Fatal("expected nil cmd for window size message")
	}
	if updated.(Model).Width() != 120 {
		t.Fatalf("expected width 120, got %d", updated.(Model).Width())
	}
}

func TestRenderContainsNavigatorTabsAndActionBar(t *testing.T) {
	m := New(WithContext(sampleContext()), WithWidth(88))

	view := m.Render()
	for _, want := range []string{
		"Variants",
		"Lenses",
		"Annotation",
		"Legend",
		"Cys -> Gly",
		"ClinVar: Pathogenic",
		"[#]",
		"Exon 5",
		"Arrows",
		"Enter",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected render to contain %q", want)
		}
	}
}

func TestCompactLensStripStillShowsAllModes(t *testing.T) {
	m := New(WithContext(sampleContext()), WithWidth(40))
	view := m.Render()

	for _, want := range []string{"1:S", "2:A", "3:H", "4:E"} {
		if !strings.Contains(view, want) {
			t.Fatalf("expected compact lens strip to contain %q", want)
		}
	}
}

func TestRenderLinesStayWithinConfiguredWidth(t *testing.T) {
	for _, width := range []int{40, 44, 56, 60, 68, 88} {
		m := New(WithContext(sampleContext()), WithWidth(width))
		view := m.Render()

		for _, line := range strings.Split(view, "\n") {
			if lipgloss.Width(line) > width {
				t.Fatalf("rendered line exceeds width %d: got %d in %q", width, lipgloss.Width(line), line)
			}
		}
	}
}

func TestRenderHelpAndEmptyState(t *testing.T) {
	empty := New()
	if output := empty.Render(); !strings.Contains(output, "No variants loaded.") {
		t.Fatalf("expected empty render to describe missing variants, got %q", output)
	}

	m := New(WithContext(sampleContext()))
	updated, _ := m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	view := updated.(Model).Render()
	if !strings.Contains(view, "VariantLens Help") {
		t.Fatal("expected help box title in render output")
	}
	if !strings.Contains(view, "move to the previous or next variant") {
		t.Fatal("expected updated help instructions in render output")
	}
	if !strings.Contains(view, "Feature glyphs: = exon, # CDS, ~ domain, : motif, + primer.") {
		t.Fatal("expected legend explanation in help output")
	}
}

func TestSettersUpdateState(t *testing.T) {
	m := New(WithContext(sampleContext()))

	m.SetSelectedVariant(1)
	if m.SelectedIndex() != 1 {
		t.Fatalf("expected selected index 1, got %d", m.SelectedIndex())
	}

	m.SetContextSize(100)
	if m.ContextSize() != len(m.Context().RefSequence) {
		t.Fatalf("expected context size to clamp to sequence length, got %d", m.ContextSize())
	}

	m.SetReferenceSequence("AAATTT")
	if m.Context().RefSequence != "AAATTT" {
		t.Fatalf("expected updated reference sequence, got %q", m.Context().RefSequence)
	}

	m.SetReferenceStart(22)
	if m.Context().ReferenceStart != 22 {
		t.Fatalf("expected reference start 22, got %d", m.Context().ReferenceStart)
	}
}

func TestInitReturnsNilCmd(t *testing.T) {
	m := New(WithContext(sampleContext()))
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected nil init cmd")
	}
}
