package thresholdslider

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of the slider.
type Theme struct {
	Value  color.Color // active value color
	Filled color.Color // filled portion of bar
	Empty  color.Color // empty portion of bar
	Cursor color.Color // cursor position
	Count  color.Color // count/total text
	Title  color.Color // title text
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		Value:  lipgloss.Color("63"),  // purple
		Filled: lipgloss.Color("63"),  // purple
		Empty:  lipgloss.Color("240"), // dim gray
		Cursor: lipgloss.Color("15"),  // white
		Count:  lipgloss.Color("245"), // light gray
		Title:  lipgloss.Color("252"), // near-white
	}
}

// Option configures a ThresholdSlider Model.
type Option func(*Model)

// WithLabel sets the slider label.
func WithLabel(label string) Option {
	return func(m *Model) { m.label = label }
}

// WithRange sets the min/max bounds.
func WithRange(min, max float64) Option {
	return func(m *Model) {
		m.min = min
		m.max = max
	}
}

// WithStep sets the adjustment step size.
func WithStep(step float64) Option {
	return func(m *Model) { m.step = step }
}

// WithDefault sets the initial value.
func WithDefault(value float64) Option {
	return func(m *Model) { m.value = value }
}

// WithUnit sets the display unit suffix.
func WithUnit(unit string) Option {
	return func(m *Model) { m.unit = unit }
}

// WithCount sets the initial count/total display.
func WithCount(count, total int) Option {
	return func(m *Model) {
		m.count = count
		m.total = total
	}
}

// WithWidth sets the rendering width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) { m.theme = t }
}
