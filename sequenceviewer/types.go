// Package sequenceviewer provides a property-aware biological sequence viewer
// for Bubble Tea applications.
package sequenceviewer

import (
	"image/color"
	"strings"
)

// SequenceType identifies the biological alphabet being rendered.
type SequenceType int

const (
	SequenceUnknown SequenceType = iota
	DNA
	RNA
	Protein
)

// String returns a human-readable sequence type.
func (t SequenceType) String() string {
	switch t {
	case DNA:
		return "DNA"
	case RNA:
		return "RNA"
	case Protein:
		return "Protein"
	default:
		return "Unknown"
	}
}

// ParseSequenceType converts a string into a SequenceType.
func ParseSequenceType(s string) SequenceType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "dna":
		return DNA
	case "rna":
		return RNA
	case "protein", "aa", "amino", "aminoacid", "amino-acid":
		return Protein
	default:
		return SequenceUnknown
	}
}

// IsNucleotide reports whether the sequence is DNA or RNA.
func (t SequenceType) IsNucleotide() bool {
	return t == DNA || t == RNA
}

// ViewMode controls how residues are colorized.
type ViewMode int

const (
	IdentityView ViewMode = iota
	HydrophobicityView
	ChargeView
	GCContentView
	MolWeightView
)

// String returns the display name of the view mode.
func (v ViewMode) String() string {
	switch v {
	case IdentityView:
		return "Identity"
	case HydrophobicityView:
		return "Hydrophobicity"
	case ChargeView:
		return "Charge"
	case GCContentView:
		return "GC Content"
	case MolWeightView:
		return "Molecular Weight"
	default:
		return "Unknown"
	}
}

// ParseViewMode converts a string into a ViewMode.
func ParseViewMode(s string) ViewMode {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "identity":
		return IdentityView
	case "hydrophobicity", "hydro":
		return HydrophobicityView
	case "charge":
		return ChargeView
	case "gc", "gccontent", "gc-content", "gc content":
		return GCContentView
	case "molweight", "molecularweight", "molecular-weight", "molecular weight", "weight":
		return MolWeightView
	default:
		return IdentityView
	}
}

// Residue is the core unit rendered by the sequence viewer.
type Residue struct {
	Position   int
	Code       byte
	Properties Properties

	Coords    *[3]float64
	BFactor   float64
	VdwRadius float64
	Bonds     []int
}

// Properties contains precomputed per-residue biochemical features.
type Properties struct {
	Hydrophobicity float64
	Charge         float64
	MolWeight      float64
	PKa            float64
	Volume         float64

	GCWindow       float64
	MeltingContrib float64

	Conservation float64
}

// Annotation marks a region of the sequence.
type Annotation struct {
	Name      string
	Start     int
	End       int
	Direction int
	Color     color.Color
}

// ORF describes a DNA open reading frame.
type ORF struct {
	Frame    int
	Start    int
	End      int
	Length   int
	Sequence string
}

// RestrictionEnzyme describes a simple recognition motif.
type RestrictionEnzyme struct {
	Name string
	Site string
}

// RestrictionSite marks a matched restriction enzyme site.
type RestrictionSite struct {
	Enzyme   RestrictionEnzyme
	Start    int
	End      int
	Sequence string
}

// FASTARecord is a parsed FASTA entry.
type FASTARecord struct {
	ID          string
	Description string
	Sequence    string
}

func applicableViews(t SequenceType) []ViewMode {
	switch t {
	case Protein:
		return []ViewMode{IdentityView, HydrophobicityView, ChargeView, MolWeightView}
	case DNA, RNA:
		return []ViewMode{IdentityView, GCContentView}
	default:
		return []ViewMode{IdentityView}
	}
}

// ApplicableViews returns the view modes supported for the sequence type.
func ApplicableViews(t SequenceType) []ViewMode {
	views := applicableViews(t)
	return append([]ViewMode(nil), views...)
}

func viewAllowed(t SequenceType, v ViewMode) bool {
	for _, allowed := range applicableViews(t) {
		if allowed == v {
			return true
		}
	}
	return false
}

func ensureApplicableView(current ViewMode, t SequenceType) ViewMode {
	if viewAllowed(t, current) {
		return current
	}
	views := applicableViews(t)
	if len(views) == 0 {
		return IdentityView
	}
	return views[0]
}

func nextView(current ViewMode, t SequenceType) ViewMode {
	views := applicableViews(t)
	if len(views) == 0 {
		return IdentityView
	}
	for i, v := range views {
		if v == current {
			return views[(i+1)%len(views)]
		}
	}
	return views[0]
}
