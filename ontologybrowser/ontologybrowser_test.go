package ontologybrowser

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

func sampleRoots() []OntologyNode {
	return []OntologyNode{
		{
			ID:          "GO:0008150",
			Name:        "biological_process",
			Description: "Processes relevant to the functioning of integrated living units.",
		},
		{
			ID:          "GO:0003674",
			Name:        "molecular_function",
			Description: "Activities at the molecular level.",
			Loaded:      true,
			Children: []OntologyNode{
				{
					ID:          "GO:0005488",
					Name:        "binding",
					Description: "Selective, non-covalent, often stoichiometric interaction.",
					Loaded:      true,
				},
				{
					ID:          "GO:0003824",
					Name:        "catalytic activity",
					Description: "Catalysis of a biochemical reaction.",
					Loaded:      true,
				},
			},
		},
	}
}

func keyText(text string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Text: text, Code: []rune(text)[0]}
}

func typeQuery(t *testing.T, m Model, query string) Model {
	t.Helper()

	for _, r := range query {
		updated, _ := m.Update(tea.KeyPressMsg{Text: string(r), Code: r})
		m = updated.(Model)
	}
	return m
}

func stripANSI(s string) string {
	ansiRE := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return ansiRE.ReplaceAllString(s, "")
}

func TestNewDefaults(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	if m.Width() != defaultWidth {
		t.Fatalf("expected default width %d, got %d", defaultWidth, m.Width())
	}
	if m.Height() != defaultHeight {
		t.Fatalf("expected default height %d, got %d", defaultHeight, m.Height())
	}

	selected := m.Selected()
	if selected == nil || selected.ID != "GO:0008150" {
		t.Fatalf("expected first root selected, got %+v", selected)
	}
}

func TestExpandUnloadedNodeEmitsExpandMsg(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("expected expand cmd")
	}

	msg := cmd()
	expand, ok := msg.(ExpandMsg)
	if !ok {
		t.Fatalf("expected ExpandMsg, got %T", msg)
	}
	if expand.NodeID != "GO:0008150" {
		t.Fatalf("expected expand node GO:0008150, got %q", expand.NodeID)
	}

	got := updated.(Model)
	if !got.expanded["GO:0008150"] {
		t.Fatal("expected node marked expanded")
	}
	if !got.loading["GO:0008150"] {
		t.Fatal("expected node marked loading")
	}
}

func TestEnterSelectsCurrentBranchInsteadOfExpanding(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd on enter")
	}

	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Data["id"] != "GO:0003674" {
		t.Fatalf("expected selected branch GO:0003674, got %v", submit.Data["id"])
	}

	got := updated.(Model)
	if got.expanded["GO:0003674"] {
		t.Fatal("expected enter not to expand the branch")
	}
}

func TestTypingStartsFilterAndRevealsCollapsedLoadedDescendant(t *testing.T) {
	m := New(WithRoots(sampleRoots()))
	m = typeQuery(t, m, "bind")

	if m.activePane != paneSearch {
		t.Fatal("expected typing from tree to enter filter mode")
	}
	if m.searchQuery != "bind" {
		t.Fatalf("expected search query 'bind', got %q", m.searchQuery)
	}
	if len(m.searchResults) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(m.searchResults))
	}
	if !m.expanded["GO:0003674"] {
		t.Fatal("expected matching path to expand automatically")
	}

	selected := m.Selected()
	if selected == nil || selected.ID != "GO:0005488" {
		t.Fatalf("expected binding selected, got %+v", selected)
	}

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd from filtered selection")
	}
	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Data["id"] != "GO:0005488" {
		t.Fatalf("expected binding node selected, got %v", submit.Data["id"])
	}
}

func TestEscClearsFilterThenCancels(t *testing.T) {
	m := New(WithRoots(sampleRoots()))
	m = typeQuery(t, m, "bind")

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatal("did not expect cancel while clearing filter")
	}
	m = updated.(Model)
	if m.searchQuery != "" {
		t.Fatalf("expected filter to clear, got %q", m.searchQuery)
	}
	if m.activePane != paneTree {
		t.Fatal("expected to return to browse mode after clearing filter")
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel cmd after filter is cleared")
	}
	msg := cmd()
	if _, ok := msg.(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", msg)
	}
}

