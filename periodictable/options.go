package periodictable

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of the periodic table.
type Theme struct {
	AlkaliMetal     color.Color
	AlkalineEarth   color.Color
	TransitionMetal color.Color
	PostTransition  color.Color
	Metalloid       color.Color
	Nonmetal        color.Color
	Halogen         color.Color
	NobleGas        color.Color
	Lanthanide      color.Color
	Actinide        color.Color
	Selected        color.Color
	Cursor          color.Color
	Border          color.Color
	Text            color.Color
	TextMuted       color.Color
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		AlkaliMetal:     lipgloss.Color("203"),
		AlkalineEarth:   lipgloss.Color("215"),
		TransitionMetal: lipgloss.Color("39"),
		PostTransition:  lipgloss.Color("109"),
		Metalloid:       lipgloss.Color("149"),
		Nonmetal:        lipgloss.Color("77"),
		Halogen:         lipgloss.Color("81"),
		NobleGas:        lipgloss.Color("117"),
		Lanthanide:      lipgloss.Color("220"),
		Actinide:        lipgloss.Color("204"),
		Selected:        lipgloss.Color("24"),
		Cursor:          lipgloss.Color("238"),
		Border:          lipgloss.Color("241"),
		Text:            lipgloss.Color("255"),
		TextMuted:       lipgloss.Color("248"),
	}
}

// Option configures a PeriodicTable Model.
type Option func(*Model)

// WithWidth sets the rendering width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithTheme overrides the default theme.
func WithTheme(theme Theme) Option {
	return func(m *Model) { m.theme = theme }
}

// WithSelected sets the initial focused element by symbol.
func WithSelected(symbol string) Option {
	return func(m *Model) {
		if element, ok := elementBySymbol(symbol); ok {
			m.focusedNumber = element.Number
		}
	}
}

// WithHighlights sets externally highlighted element symbols.
func WithHighlights(symbols ...string) Option {
	return func(m *Model) {
		m.highlights = highlightSet(symbols)
	}
}

func highlightSet(symbols []string) map[string]struct{} {
	set := make(map[string]struct{}, len(symbols))
	for _, symbol := range symbols {
		symbol = canonicalSymbol(symbol)
		if symbol == "" {
			continue
		}
		if _, ok := elementBySymbol(symbol); !ok {
			continue
		}
		set[symbol] = struct{}{}
	}
	return set
}

func canonicalSymbol(symbol string) string {
	symbol = strings.TrimSpace(strings.ToLower(symbol))
	if symbol == "" {
		return ""
	}
	if len(symbol) == 1 {
		return strings.ToUpper(symbol)
	}
	return strings.ToUpper(symbol[:1]) + symbol[1:]
}
