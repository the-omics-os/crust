package thresholdslider

import (
	"math"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

func floatEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestNew(t *testing.T) {
	m := New(
		WithLabel("p-value cutoff"),
		WithRange(0, 1),
		WithStep(0.01),
		WithDefault(0.05),
		WithCount(1542, 20000),
	)
	if m.label != "p-value cutoff" {
		t.Fatalf("expected label 'p-value cutoff', got %q", m.label)
	}
	if !floatEq(m.Value(), 0.05) {
		t.Fatalf("expected value 0.05, got %f", m.Value())
	}
}

func TestNewClampsDefault(t *testing.T) {
	m := New(WithRange(0, 1), WithDefault(5.0))
	if !floatEq(m.Value(), 1.0) {
		t.Fatalf("expected clamped value 1.0, got %f", m.Value())
	}
}

func TestAdjustRight(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.50))
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd == nil {
		t.Fatal("expected ValueChangedMsg cmd")
	}
	msg := cmd()
	if _, ok := msg.(ValueChangedMsg); !ok {
		t.Fatalf("expected ValueChangedMsg, got %T", msg)
	}
	got := updated.(Model)
	if !floatEq(got.Value(), 0.51) {
		t.Fatalf("expected 0.51, got %f", got.Value())
	}
}

func TestAdjustLeft(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.50))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	got := updated.(Model)
	if !floatEq(got.Value(), 0.49) {
		t.Fatalf("expected 0.49, got %f", got.Value())
	}
}

func TestCoarseAdjust(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.50))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	got := updated.(Model)
	if !floatEq(got.Value(), 0.60) {
		t.Fatalf("expected 0.60, got %f", got.Value())
	}
}

func TestClampMin(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.0))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	got := updated.(Model)
	if !floatEq(got.Value(), 0.0) {
		t.Fatalf("expected 0.0 (clamped), got %f", got.Value())
	}
}

func TestClampMax(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(1.0))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	got := updated.(Model)
	if !floatEq(got.Value(), 1.0) {
		t.Fatalf("expected 1.0 (clamped), got %f", got.Value())
	}
}

func TestSubmit(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.05))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected SubmitMsg cmd on enter")
	}
	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != "threshold_slider" {
		t.Fatalf("expected component 'threshold_slider', got %q", submit.Component)
	}
	val, ok := submit.Data["value"].(float64)
	if !ok || !floatEq(val, 0.05) {
		t.Fatalf("expected value 0.05, got %v", submit.Data["value"])
	}
}

func TestCancel(t *testing.T) {
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.05))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected CancelMsg cmd on esc")
	}
	msg := cmd()
	cancel, ok := msg.(crust.CancelMsg)
	if !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", msg)
	}
	if cancel.Component != "threshold_slider" {
		t.Fatalf("expected component 'threshold_slider', got %q", cancel.Component)
	}
}

func TestView(t *testing.T) {
	m := New(
		WithLabel("Test Slider"),
		WithRange(0, 1),
		WithStep(0.01),
		WithDefault(0.5),
		WithCount(500, 1000),
	)
	for _, w := range []int{20, 40, 60, 80, 120} {
		m.SetWidth(w)
		v := m.Render()
		if v == "" {
			t.Fatalf("empty view at width %d", w)
		}
	}
}

func TestSetCount(t *testing.T) {
	m := New(WithRange(0, 1), WithDefault(0.5))
	m.SetCount(42, 100)
	if m.count != 42 || m.total != 100 {
		t.Fatalf("expected count=42, total=100, got %d, %d", m.count, m.total)
	}
}

func TestInitNoop(t *testing.T) {
	m := New()
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}

func TestNonKeyMsgIgnored(t *testing.T) {
	m := New(WithRange(0, 1), WithDefault(0.5))
	_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	if cmd != nil {
		t.Fatal("expected nil cmd for non-key message")
	}
}

func TestDecimalPlaces_ScientificNotation(t *testing.T) {
	tests := []struct {
		step float64
		want int
	}{
		{0.01, 2},
		{0.001, 3},
		{0.000001, 6},  // 1e-06
		{0.0005, 4},    // 5e-04
		{1.0, 0},
		{10.0, 0},
		{0.1, 1},
		{0.5, 1},
		{0.0000001, 7}, // 1e-07
	}
	for _, tt := range tests {
		got := decimalPlaces(tt.step)
		if got != tt.want {
			t.Errorf("decimalPlaces(%g) = %d, want %d", tt.step, got, tt.want)
		}
	}
}

func TestNoValueChangedMsg_AtBoundaries(t *testing.T) {
	// At min, pressing left should NOT emit ValueChangedMsg.
	m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.0))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft})
	if cmd != nil {
		t.Fatal("expected nil cmd when already at min boundary")
	}

	// At max, pressing right should NOT emit ValueChangedMsg.
	m2 := New(WithRange(0, 1), WithStep(0.01), WithDefault(1.0))
	_, cmd2 := m2.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	if cmd2 != nil {
		t.Fatal("expected nil cmd when already at max boundary")
	}

	// Shift variants at boundaries.
	_, cmd3 := m.Update(tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	if cmd3 != nil {
		t.Fatal("expected nil cmd for shift+left at min boundary")
	}
	_, cmd4 := m2.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})
	if cmd4 != nil {
		t.Fatal("expected nil cmd for shift+right at max boundary")
	}
}
