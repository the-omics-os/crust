package sequenceviewer

import (
	"strings"
	"unicode"
)

var dnaComplements = map[byte]byte{
	'A': 'T',
	'T': 'A',
	'U': 'A',
	'G': 'C',
	'C': 'G',
	'R': 'Y',
	'Y': 'R',
	'S': 'S',
	'W': 'W',
	'K': 'M',
	'M': 'K',
	'B': 'V',
	'D': 'H',
	'H': 'D',
	'V': 'B',
	'N': 'N',
}

var rnaComplements = map[byte]byte{
	'A': 'U',
	'T': 'A',
	'U': 'A',
	'G': 'C',
	'C': 'G',
	'R': 'Y',
	'Y': 'R',
	'S': 'S',
	'W': 'W',
	'K': 'M',
	'M': 'K',
	'B': 'V',
	'D': 'H',
	'H': 'D',
	'V': 'B',
	'N': 'N',
}

var dnaIUPAC = map[byte]string{
	'A': "A",
	'C': "C",
	'G': "G",
	'T': "T",
	'U': "T",
	'R': "AG",
	'Y': "CT",
	'S': "GC",
	'W': "AT",
	'K': "GT",
	'M': "AC",
	'B': "CGT",
	'D': "AGT",
	'H': "ACT",
	'V': "ACG",
	'N': "ACGT",
}

var rnaIUPAC = map[byte]string{
	'A': "A",
	'C': "C",
	'G': "G",
	'T': "U",
	'U': "U",
	'R': "AG",
	'Y': "CU",
	'S': "GC",
	'W': "AU",
	'K': "GU",
	'M': "AC",
	'B': "CGU",
	'D': "AGU",
	'H': "ACU",
	'V': "ACG",
	'N': "ACGU",
}

// NormalizeSequence removes whitespace and uppercases a biological sequence.
func NormalizeSequence(seq string) string {
	var b strings.Builder
	b.Grow(len(seq))
	for _, r := range seq {
		switch {
		case unicode.IsSpace(r):
			continue
		case unicode.IsDigit(r):
			continue
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

func isNucleotideCode(code byte) bool {
	_, ok := dnaIUPAC[code]
	return ok
}

// Complement returns the complement of a nucleotide code.
func Complement(code byte, t SequenceType) byte {
	code = byte(unicode.ToUpper(rune(code)))
	if t == RNA {
		if comp, ok := rnaComplements[code]; ok {
			return comp
		}
		return 'N'
	}
	if comp, ok := dnaComplements[code]; ok {
		return comp
	}
	return 'N'
}

// ReverseComplement returns the reverse-complement of a sequence.
func ReverseComplement(seq string, t SequenceType) string {
	normalized := NormalizeSequence(seq)
	if normalized == "" {
		return ""
	}
	out := make([]byte, len(normalized))
	for i := range normalized {
		out[len(normalized)-1-i] = Complement(normalized[i], t)
	}
	return string(out)
}

func ambiguitySet(code byte, t SequenceType) string {
	code = byte(unicode.ToUpper(rune(code)))
	if t == RNA {
		if set, ok := rnaIUPAC[code]; ok {
			return set
		}
		return ""
	}
	if set, ok := dnaIUPAC[code]; ok {
		return set
	}
	return ""
}

func gcProbability(code byte, t SequenceType) float64 {
	set := ambiguitySet(code, t)
	if set == "" {
		return 0
	}
	var gc int
	for i := 0; i < len(set); i++ {
		if set[i] == 'G' || set[i] == 'C' {
			gc++
		}
	}
	return float64(gc) / float64(len(set))
}

func meltingContribution(code byte, t SequenceType) float64 {
	gc := gcProbability(code, t)
	return 2 + 2*gc
}

func inferSequenceTypeFromSequence(seq string) SequenceType {
	normalized := NormalizeSequence(seq)
	if normalized == "" {
		return SequenceUnknown
	}
	allNucleotide := true
	hasU := false
	for i := 0; i < len(normalized); i++ {
		code := normalized[i]
		if code == 'U' {
			hasU = true
		}
		if !isNucleotideCode(code) {
			allNucleotide = false
			break
		}
	}
	if !allNucleotide {
		return Protein
	}
	if hasU && !strings.ContainsRune(normalized, 'T') {
		return RNA
	}
	return DNA
}

func inferSequenceTypeFromResidues(residues []Residue) SequenceType {
	if len(residues) == 0 {
		return SequenceUnknown
	}

	hasProteinProps := false
	hasNucleotideProps := false
	hasU := false
	var b strings.Builder
	b.Grow(len(residues))
	for _, residue := range residues {
		code := byte(unicode.ToUpper(rune(residue.Code)))
		b.WriteByte(code)
		if code == 'U' {
			hasU = true
		}
		if residue.Properties.MolWeight > 0 || residue.Properties.Volume > 0 || residue.Properties.Hydrophobicity != 0 || residue.Properties.Charge != 0 || residue.Properties.PKa != 0 {
			hasProteinProps = true
		}
		if residue.Properties.MeltingContrib > 0 || residue.Properties.GCWindow > 0 {
			hasNucleotideProps = true
		}
	}

	sequenceType := inferSequenceTypeFromSequence(b.String())
	if sequenceType == DNA && hasProteinProps && !hasNucleotideProps {
		return Protein
	}
	if sequenceType == SequenceUnknown && hasProteinProps && !hasNucleotideProps {
		return Protein
	}
	if sequenceType == DNA && hasU {
		return RNA
	}
	return sequenceType
}

func defaultResiduesPerLine(t SequenceType) int {
	switch t {
	case Protein:
		return 50
	case DNA, RNA:
		return 30
	default:
		return 30
	}
}

func groupSizeForType(t SequenceType) int {
	switch t {
	case Protein:
		return 10
	default:
		return 3
	}
}

func sequenceFromResidues(residues []Residue) string {
	if len(residues) == 0 {
		return ""
	}
	out := make([]byte, len(residues))
	for i, residue := range residues {
		out[i] = byte(unicode.ToUpper(rune(residue.Code)))
	}
	return string(out)
}
