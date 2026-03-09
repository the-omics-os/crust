package qcdashboard

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func sampleMetrics() []Metric {
	return []Metric{
		{Name: "Reads", Value: 82, Min: 0, Max: 100, Unit: "%", Status: "pass"},
		{Name: "Genes", Value: 65, Min: 0, Max: 100, Unit: "%", Status: "warn"},
		{Name: "Mito %", Value: 3.2, Min: 0, Max: 20, Unit: "%", Status: "pass"},
	}
}

func TestNew(t *testing.T) {
	m := New(WithMetrics(sampleMetrics()), WithTitle("Test QC"))
	if m.Title() != "Test QC" {
		t.Fatalf("expected title 'Test QC', got %q", m.Title())
	}
	if len(m.Metrics()) != 3 {
		t.Fatalf("expected 3 metrics, got %d", len(m.Metrics()))
	}
}

func TestNewDefaults(t *testing.T) {
	m := New()
	if m.Title() != "QC Summary" {
		t.Fatalf("expected default title 'QC Summary', got %q", m.Title())
	}
	if m.width != 80 {
		t.Fatalf("expected default width 80, got %d", m.width)
	}
}

func TestViewContainsMetricNames(t *testing.T) {
	m := New(WithMetrics(sampleMetrics()))
	v := m.Render()
	for _, name := range []string{"Reads", "Genes", "Mito %"} {
		if !strings.Contains(v, name) {
			t.Fatalf("View should contain metric name %q", name)
		}
	}
}

func TestViewContainsStatusText(t *testing.T) {
	metrics := []Metric{
		{Name: "PassMetric", Value: 90, Min: 0, Max: 100, Status: "pass"},
		{Name: "WarnMetric", Value: 50, Min: 0, Max: 100, Status: "warn"},
		{Name: "FailMetric", Value: 10, Min: 0, Max: 100, Status: "fail"},
	}
	m := New(WithMetrics(metrics))
	v := m.Render()
	for _, status := range []string{"PASS", "WARN", "FAIL"} {
		if !strings.Contains(v, status) {
			t.Fatalf("View should contain status %q", status)
		}
	}
}

func TestViewVariousWidths(t *testing.T) {
	m := New(WithMetrics(sampleMetrics()))
	for _, w := range []int{20, 40, 60, 80, 120} {
		m.SetWidth(w)
		v := m.Render()
		if v == "" {
			t.Fatalf("empty view at width %d", w)
		}
	}
}

func TestSetMetrics(t *testing.T) {
	m := New(WithMetrics(sampleMetrics()))
	m.SetMetrics([]Metric{{Name: "New", Value: 1, Min: 0, Max: 10, Status: "pass"}})
	if len(m.Metrics()) != 1 {
		t.Fatalf("expected 1 metric after SetMetrics, got %d", len(m.Metrics()))
	}
}

func TestUpdateNoop(t *testing.T) {
	m := New(WithMetrics(sampleMetrics()))
	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("expected nil cmd from non-interactive component")
	}
	got := updated.(Model)
	if len(got.Metrics()) != 3 {
		t.Fatal("update should not change state")
	}
}

func TestInitNoop(t *testing.T) {
	m := New()
	if cmd := m.Init(); cmd != nil {
		t.Fatal("expected nil cmd from Init")
	}
}

func TestWithTheme(t *testing.T) {
	theme := DefaultTheme()
	theme.Pass = lipgloss.Color("46") // different green
	m := New(WithTheme(theme))
	if m.theme.Pass != lipgloss.Color("46") {
		t.Fatalf("expected custom theme pass color")
	}
}

func TestMetrics_ReturnsCopy(t *testing.T) {
	original := sampleMetrics()
	m := New(WithMetrics(original))

	// Mutate the returned slice — should NOT affect the model.
	got := m.Metrics()
	got[0].Name = "MUTATED"

	internal := m.Metrics()
	if internal[0].Name == "MUTATED" {
		t.Fatal("Metrics() returned a reference, not a copy — mutation propagated")
	}
}

func TestSetMetrics_DefensiveCopy(t *testing.T) {
	m := New()
	metrics := sampleMetrics()
	m.SetMetrics(metrics)

	// Mutate the original slice — should NOT affect the model.
	metrics[0].Name = "MUTATED"

	internal := m.Metrics()
	if internal[0].Name == "MUTATED" {
		t.Fatal("SetMetrics did not make a defensive copy — mutation propagated")
	}
}

func TestWithMetrics_DefensiveCopy(t *testing.T) {
	metrics := sampleMetrics()
	m := New(WithMetrics(metrics))

	// Mutate the original slice.
	metrics[0].Name = "MUTATED"

	internal := m.Metrics()
	if internal[0].Name == "MUTATED" {
		t.Fatal("WithMetrics did not make a defensive copy — mutation propagated")
	}
}
