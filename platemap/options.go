package platemap

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of PlateMap.
type Theme struct {
	Hit          color.Color
	PositiveCtrl color.Color
	NegativeCtrl color.Color
	Sample       color.Color
	Empty        color.Color
	Missing      color.Color
	CursorBg     color.Color
	SelectedRow  color.Color
	SelectedCol  color.Color
	Border       color.Color
	Text         color.Color
	TextMuted    color.Color
	Header       color.Color
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		Hit:          lipgloss.Color("203"),
		PositiveCtrl: lipgloss.Color("42"),
		NegativeCtrl: lipgloss.Color("75"),
		Sample:       lipgloss.Color("252"),
		Empty:        lipgloss.Color("240"),
		Missing:      lipgloss.Color("214"),
		CursorBg:     lipgloss.Color("236"),
		SelectedRow:  lipgloss.Color("238"),
		SelectedCol:  lipgloss.Color("235"),
		Border:       lipgloss.Color("240"),
		Text:         lipgloss.Color("252"),
		TextMuted:    lipgloss.Color("245"),
		Header:       lipgloss.Color("117"),
	}
}

// Option configures a PlateMap model.
type Option func(*Model)

// WithPlate sets the initial plate payload.
func WithPlate(plate PlateData) Option {
	return func(m *Model) {
		m.plate = plate.Copy()
	}
}

// WithFormat sets the current plate format.
func WithFormat(format PlateFormat) Option {
	return func(m *Model) {
		m.plate.Format = format
	}
}

// WithTitle sets the plate title.
func WithTitle(title string) Option {
	return func(m *Model) {
		m.plate.Title = title
	}
}

// WithTheme overrides the default theme.
func WithTheme(theme Theme) Option {
	return func(m *Model) {
		m.theme = theme
	}
}

// WithWidth sets the rendering width.
func WithWidth(width int) Option {
	return func(m *Model) {
		m.width = width
	}
}

// WithHeight sets the rendering height budget.
func WithHeight(height int) Option {
	return func(m *Model) {
		m.height = height
	}
}

// WithViewMode sets the initial view mode.
func WithViewMode(mode ViewMode) Option {
	return func(m *Model) {
		m.mode = mode
	}
}

// WithCursor sets the initial focused well coordinate.
func WithCursor(row, col int) Option {
	return func(m *Model) {
		m.cursorRow = row
		m.cursorCol = col
	}
}
