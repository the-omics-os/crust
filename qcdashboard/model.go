// Package qcdashboard provides an inline QC metrics dashboard component.
//
// QCDashboard is non-interactive — it displays quality-control metrics
// as colored status bars with pass/warn/fail indicators. Metrics can be
// updated programmatically via SetMetrics.
package qcdashboard

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Model is the BubbleTea model for the QC dashboard.
type Model struct {
	title   string
	metrics []Metric
	theme   Theme
	width   int
}

// New creates a QCDashboard with the given options.
func New(opts ...Option) Model {
	m := Model{
		title: "QC Summary",
		theme: DefaultTheme(),
		width: 80,
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model. Non-interactive — returns model unchanged.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the dashboard as a plain string (useful for embedding).
func (m Model) Render() string {
	return m.render()
}

func (m Model) render() string {
	w := m.width
	if w < 20 {
		w = 20
	}

	borderStyle := lipgloss.NewStyle().Foreground(m.theme.Border)

	innerW := w - 4 // "| " prefix + " |" suffix
	if innerW < 16 {
		innerW = 16
	}

	var lines []string

	// Top border with title.
	titleText := " " + m.title + " "
	dashCount := innerW - len(titleText)
	if dashCount < 2 {
		dashCount = 2
	}
	leftDash := 2
	rightDash := dashCount - leftDash
	if rightDash < 0 {
		rightDash = 0
	}
	topBorder := borderStyle.Render("+" + strings.Repeat("-", leftDash) + titleText + strings.Repeat("-", rightDash) + "+")
	lines = append(lines, topBorder)

	for _, metric := range m.metrics {
		line := m.renderMetricLine(metric, innerW)
		lines = append(lines, borderStyle.Render("| ")+line+borderStyle.Render(" |"))
	}

	bottomBorder := borderStyle.Render("+" + strings.Repeat("-", innerW) + "+")
	lines = append(lines, bottomBorder)

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// SetMetrics replaces all metrics (defensive copy).
func (m *Model) SetMetrics(metrics []Metric) {
	m.metrics = append([]Metric(nil), metrics...)
}

// SetTitle updates the dashboard title.
func (m *Model) SetTitle(title string) { m.title = title }

// SetWidth updates the rendering width.
func (m *Model) SetWidth(w int) { m.width = w }

// Metrics returns a copy of the current metrics.
func (m Model) Metrics() []Metric { return append([]Metric(nil), m.metrics...) }

// Title returns the current title.
func (m Model) Title() string { return m.title }

func (m Model) renderMetricLine(metric Metric, innerW int) string {
	name := metric.Name
	if len(name) > 12 {
		name = name[:12]
	}
	nameStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	namePadded := nameStyle.Render(fmt.Sprintf("%-12s", name))

	statusText := strings.ToUpper(metric.Status)
	if statusText == "" {
		statusText = "    "
	}
	statusStyled := lipgloss.NewStyle().Bold(true).Foreground(m.statusColor(metric.Status)).Render(statusText)

	valStr := formatValue(metric.Value, metric.Unit)

	valDisplayLen := len(valStr)
	if valDisplayLen < 6 {
		valDisplayLen = 6
	}
	statusDisplayLen := len(statusText)
	if statusDisplayLen < 4 {
		statusDisplayLen = 4
	}
	overhead := 12 + 1 + 1 + valDisplayLen + 2 + statusDisplayLen
	barWidth := innerW - overhead
	if barWidth < 2 {
		barWidth = 2
	}
	if barWidth > 30 {
		barWidth = 30
	}

	fillRatio := 0.0
	rangeVal := metric.Max - metric.Min
	if rangeVal > 0 {
		fillRatio = (metric.Value - metric.Min) / rangeVal
	}
	fillRatio = math.Max(0, math.Min(1, fillRatio))
	fillCount := int(math.Round(fillRatio * float64(barWidth)))

	barStyle := lipgloss.NewStyle().Foreground(m.statusColor(metric.Status))
	emptyStyle := lipgloss.NewStyle().Foreground(m.theme.Border)

	bar := barStyle.Render(strings.Repeat("#", fillCount)) +
		emptyStyle.Render(strings.Repeat(" ", barWidth-fillCount))

	valStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	valPadded := valStyle.Render(fmt.Sprintf("%-*s", valDisplayLen, valStr))

	return namePadded + " " + bar + " " + valPadded + "  " + statusStyled
}

func (m Model) statusColor(status string) color.Color {
	switch strings.ToLower(status) {
	case "pass":
		return m.theme.Pass
	case "warn":
		return m.theme.Warn
	case "fail":
		return m.theme.Fail
	default:
		return m.theme.Border
	}
}

func formatValue(value float64, unit string) string {
	var s string
	if value == math.Trunc(value) {
		s = fmt.Sprintf("%.0f", value)
	} else {
		s = fmt.Sprintf("%.1f", value)
	}
	if unit != "" {
		s += unit
	}
	return s
}
