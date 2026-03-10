package periodictable

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

func TestNewDefaults(t *testing.T) {
	model := New()
	if model.width != 118 {
		t.Fatalf("expected default width 118, got %d", model.width)
	}
	selected := model.Selected()
	if selected == nil || selected.Symbol != "H" {
		t.Fatalf("expected default selection to be H, got %#v", selected)
	}
}

func TestWithSelected(t *testing.T) {
	model := New(WithSelected("Fe"))
	selected := model.Selected()
	if selected == nil || selected.Symbol != "Fe" {
		t.Fatalf("expected Fe to be selected, got %#v", selected)
	}
}

func TestSelectedReturnsCopy(t *testing.T) {
	model := New(WithSelected("C"))
	selected := model.Selected()
	if selected == nil {
		t.Fatal("expected a selected element")
	}
	selected.Name = "Mutated"

	again := model.Selected()
	if again.Name == "Mutated" {
		t.Fatal("Selected returned internal state instead of a copy")
	}
}

func TestHorizontalNavigationSkipsGaps(t *testing.T) {
	model := New()
	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	got := updated.(Model).Selected()
	if got == nil || got.Symbol != "He" {
		t.Fatalf("expected right from H to land on He, got %#v", got)
	}
}

func TestVerticalNavigation(t *testing.T) {
	model := New(WithSelected("Fe"))
	updated, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	got := updated.(Model).Selected()
	if got == nil || got.Symbol != "Ru" {
		t.Fatalf("expected down from Fe to land on Ru, got %#v", got)
	}
}

func TestJumpToPeriod(t *testing.T) {
	model := New(WithSelected("H"))
	updated, _ := model.Update(tea.KeyPressMsg{Text: "4"})
	got := updated.(Model).Selected()
	if got == nil || got.Symbol != "K" {
		t.Fatalf("expected jump to period 4 to land on K, got %#v", got)
	}
}

func TestTabCyclesModes(t *testing.T) {
	model := New()
	next, _ := model.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if next.(Model).mode != viewModeMass {
		t.Fatalf("expected first tab to switch to mass view, got %v", next.(Model).mode)
	}
}

func TestEnterReturnsSubmitMsg(t *testing.T) {
	model := New(WithSelected("Fe"))
	_, cmd := model.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}

	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != componentKey {
		t.Fatalf("expected component %q, got %q", componentKey, submit.Component)
	}
	if submit.Data["symbol"] != "Fe" {
		t.Fatalf("expected selected symbol Fe, got %#v", submit.Data["symbol"])
	}
}

func TestEscClosesHelpBeforeCancelling(t *testing.T) {
	model := New()
	updated, _ := model.Update(tea.KeyPressMsg{Text: "?"})
	if !updated.(Model).showHelp {
		t.Fatal("expected help to be enabled")
	}

	updated, cmd := updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if updated.(Model).showHelp {
		t.Fatal("expected esc to close help")
	}
	if cmd != nil {
		t.Fatal("did not expect cancel cmd while help was open")
	}

	_, cmd = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel cmd after help is closed")
	}
	if _, ok := cmd().(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", cmd())
	}
}

func TestWindowSizeMsgSetsWidth(t *testing.T) {
	model := New()
	updated, _ := model.Update(tea.WindowSizeMsg{Width: 132})
	if updated.(Model).width != 132 {
		t.Fatalf("expected width 132, got %d", updated.(Model).width)
	}
}

func TestRenderContainsFocusedElementDetails(t *testing.T) {
	model := New(WithSelected("Fe"), WithHighlights("C", "N", "O", "S"))
	output := model.Render()
	for _, want := range []string{"Periodic Table", "Iron", "Atomic #26", "Config:", "Highlights: 4"} {
		if !strings.Contains(output, want) {
			t.Fatalf("expected render output to contain %q", want)
		}
	}
}

func TestElementDatasetHasAllElements(t *testing.T) {
	if len(allElements) != 118 {
		t.Fatalf("expected 118 elements, got %d", len(allElements))
	}
	iron, ok := elementBySymbol("Fe")
	if !ok {
		t.Fatal("expected to find iron")
	}
	if iron.Period != 4 || iron.Group != 8 {
		t.Fatalf("unexpected iron position: period=%d group=%d", iron.Period, iron.Group)
	}
	if iron.ElectronConfig == "" || iron.AtomicMass == 0 {
		t.Fatalf("expected populated iron properties, got %#v", iron)
	}
}
