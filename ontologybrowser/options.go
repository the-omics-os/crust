package ontologybrowser

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of the ontology browser.
type Theme struct {
	Selected        color.Color
	Expanded        color.Color
	Collapsed       color.Color
	Leaf            color.Color
	Branch          color.Color
	SearchHighlight color.Color
	Text            color.Color
	TextMuted       color.Color
	Border          color.Color
	Title           color.Color
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		Selected:        lipgloss.Color("45"),
		Expanded:        lipgloss.Color("42"),
		Collapsed:       lipgloss.Color("214"),
		Leaf:            lipgloss.Color("81"),
		Branch:          lipgloss.Color("252"),
		SearchHighlight: lipgloss.Color("51"),
		Text:            lipgloss.Color("252"),
		TextMuted:       lipgloss.Color("245"),
		Border:          lipgloss.Color("240"),
		Title:           lipgloss.Color("230"),
	}
}

// Option configures an OntologyBrowser model.
type Option func(*Model)

// WithRoots sets the initial root ontology nodes.
func WithRoots(nodes []OntologyNode) Option {
	return func(m *Model) {
		m.roots = cloneNodes(nodes)
	}
}

// WithWidth sets the rendering width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithHeight sets the rendering height.
func WithHeight(h int) Option {
	return func(m *Model) { m.height = h }
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) { m.theme = t }
}