func TestLeftNavigatesToParentAndCollapses(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd != nil {
		t.Fatal("did not expect lazy expand cmd for preloaded branch")
	}
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)
	if selected := m.Selected(); selected == nil || selected.ID != "GO:0005488" {
		t.Fatalf("expected child selected, got %+v", selected)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = updated.(Model)
	if selected := m.Selected(); selected == nil || selected.ID != "GO:0003674" {
		t.Fatalf("expected parent selected, got %+v", selected)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	m = updated.(Model)
	if m.expanded["GO:0003674"] {
		t.Fatal("expected parent collapsed")
	}
}

func TestWithRootsMakesDefensiveCopy(t *testing.T) {
	roots := sampleRoots()
	m := New(WithRoots(roots))

	roots[0].Name = "mutated"
	selected := m.Selected()
	if selected == nil {
		t.Fatal("expected selection")
	}
	if selected.Name == "mutated" {
		t.Fatal("expected WithRoots to make a defensive copy")
	}
}

func TestSelectedReturnsCopy(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	selected := m.Selected()
	if selected == nil {
		t.Fatal("expected selection")
	}
	selected.Name = "mutated"

	selectedAgain := m.Selected()
	if selectedAgain == nil || selectedAgain.Name == "mutated" {
		t.Fatal("expected Selected to return a copy")
	}
}

func TestRenderShowsFilterGuidanceAndHelp(t *testing.T) {
	m := New(WithRoots(sampleRoots()), WithWidth(72), WithHeight(20))

	rendered := m.Render()
	if !strings.Contains(rendered, "Filter>") {
		t.Fatalf("expected render to contain filter prompt, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Type to filter loaded terms") {
		t.Fatalf("expected render to contain filter guidance, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "selected") || !strings.Contains(rendered, "branch") {
		t.Fatalf("expected render to contain legend text, got:\n%s", rendered)
	}

	updated, _ := m.Update(keyText("?"))
	m = updated.(Model)
	rendered = m.Render()
	if !strings.Contains(rendered, "Enter: confirm the currently highlighted term") {
		t.Fatalf("expected help text rendered, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "Legend") {
		t.Fatalf("expected help to contain legend section, got:\n%s", rendered)
	}
}

func TestRenderFitsRequestedHeight(t *testing.T) {
	m := New(
		WithRoots(sampleRoots()),
		WithWidth(96),
		WithHeight(28),
	)

	rendered := stripANSI(m.Render())
	lineCount := strings.Count(rendered, "\n") + 1
	if lineCount > 28 {
		t.Fatalf("expected render to fit height 28, got %d lines:\n%s", lineCount, rendered)
	}
}

func TestDownArrowRefreshesRenderedSelection(t *testing.T) {
	m := New(WithRoots([]OntologyNode{
		{ID: "A", Name: "alpha", Loaded: true},
		{ID: "B", Name: "beta", Loaded: true},
	}))

	before := stripANSI(m.Render())
	if !strings.Contains(before, "› • A alpha") {
		t.Fatalf("expected initial selection marker on A, got:\n%s", before)
	}

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	after := stripANSI(updated.(Model).Render())

	if !strings.Contains(after, "› • B beta") {
		t.Fatalf("expected selection marker to move to B, got:\n%s", after)
	}
	if strings.Contains(after, "› • A alpha") {
		t.Fatalf("expected selection marker to leave A, got:\n%s", after)
	}
}

func TestSetWidthAndHeightClamp(t *testing.T) {
	m := New(WithRoots(sampleRoots()))
	m.SetWidth(10)
	m.SetHeight(8)

	if m.Width() != minWidth {
		t.Fatalf("expected width clamped to %d, got %d", minWidth, m.Width())
	}
	if m.Height() != minHeight {
		t.Fatalf("expected height clamped to %d, got %d", minHeight, m.Height())
	}
}
