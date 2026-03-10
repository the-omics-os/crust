package variantlens

import (
	"slices"
	"sort"
	"strings"
)

// Variant describes a single sequence change and its biological annotation.
type Variant struct {
	Position    int
	Ref         string
	Alt         string
	Type        string
	Consequence string
	HGVS        string
	Gene        string
	Impact      string
	Evidence    string
}

// Feature marks an annotated interval near the visible variant window.
type Feature struct {
	Name  string
	Type  string
	Start int
	End   int
}

// VariantContext carries the reference sequence window and its annotations.
//
// ReferenceStart is optional. When present it anchors RefSequence and Features
// to stable genomic or transcript coordinates so multiple variants can share a
// single aligned region.
type VariantContext struct {
	RefSequence    string
	Variants       []Variant
	Features       []Feature
	ContextSize    int
	ReferenceStart int
}

func cloneContext(ctx VariantContext) VariantContext {
	return VariantContext{
		RefSequence:    normalizeSequence(ctx.RefSequence),
		Variants:       cloneVariants(ctx.Variants),
		Features:       cloneFeatures(ctx.Features),
		ContextSize:    ctx.ContextSize,
		ReferenceStart: ctx.ReferenceStart,
	}
}

func cloneVariants(variants []Variant) []Variant {
	cloned := make([]Variant, len(variants))
	for i, variant := range variants {
		cloned[i] = Variant{
			Position:    variant.Position,
			Ref:         normalizeAllele(variant.Ref),
			Alt:         normalizeAllele(variant.Alt),
			Type:        normalizeVariantType(variant.Type, variant.Ref, variant.Alt),
			Consequence: strings.TrimSpace(strings.ToLower(variant.Consequence)),
			HGVS:        strings.TrimSpace(variant.HGVS),
			Gene:        strings.TrimSpace(variant.Gene),
			Impact:      strings.ToUpper(strings.TrimSpace(variant.Impact)),
			Evidence:    strings.TrimSpace(variant.Evidence),
		}
	}

	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Position != cloned[j].Position {
			return cloned[i].Position < cloned[j].Position
		}
		if cloned[i].Gene != cloned[j].Gene {
			return cloned[i].Gene < cloned[j].Gene
		}
		return cloned[i].HGVS < cloned[j].HGVS
	})
	return cloned
}

func cloneFeatures(features []Feature) []Feature {
	cloned := make([]Feature, len(features))
	for i, feature := range features {
		start := feature.Start
		end := feature.End
		if start > 0 && end > 0 && end < start {
			start, end = end, start
		}
		cloned[i] = Feature{
			Name:  strings.TrimSpace(feature.Name),
			Type:  strings.ToLower(strings.TrimSpace(feature.Type)),
			Start: start,
			End:   end,
		}
	}

	sort.SliceStable(cloned, func(i, j int) bool {
		if cloned[i].Start != cloned[j].Start {
			return cloned[i].Start < cloned[j].Start
		}
		if cloned[i].End != cloned[j].End {
			return cloned[i].End < cloned[j].End
		}
		return cloned[i].Name < cloned[j].Name
	})
	return cloned
}

func normalizeSequence(seq string) string {
	seq = strings.ToUpper(strings.TrimSpace(seq))
	seq = strings.ReplaceAll(seq, " ", "")
	seq = strings.ReplaceAll(seq, "\n", "")
	seq = strings.ReplaceAll(seq, "\t", "")
	return seq
}

func normalizeAllele(allele string) string {
	allele = strings.ToUpper(strings.TrimSpace(allele))
	return strings.ReplaceAll(allele, "-", "")
}

func normalizeVariantType(kind, ref, alt string) string {
	kind = strings.TrimSpace(strings.ToUpper(kind))
	if kind != "" {
		return kind
	}

	ref = normalizeAllele(ref)
	alt = normalizeAllele(alt)
	switch {
	case len(ref) == len(alt) && len(ref) <= 1:
		return "SNV"
	case len(ref) == len(alt):
		return "MNV"
	case len(ref) < len(alt):
		return "INSERTION"
	case len(ref) > len(alt):
		return "DELETION"
	default:
		return "VARIANT"
	}
}

func overlappingFeatures(features []Feature, start, end int) []Feature {
	if len(features) == 0 {
		return nil
	}

	var overlaps []Feature
	for _, feature := range features {
		if feature.End > 0 && feature.Start > 0 && feature.End < start {
			continue
		}
		if feature.Start > end {
			continue
		}
		overlaps = append(overlaps, feature)
	}
	return overlaps
}

func findCDSFeature(features []Feature, position int) (Feature, bool) {
	for _, feature := range features {
		if feature.Type != "cds" {
			continue
		}
		if position >= feature.Start && position <= feature.End {
			return feature, true
		}
	}
	return Feature{}, false
}

func splitAnnotatedText(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ';' || r == '|' || r == ','
	})
	tokens := make([]string, 0, len(fields))
	for _, field := range fields {
		token := strings.TrimSpace(field)
		if token != "" {
			tokens = append(tokens, token)
		}
	}
	return slices.Clip(tokens)
}
