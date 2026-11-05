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
	// Groups colours the groups of the Groups view, in PlateData.Groups
	// order; it repeats when there are more groups than colours.
	Groups []color.Color
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
		// Crust's category palette (periodictable), ordered so that the
		// first groups are far apart and none is Missing's orange, which
		// the Groups view uses for "needs attention".
		Groups: []color.Color{
			lipgloss.Color("39"),
			lipgloss.Color("77"),
			lipgloss.Color("204"),
			lipgloss.Color("220"),
			lipgloss.Color("81"),
			lipgloss.Color("149"),
			lipgloss.Color("109"),
			lipgloss.Color("117"),
			lipgloss.Color("203"),
		},
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

// WithGridOnly draws only the grid (column numbers, row letters and wells),
// without the header, separator and footer, for a host that draws its own
// lines around the plate. All rows fit when the height allows one line for
// the column numbers plus one per row.
func WithGridOnly(gridOnly bool) Option {
	return func(m *Model) {
		m.gridOnly = gridOnly
	}
}
