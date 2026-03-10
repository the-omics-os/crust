package moleculeviewer

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

func TestNewWithSMILES(t *testing.T) {
	m := New(WithSMILES("CCO"), WithName("Ethanol"))
	if m.Err() != nil {
		t.Fatalf("expected no load error, got %v", m.Err())
	}
	if m.title != "Ethanol" {
		t.Fatalf("expected title Ethanol, got %q", m.title)
	}
	if m.selectedAtom != 0 {
		t.Fatalf("expected initial selection at atom 0, got %d", m.selectedAtom)
	}
}

func TestNavigationUsesCoordinates(t *testing.T) {
	m := New(WithMOL(sampleMolBlock()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = updated.(Model)
	if m.selectedAtom != 1 {
		t.Fatalf("expected focus to move to atom 1, got %d", m.selectedAtom)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = updated.(Model)
	if m.selectedAtom != 2 {
		t.Fatalf("expected focus to move to atom 2, got %d", m.selectedAtom)
	}
}

func TestSearchModeFocusesMatches(t *testing.T) {
	m := New(WithSMILES("CCO"))

	updated, _ := m.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	m = updated.(Model)
	if !m.searching {
		t.Fatal("expected search mode to open")
	}

	for _, key := range []tea.KeyPressMsg{
		{Text: "h", Code: 'h'},
		{Text: "y", Code: 'y'},
		{Text: "d", Code: 'd'},
		{Text: "r", Code: 'r'},
		{Text: "o", Code: 'o'},
		{Text: "x", Code: 'x'},
		{Text: "y", Code: 'y'},
		{Text: "l", Code: 'l'},
	} {
		updated, _ = m.Update(key)
		m = updated.(Model)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	if m.searching {
		t.Fatal("expected search mode to close after enter")
	}
	if m.selectedAtom != 2 {
		t.Fatalf("expected hydroxyl search to focus oxygen atom, got %d", m.selectedAtom)
	}
	if !m.searchMatches[2] {
		t.Fatal("expected focused atom to remain marked as a search match")
	}
}

func TestSubmitAndCancel(t *testing.T) {
	m := New(WithSMILES("CCO"))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != "molecule_viewer" {
		t.Fatalf("expected component molecule_viewer, got %q", submit.Component)
	}
	if submit.Data["atom_label"] != "C1" {
		t.Fatalf("expected initial atom label C1, got %v", submit.Data["atom_label"])
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel command")
	}
	if _, ok := cmd().(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", cmd())
	}
}

func TestRenderIncludesDiagramAndFallback(t *testing.T) {
	m := New(WithMOL(sampleMolBlock()), WithName("Ethanol"), WithWidth(96), WithHeight(24))
	rendered := stripANSI(m.Render())
	for _, want := range []string{"Ethanol", "C1", "Neighbors:"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected render to contain %q", want)
		}
	}

	m.SetWidth(50)
	fallback := stripANSI(m.Render())
	if !strings.Contains(fallback, "Layout: Adjacency view") {
		t.Fatalf("expected narrow render to fall back to adjacency rendering, got:\n%s", fallback)
	}
}

func TestHelpToggle(t *testing.T) {
	m := New(WithSMILES("CCO"))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "?", Code: '?'})
	m = updated.(Model)
	if !m.showHelp {
		t.Fatal("expected help to open")
	}
	help := stripANSI(m.Render())
	if !strings.Contains(help, "Arrow keys navigate the graph") {
		t.Fatalf("expected help text in render output, got:\n%s", help)
	}
}

func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}
