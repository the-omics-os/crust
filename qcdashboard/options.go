package qcdashboard

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of the dashboard.
type Theme struct {
	Pass      color.Color
	Warn      color.Color
	Fail      color.Color
	Border    color.Color
	Text      color.Color
	TextMuted color.Color
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		Pass:      lipgloss.Color("42"),  // green
		Warn:      lipgloss.Color("214"), // yellow/orange
		Fail:      lipgloss.Color("196"), // red
		Border:    lipgloss.Color("240"), // dim gray
		Text:      lipgloss.Color("252"), // light gray
		TextMuted: lipgloss.Color("240"), // dim gray
	}
}

// Option configures a QCDashboard Model.
type Option func(*Model)

// WithTitle sets the dashboard title.
func WithTitle(title string) Option {
	return func(m *Model) { m.title = title }
}

// WithMetrics sets the initial metrics (defensive copy).
func WithMetrics(metrics []Metric) Option {
	return func(m *Model) { m.metrics = append([]Metric(nil), metrics...) }
}

// WithWidth sets the rendering width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) { m.theme = t }
}
