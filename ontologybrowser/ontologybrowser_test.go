package ontologybrowser

import (
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
		updated, cmd := m.Update(tea.KeyPressMsg{Text: string(r), Code: r})
		if cmd != nil {
			cmd()
		}
		m = updated.(Model)
	}
	return m
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

func TestSetChildrenAndLeafSubmit(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("expected expand cmd")
	}
	_ = cmd()
	m = updated.(Model)

	m.SetChildren("GO:0008150", []OntologyNode{
		{
			ID:          "GO:0044237",
			Name:        "cellular metabolic process",
			Description: "Metabolic processes carried out at the cellular level.",
			Loaded:      true,
		},
	})

	if len(m.visible) != 3 {
		t.Fatalf("expected 3 visible nodes after loading children, got %d", len(m.visible))
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)
	updated, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd for loaded leaf")
	}

	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != componentName {
		t.Fatalf("expected component %q, got %q", componentName, submit.Component)
	}
	if submit.Data["id"] != "GO:0044237" {
		t.Fatalf("expected selected child id, got %v", submit.Data["id"])
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

func TestSearchFocusAndSubmit(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = updated.(Model)

	updated, _ = m.Update(keyText("/"))
	m = updated.(Model)

	m = typeQuery(t, m, "bind")

	if m.activePane != paneSearch {
		t.Fatal("expected search pane active")
	}
	if len(m.searchResults) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(m.searchResults))
	}
	if selected := m.Selected(); selected == nil || selected.ID != "GO:0005488" {
		t.Fatalf("expected search to move selection to GO:0005488, got %+v", selected)
	}

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit cmd from search result")
	}

	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Data["id"] != "GO:0005488" {
		t.Fatalf("expected binding node selected, got %v", submit.Data["id"])
	}
	_ = updated
}

func TestEscLeavesSearchThenCancels(t *testing.T) {
	m := New(WithRoots(sampleRoots()))

	updated, _ := m.Update(keyText("/"))
	m = updated.(Model)

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd != nil {
		t.Fatal("did not expect cancel while leaving search")
	}
	m = updated.(Model)
	if m.activePane != paneTree {
		t.Fatal("expected focus to return to tree")
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel cmd from tree pane")
	}
	msg := cmd()
	if _, ok := msg.(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", msg)
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

func TestRenderShowsHelpAndSearchGuidance(t *testing.T) {
	m := New(WithRoots(sampleRoots()), WithWidth(72), WithHeight(20))

	rendered := m.Render()
	if !strings.Contains(rendered, "Ontology Browser") {
		t.Fatalf("expected render to contain title, got:\n%s", rendered)
	}
	if !strings.Contains(rendered, "visible nodes only") {
		t.Fatalf("expected render to contain search guidance, got:\n%s", rendered)
	}

	updated, _ := m.Update(keyText("?"))
	m = updated.(Model)
	rendered = m.Render()
	if !strings.Contains(rendered, "Right or Enter: expand node") {
		t.Fatalf("expected help text rendered, got:\n%s", rendered)
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
