package moleculeviewer

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the visual appearance of the molecule viewer.
type Theme struct {
	Carbon       color.Color
	Nitrogen     color.Color
	Oxygen       color.Color
	Sulfur       color.Color
	Phosphorus   color.Color
	Halogen      color.Color
	Metal        color.Color
	Hydrogen     color.Color
	Bond         color.Color
	DoubleBond   color.Color
	TripleBond   color.Color
	AromaticBond color.Color
	Positive     color.Color
	Negative     color.Color
	Scaffold     color.Color
	RGroup       color.Color
	Selected     color.Color
	Search       color.Color
	Border       color.Color
	Text         color.Color
	TextMuted    color.Color
	Help         color.Color
}

// DefaultTheme returns a chemistry-oriented terminal palette.
func DefaultTheme() Theme {
	return Theme{
		Carbon:       lipgloss.Color("250"),
		Nitrogen:     lipgloss.Color("39"),
		Oxygen:       lipgloss.Color("196"),
		Sulfur:       lipgloss.Color("220"),
		Phosphorus:   lipgloss.Color("208"),
		Halogen:      lipgloss.Color("35"),
		Metal:        lipgloss.Color("45"),
		Hydrogen:     lipgloss.Color("255"),
		Bond:         lipgloss.Color("244"),
		DoubleBond:   lipgloss.Color("252"),
		TripleBond:   lipgloss.Color("111"),
		AromaticBond: lipgloss.Color("179"),
		Positive:     lipgloss.Color("201"),
		Negative:     lipgloss.Color("81"),
		Scaffold:     lipgloss.Color("117"),
		RGroup:       lipgloss.Color("180"),
		Selected:     lipgloss.Color("15"),
		Search:       lipgloss.Color("48"),
		Border:       lipgloss.Color("240"),
		Text:         lipgloss.Color("252"),
		TextMuted:    lipgloss.Color("244"),
		Help:         lipgloss.Color("111"),
	}
}

// Option configures a SmallMoleculeViewer model.
type Option func(*Model)

// WithMolecule sets the initial molecule.
func WithMolecule(mol Molecule) Option {
	return func(m *Model) {
		m.setMolecule(mol)
	}
}

// WithSMILES parses and sets the initial molecule from a SMILES string.
func WithSMILES(smiles string) Option {
	return func(m *Model) {
		if err := m.SetSMILES(smiles); err != nil {
			m.loadErr = err
		}
	}
}

// WithMOL parses and sets the initial molecule from a MOL block.
func WithMOL(molBlock string) Option {
	return func(m *Model) {
		if err := m.SetMOL(molBlock); err != nil {
			m.loadErr = err
		}
	}
}

// WithMolBlock is an alias for WithMOL.
func WithMolBlock(molBlock string) Option {
	return WithMOL(molBlock)
}

// WithSDF parses and sets the initial molecule from an SDF payload.
func WithSDF(sdf string) Option {
	return func(m *Model) {
		if err := m.SetSDF(sdf); err != nil {
			m.loadErr = err
		}
	}
}

// WithName overrides the displayed molecule name.
func WithName(name string) Option {
	return func(m *Model) {
		m.title = name
		m.molecule.Name = name
	}
}

// WithWidth sets the render width.
func WithWidth(w int) Option {
	return func(m *Model) {
		m.width = w
	}
}

// WithHeight sets the render height.
func WithHeight(h int) Option {
	return func(m *Model) {
		m.height = h
	}
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) {
		m.theme = t
	}
}
