package sequenceviewer

import (
	"image/color"
	"math"
	"unicode"

	"charm.land/lipgloss/v2"
)

const (
	minHydrophobicity = -4.5
	maxHydrophobicity = 4.5
	minAAMass         = 75.07
	maxAAMass         = 204.23
)

func clamp01(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}

func (m Model) identityColor(code byte) color.Color {
	code = byte(unicode.ToUpper(rune(code)))
	if m.seqType.IsNucleotide() {
		switch code {
		case 'A':
			return m.theme.Adenine
		case 'T':
			return m.theme.Thymine
		case 'U':
			return m.theme.Uracil
		case 'G':
			return m.theme.Guanine
		case 'C':
			return m.theme.Cytosine
		default:
			return m.theme.Unknown
		}
	}

	switch aminoAcidIdentityGroup(code) {
	case "hydrophobic":
		return m.theme.Hydrophobic
	case "positive":
		return m.theme.Positive
	case "negative":
		return m.theme.Negative
	case "polar":
		return m.theme.Polar
	case "aromatic":
		return m.theme.Aromatic
	case "special":
		return m.theme.Special
	default:
		return m.theme.Unknown
	}
}

func (m Model) scaledGradient(norm float64) color.Color {
	switch {
	case norm < 0.34:
		return m.theme.GradientLow
	case norm < 0.67:
		return m.theme.ViewLabel
	default:
		return m.theme.GradientHigh
	}
}

func (m Model) propertyScale(residue Residue) (float64, bool) {
	switch m.view {
	case HydrophobicityView:
		return clamp01((residue.Properties.Hydrophobicity - minHydrophobicity) / (maxHydrophobicity - minHydrophobicity)), true
	case ChargeView:
		return clamp01((residue.Properties.Charge + 1) / 2), true
	case GCContentView:
		return clamp01(residue.Properties.GCWindow), true
	case MolWeightView:
		return clamp01((residue.Properties.MolWeight - minAAMass) / (maxAAMass - minAAMass)), true
	default:
		return 0, false
	}
}

func (m Model) colorForResidue(residue Residue) color.Color {
	switch m.view {
	case IdentityView:
		return m.identityColor(residue.Code)
	case HydrophobicityView, GCContentView, MolWeightView:
		scale, _ := m.propertyScale(residue)
		return m.scaledGradient(scale)
	case ChargeView:
		switch {
		case residue.Properties.Charge < -0.2:
			return m.theme.Negative
		case residue.Properties.Charge > 0.2:
			return m.theme.Positive
		default:
			return m.theme.ViewLabel
		}
	default:
		return m.theme.Unknown
	}
}

func (m Model) styledResidue(residue Residue) string {
	style := lipgloss.NewStyle().Foreground(m.colorForResidue(residue))
	if annColor, ok := m.annotationColorAt(residue.Position); ok {
		style = style.Bold(true).Underline(true).UnderlineColor(annColor)
	}
	return style.Render(string(residue.Code))
}

func (m Model) styledComplement(code byte) string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Complement).
		Render(string(Complement(code, DNA)))
}

func (m Model) styledPropertyGlyph(residue Residue) string {
	scale, ok := m.propertyScale(residue)
	if !ok {
		return lipgloss.NewStyle().Foreground(m.theme.Unknown).Render(" ")
	}
	return lipgloss.NewStyle().
		Foreground(m.colorForResidue(residue)).
		Render(propertyGlyph(scale))
}

func propertyGlyph(scale float64) string {
	glyphs := []string{" ", ".", ":", "-", "=", "+", "*", "#", "%", "@"}
	idx := int(math.Round(clamp01(scale) * float64(len(glyphs)-1)))
	return glyphs[idx]
}
