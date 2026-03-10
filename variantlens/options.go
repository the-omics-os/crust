package variantlens

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// ViewMode selects which interpretive layer is emphasized in the body.
type ViewMode string

const (
	ViewSummary  ViewMode = "summary"
	ViewDetail   ViewMode = "detail"
	ViewHGVS     ViewMode = "hgvs"
	ViewEvidence ViewMode = "evidence"
)

func (v ViewMode) String() string {
	if v == "" {
		return string(ViewSummary)
	}
	return string(v)
}

func (v ViewMode) next() ViewMode {
	switch v {
	case ViewDetail:
		return ViewHGVS
	case ViewHGVS:
		return ViewEvidence
	case ViewEvidence:
		return ViewSummary
	default:
		return ViewDetail
	}
}

// Theme controls the visual appearance of VariantLens.
type Theme struct {
	RefBase        color.Color
	AltBase        color.Color
	MismatchBg     color.Color
	HighImpact     color.Color
	ModImpact      color.Color
	LowImpact      color.Color
	ModifierImpact color.Color
	FeatureExon    color.Color
	FeatureCDS     color.Color
	FeatureDomain  color.Color
	FeatureMotif   color.Color
	FeaturePrimer  color.Color
	Text           color.Color
	TextMuted      color.Color
	Border         color.Color
	Header         color.Color
	Selection      color.Color
}

// DefaultTheme returns sensible terminal defaults.
func DefaultTheme() Theme {
	return Theme{
		RefBase:        lipgloss.Color("81"),
		AltBase:        lipgloss.Color("214"),
		MismatchBg:     lipgloss.Color("236"),
		HighImpact:     lipgloss.Color("196"),
		ModImpact:      lipgloss.Color("214"),
		LowImpact:      lipgloss.Color("42"),
		ModifierImpact: lipgloss.Color("39"),
		FeatureExon:    lipgloss.Color("81"),
		FeatureCDS:     lipgloss.Color("121"),
		FeatureDomain:  lipgloss.Color("205"),
		FeatureMotif:   lipgloss.Color("177"),
		FeaturePrimer:  lipgloss.Color("220"),
		Text:           lipgloss.Color("252"),
		TextMuted:      lipgloss.Color("245"),
		Border:         lipgloss.Color("240"),
		Header:         lipgloss.Color("159"),
		Selection:      lipgloss.Color("230"),
	}
}

// Option configures a VariantLens model.
type Option func(*Model)

// WithContext sets the full initial variant context.
func WithContext(ctx VariantContext) Option {
	return func(m *Model) { m.context = cloneContext(ctx) }
}

// WithVariants sets the initial variant set.
func WithVariants(variants []Variant) Option {
	return func(m *Model) { m.context.Variants = cloneVariants(variants) }
}

// WithFeatures sets the initial feature track annotations.
func WithFeatures(features []Feature) Option {
	return func(m *Model) { m.context.Features = cloneFeatures(features) }
}

// WithReferenceSequence sets the reference sequence window.
func WithReferenceSequence(seq string) Option {
	return func(m *Model) { m.context.RefSequence = normalizeSequence(seq) }
}

// WithReferenceStart anchors the reference sequence to a stable coordinate.
func WithReferenceStart(start int) Option {
	return func(m *Model) { m.context.ReferenceStart = start }
}

// WithContextSize sets the visible upstream/downstream sequence window.
func WithContextSize(size int) Option {
	return func(m *Model) { m.context.ContextSize = size }
}

// WithWidth sets the rendering width.
func WithWidth(w int) Option {
	return func(m *Model) { m.width = w }
}

// WithTheme overrides the default theme.
func WithTheme(t Theme) Option {
	return func(m *Model) { m.theme = t }
}

// WithSelectedVariant sets the initially focused variant index.
func WithSelectedVariant(index int) Option {
	return func(m *Model) { m.selected = index }
}

// WithViewMode sets the initial body view.
func WithViewMode(mode ViewMode) Option {
	return func(m *Model) { m.viewMode = mode }
}
