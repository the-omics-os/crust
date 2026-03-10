package sequenceviewer

// Element stores minimal atomic reference data useful for structure-aware
// residue enrichment.
type Element struct {
	Symbol         string
	Name           string
	AtomicNumber   int
	AtomicMass     float64
	CovalentRadius float64
	VdwRadius      float64
}

// BioElements is a small bio-relevant atomic lookup table.
var BioElements = map[string]Element{
	"H":  {Symbol: "H", Name: "Hydrogen", AtomicNumber: 1, AtomicMass: 1.008, CovalentRadius: 0.31, VdwRadius: 1.20},
	"C":  {Symbol: "C", Name: "Carbon", AtomicNumber: 6, AtomicMass: 12.011, CovalentRadius: 0.76, VdwRadius: 1.70},
	"N":  {Symbol: "N", Name: "Nitrogen", AtomicNumber: 7, AtomicMass: 14.007, CovalentRadius: 0.71, VdwRadius: 1.55},
	"O":  {Symbol: "O", Name: "Oxygen", AtomicNumber: 8, AtomicMass: 15.999, CovalentRadius: 0.66, VdwRadius: 1.52},
	"P":  {Symbol: "P", Name: "Phosphorus", AtomicNumber: 15, AtomicMass: 30.974, CovalentRadius: 1.07, VdwRadius: 1.80},
	"S":  {Symbol: "S", Name: "Sulfur", AtomicNumber: 16, AtomicMass: 32.06, CovalentRadius: 1.05, VdwRadius: 1.80},
	"SE": {Symbol: "Se", Name: "Selenium", AtomicNumber: 34, AtomicMass: 78.971, CovalentRadius: 1.20, VdwRadius: 1.90},
	"FE": {Symbol: "Fe", Name: "Iron", AtomicNumber: 26, AtomicMass: 55.845, CovalentRadius: 1.24, VdwRadius: 2.00},
	"ZN": {Symbol: "Zn", Name: "Zinc", AtomicNumber: 30, AtomicMass: 65.38, CovalentRadius: 1.22, VdwRadius: 2.10},
	"CA": {Symbol: "Ca", Name: "Calcium", AtomicNumber: 20, AtomicMass: 40.078, CovalentRadius: 1.76, VdwRadius: 2.31},
	"MG": {Symbol: "Mg", Name: "Magnesium", AtomicNumber: 12, AtomicMass: 24.305, CovalentRadius: 1.41, VdwRadius: 1.73},
}
