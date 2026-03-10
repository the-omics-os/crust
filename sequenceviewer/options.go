package sequenceviewer

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme controls the appearance of the sequence viewer.
type Theme struct {
	Adenine     color.Color
	Thymine     color.Color
	Guanine     color.Color
	Cytosine    color.Color
	Uracil      color.Color
	Hydrophobic color.Color
	Positive    color.Color
	Negative    color.Color
	Polar       color.Color
	Aromatic    color.Color
	Special     color.Color

	GradientLow  color.Color
	GradientHigh color.Color

	LineNumber color.Color
	Header     color.Color
	Separator  color.Color
	Complement color.Color
	Unknown    color.Color
	ViewLabel  color.Color
}

// DefaultTheme returns a scientifically conventional default palette.
func DefaultTheme() Theme {
	return Theme{
		Adenine:      lipgloss.Color("34"),
		Thymine:      lipgloss.Color("160"),
		Guanine:      lipgloss.Color("178"),
		Cytosine:     lipgloss.Color("33"),
		Uracil:       lipgloss.Color("160"),
		Hydrophobic:  lipgloss.Color("33"),
		Positive:     lipgloss.Color("160"),
		Negative:     lipgloss.Color("163"),
		Polar:        lipgloss.Color("34"),
		Aromatic:     lipgloss.Color("37"),
		Special:      lipgloss.Color("208"),
		GradientLow:  lipgloss.Color("39"),
		GradientHigh: lipgloss.Color("196"),
		LineNumber:   lipgloss.Color("240"),
		Header:       lipgloss.Color("252"),
		Separator:    lipgloss.Color("240"),
		Complement:   lipgloss.Color("245"),
		Unknown:      lipgloss.Color("240"),
		ViewLabel:    lipgloss.Color("81"),
	}
}

// Option configures a SequenceViewer model.
type Option func(*Model)

// WithSequence sets the input sequence and type.
func WithSequence(seq string, t SequenceType) Option {
	return func(m *Model) {
		m.sequence = NormalizeSequence(seq)
		m.seqType = t
		m.residues = nil
	}
}

// WithResidues sets precomputed residues.
func WithResidues(residues []Residue) Option {
	return func(m *Model) {
		m.residues = copyResidues(residues)
		m.sequence = ""
	}
}

// WithView sets the initial view mode.
func WithView(v ViewMode) Option {
	return func(m *Model) { m.view = v }
}

// WithComplement sets the initial complement visibility.
func WithComplement(show bool) Option {
	return func(m *Model) { m.showComplement = show }
}

// WithAnnotations sets feature annotations.
func WithAnnotations(annotations []Annotation) Option {
	return func(m *Model) { m.annotations = copyAnnotations(annotations) }
}

// WithResiduesPerLine overrides the preferred residues per line.
func WithResiduesPerLine(n int) Option {
	return func(m *Model) { m.residuesPerLine = n }
}

// WithWidth sets the render width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) { m.theme = t }
}

// WithHeader toggles the header line.
func WithHeader(show bool) Option {
	return func(m *Model) { m.showHeader = show }
}

// WithGCWindow sets the nucleotide GC sliding-window size.
func WithGCWindow(size int) Option {
	return func(m *Model) { m.gcWindow = size }
}

func copyResidues(src []Residue) []Residue {
	if len(src) == 0 {
		return nil
	}
	dst := make([]Residue, len(src))
	for i, residue := range src {
		dst[i] = residue
		dst[i].Bonds = append([]int(nil), residue.Bonds...)
	}
	return dst
}

func copyAnnotations(src []Annotation) []Annotation {
	if len(src) == 0 {
		return nil
	}
	dst := make([]Annotation, len(src))
	copy(dst, src)
	return dst
}

func copyORFs(src []ORF) []ORF {
	if len(src) == 0 {
		return nil
	}
	dst := make([]ORF, len(src))
	copy(dst, src)
	return dst
}

func copyRestrictionSites(src []RestrictionSite) []RestrictionSite {
	if len(src) == 0 {
		return nil
	}
	dst := make([]RestrictionSite, len(src))
	copy(dst, src)
	return dst
}
