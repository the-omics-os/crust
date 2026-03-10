package sequenceviewer

import "strings"

// AminoAcid stores standard physicochemical values for a residue.
type AminoAcid struct {
	Code            byte
	ThreeLetterCode string
	Name            string
	Hydrophobicity  float64
	Charge          float64
	MolWeight       float64
	PKa             float64
	Volume          float64
}

var aminoAcidTable = map[byte]AminoAcid{
	'A': {Code: 'A', ThreeLetterCode: "Ala", Name: "Alanine", Hydrophobicity: 1.8, Charge: 0, MolWeight: 89.09, PKa: 0, Volume: 88.6},
	'R': {Code: 'R', ThreeLetterCode: "Arg", Name: "Arginine", Hydrophobicity: -4.5, Charge: 1.0, MolWeight: 174.20, PKa: 12.48, Volume: 173.4},
	'N': {Code: 'N', ThreeLetterCode: "Asn", Name: "Asparagine", Hydrophobicity: -3.5, Charge: 0, MolWeight: 132.12, PKa: 0, Volume: 114.1},
	'D': {Code: 'D', ThreeLetterCode: "Asp", Name: "Aspartate", Hydrophobicity: -3.5, Charge: -1.0, MolWeight: 133.10, PKa: 3.65, Volume: 111.1},
	'C': {Code: 'C', ThreeLetterCode: "Cys", Name: "Cysteine", Hydrophobicity: 2.5, Charge: 0, MolWeight: 121.15, PKa: 8.18, Volume: 108.5},
	'Q': {Code: 'Q', ThreeLetterCode: "Gln", Name: "Glutamine", Hydrophobicity: -3.5, Charge: 0, MolWeight: 146.15, PKa: 0, Volume: 143.8},
	'E': {Code: 'E', ThreeLetterCode: "Glu", Name: "Glutamate", Hydrophobicity: -3.5, Charge: -1.0, MolWeight: 147.13, PKa: 4.25, Volume: 138.4},
	'G': {Code: 'G', ThreeLetterCode: "Gly", Name: "Glycine", Hydrophobicity: -0.4, Charge: 0, MolWeight: 75.07, PKa: 0, Volume: 60.1},
	'H': {Code: 'H', ThreeLetterCode: "His", Name: "Histidine", Hydrophobicity: -3.2, Charge: 0.1, MolWeight: 155.16, PKa: 6.00, Volume: 153.2},
	'I': {Code: 'I', ThreeLetterCode: "Ile", Name: "Isoleucine", Hydrophobicity: 4.5, Charge: 0, MolWeight: 131.17, PKa: 0, Volume: 166.7},
	'L': {Code: 'L', ThreeLetterCode: "Leu", Name: "Leucine", Hydrophobicity: 3.8, Charge: 0, MolWeight: 131.17, PKa: 0, Volume: 166.7},
	'K': {Code: 'K', ThreeLetterCode: "Lys", Name: "Lysine", Hydrophobicity: -3.9, Charge: 1.0, MolWeight: 146.19, PKa: 10.53, Volume: 168.6},
	'M': {Code: 'M', ThreeLetterCode: "Met", Name: "Methionine", Hydrophobicity: 1.9, Charge: 0, MolWeight: 149.21, PKa: 0, Volume: 162.9},
	'F': {Code: 'F', ThreeLetterCode: "Phe", Name: "Phenylalanine", Hydrophobicity: 2.8, Charge: 0, MolWeight: 165.19, PKa: 0, Volume: 189.9},
	'P': {Code: 'P', ThreeLetterCode: "Pro", Name: "Proline", Hydrophobicity: -1.6, Charge: 0, MolWeight: 115.13, PKa: 0, Volume: 112.7},
	'S': {Code: 'S', ThreeLetterCode: "Ser", Name: "Serine", Hydrophobicity: -0.8, Charge: 0, MolWeight: 105.09, PKa: 0, Volume: 89.0},
	'T': {Code: 'T', ThreeLetterCode: "Thr", Name: "Threonine", Hydrophobicity: -0.7, Charge: 0, MolWeight: 119.12, PKa: 0, Volume: 116.1},
	'W': {Code: 'W', ThreeLetterCode: "Trp", Name: "Tryptophan", Hydrophobicity: -0.9, Charge: 0, MolWeight: 204.23, PKa: 0, Volume: 227.8},
	'Y': {Code: 'Y', ThreeLetterCode: "Tyr", Name: "Tyrosine", Hydrophobicity: -1.3, Charge: 0, MolWeight: 181.19, PKa: 10.07, Volume: 193.6},
	'V': {Code: 'V', ThreeLetterCode: "Val", Name: "Valine", Hydrophobicity: 4.2, Charge: 0, MolWeight: 117.15, PKa: 0, Volume: 140.0},
}

var aminoAcidThreeLetter = map[string]byte{
	"ALA": 'A',
	"ARG": 'R',
	"ASN": 'N',
	"ASP": 'D',
	"CYS": 'C',
	"GLN": 'Q',
	"GLU": 'E',
	"GLY": 'G',
	"HIS": 'H',
	"ILE": 'I',
	"LEU": 'L',
	"LYS": 'K',
	"MET": 'M',
	"PHE": 'F',
	"PRO": 'P',
	"SER": 'S',
	"THR": 'T',
	"TRP": 'W',
	"TYR": 'Y',
	"VAL": 'V',
}

func lookupAminoAcid(code byte) (AminoAcid, bool) {
	aa, ok := aminoAcidTable[code]
	return aa, ok
}

func aminoAcidFromThreeLetter(code string) (AminoAcid, bool) {
	one, ok := aminoAcidThreeLetter[strings.ToUpper(strings.TrimSpace(code))]
	if !ok {
		return AminoAcid{}, false
	}
	return lookupAminoAcid(one)
}

func aminoAcidIdentityGroup(code byte) string {
	switch code {
	case 'A', 'I', 'L', 'M', 'F', 'W', 'V':
		return "hydrophobic"
	case 'K', 'R':
		return "positive"
	case 'D', 'E':
		return "negative"
	case 'N', 'Q', 'S', 'T':
		return "polar"
	case 'H', 'Y':
		return "aromatic"
	case 'G', 'P', 'C':
		return "special"
	default:
		return "unknown"
	}
}
