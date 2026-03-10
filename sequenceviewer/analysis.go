package sequenceviewer

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

var commonRestrictionEnzymes = []RestrictionEnzyme{
	{Name: "EcoRI", Site: "GAATTC"},
	{Name: "BamHI", Site: "GGATCC"},
	{Name: "HindIII", Site: "AAGCTT"},
	{Name: "NotI", Site: "GCGGCCGC"},
	{Name: "XhoI", Site: "CTCGAG"},
	{Name: "NheI", Site: "GCTAGC"},
	{Name: "SpeI", Site: "ACTAGT"},
	{Name: "SalI", Site: "GTCGAC"},
	{Name: "KpnI", Site: "GGTACC"},
	{Name: "SacI", Site: "GAGCTC"},
	{Name: "PstI", Site: "CTGCAG"},
	{Name: "BglII", Site: "AGATCT"},
	{Name: "NcoI", Site: "CCATGG"},
	{Name: "NdeI", Site: "CATATG"},
	{Name: "XbaI", Site: "TCTAGA"},
	{Name: "SmaI", Site: "CCCGGG"},
	{Name: "ApaI", Site: "GGGCCC"},
	{Name: "ClaI", Site: "ATCGAT"},
	{Name: "EcoRV", Site: "GATATC"},
	{Name: "MluI", Site: "ACGCGT"},
}

// GCContent returns the local GC fraction for each position in a nucleotide sequence.
func GCContent(residues []Residue, windowSize int) []float64 {
	if len(residues) == 0 {
		return nil
	}
	if windowSize <= 0 {
		windowSize = 1
	}
	if windowSize > len(residues) {
		windowSize = len(residues)
	}
	seqType := inferSequenceTypeFromResidues(residues)
	if !seqType.IsNucleotide() {
		return make([]float64, len(residues))
	}

	values := make([]float64, len(residues))
	half := windowSize / 2
	for i := range residues {
		start := i - half
		end := start + windowSize
		if start < 0 {
			start = 0
			end = windowSize
		}
		if end > len(residues) {
			end = len(residues)
			start = end - windowSize
			if start < 0 {
				start = 0
			}
		}
		var gc float64
		for j := start; j < end; j++ {
			gc += gcProbability(residues[j].Code, seqType)
		}
		values[i] = gc / float64(end-start)
	}
	return values
}

func overallGCContent(residues []Residue) float64 {
	if len(residues) == 0 {
		return 0
	}
	seqType := inferSequenceTypeFromResidues(residues)
	if !seqType.IsNucleotide() {
		return 0
	}
	var gc float64
	for _, residue := range residues {
		gc += gcProbability(residue.Code, seqType)
	}
	return gc / float64(len(residues))
}

// FindORFs finds forward-strand ORFs in a DNA sequence.
func FindORFs(residues []Residue, minLength int) []ORF {
	seqType := inferSequenceTypeFromResidues(residues)
	if seqType != DNA || len(residues) < 3 {
		return nil
	}
	if minLength <= 0 {
		minLength = 1
	}

	sequence := strings.ReplaceAll(sequenceFromResidues(residues), "U", "T")
	var orfs []ORF
	stops := map[string]struct{}{
		"TAA": {},
		"TAG": {},
		"TGA": {},
	}

	for frame := 0; frame < 3; frame++ {
		for i := frame; i <= len(sequence)-3; i += 3 {
			if sequence[i:i+3] != "ATG" {
				continue
			}
			for j := i + 3; j <= len(sequence)-3; j += 3 {
				codon := sequence[j : j+3]
				if _, ok := stops[codon]; !ok {
					continue
				}
				aaLength := (j - i) / 3
				if aaLength < minLength {
					break
				}
				start := i + 1
				end := j + 3
				orfs = append(orfs, ORF{
					Frame:    frame + 1,
					Start:    start,
					End:      end,
					Length:   end - start + 1,
					Sequence: sequence[i:end],
				})
				break
			}
		}
	}

	return orfs
}

// CommonRestrictionEnzymes returns a copy of the built-in enzyme panel.
func CommonRestrictionEnzymes() []RestrictionEnzyme {
	return append([]RestrictionEnzyme(nil), commonRestrictionEnzymes...)
}

// FindRestrictionSites locates all motif matches for the provided enzymes.
func FindRestrictionSites(residues []Residue, enzymes []RestrictionEnzyme) []RestrictionSite {
	seqType := inferSequenceTypeFromResidues(residues)
	if seqType != DNA {
		return nil
	}
	sequence := strings.ReplaceAll(sequenceFromResidues(residues), "U", "T")
	if len(enzymes) == 0 {
		enzymes = commonRestrictionEnzymes
	}

	var sites []RestrictionSite
	for _, enzyme := range enzymes {
		motif := strings.ToUpper(strings.TrimSpace(enzyme.Site))
		if motif == "" {
			continue
		}
		searchFrom := 0
		for {
			idx := strings.Index(sequence[searchFrom:], motif)
			if idx < 0 {
				break
			}
			start := searchFrom + idx + 1
			end := start + len(motif) - 1
			sites = append(sites, RestrictionSite{
				Enzyme:   enzyme,
				Start:    start,
				End:      end,
				Sequence: motif,
			})
			searchFrom = searchFrom + idx + 1
		}
	}

	sort.Slice(sites, func(i, j int) bool {
		if sites[i].Start != sites[j].Start {
			return sites[i].Start < sites[j].Start
		}
		return sites[i].Enzyme.Name < sites[j].Enzyme.Name
	})

	return sites
}

// EstimateTm estimates melting temperature with the Wallace rule.
func EstimateTm(residues []Residue) float64 {
	if len(residues) == 0 {
		return 0
	}
	seqType := inferSequenceTypeFromResidues(residues)
	if !seqType.IsNucleotide() {
		return 0
	}
	var tm float64
	for _, residue := range residues {
		tm += meltingContribution(residue.Code, seqType)
	}
	return tm
}

// EstimatePI estimates the isoelectric point by bisection.
func EstimatePI(residues []Residue) float64 {
	if len(residues) == 0 {
		return 0
	}
	seqType := inferSequenceTypeFromResidues(residues)
	if seqType != Protein {
		return 0
	}

	low, high := 0.0, 14.0
	for i := 0; i < 64; i++ {
		mid := (low + high) / 2
		charge := netChargeAtPH(residues, mid)
		if charge > 0 {
			low = mid
		} else {
			high = mid
		}
	}
	return math.Round(((low+high)/2)*100) / 100
}

func netChargeAtPH(residues []Residue, pH float64) float64 {
	if len(residues) == 0 {
		return 0
	}

	charge := basicCharge(9.69, pH) - acidicCharge(2.34, pH)
	for _, residue := range residues {
		code := byte(unicode.ToUpper(rune(residue.Code)))
		switch code {
		case 'K':
			charge += basicCharge(10.53, pH)
		case 'R':
			charge += basicCharge(12.48, pH)
		case 'H':
			charge += basicCharge(6.00, pH)
		case 'D':
			charge -= acidicCharge(3.65, pH)
		case 'E':
			charge -= acidicCharge(4.25, pH)
		case 'C':
			charge -= acidicCharge(8.18, pH)
		case 'Y':
			charge -= acidicCharge(10.07, pH)
		}
	}
	return charge
}

func basicCharge(pKa, pH float64) float64 {
	return 1 / (1 + math.Pow(10, pH-pKa))
}

func acidicCharge(pKa, pH float64) float64 {
	return 1 / (1 + math.Pow(10, pKa-pH))
}

func buildResiduesFromSequence(seq string, seqType SequenceType, gcWindow int) []Residue {
	normalized := NormalizeSequence(seq)
	residues := make([]Residue, 0, len(normalized))
	for i := 0; i < len(normalized); i++ {
		residues = append(residues, Residue{
			Position: i + 1,
			Code:     byte(unicode.ToUpper(rune(normalized[i]))),
		})
	}
	return enrichResidues(residues, seqType, gcWindow)
}

func enrichResidues(residues []Residue, seqType SequenceType, gcWindow int) []Residue {
	if len(residues) == 0 {
		return nil
	}
	if seqType == SequenceUnknown {
		seqType = inferSequenceTypeFromResidues(residues)
	}

	enriched := make([]Residue, len(residues))
	for i, residue := range residues {
		copyResidue := residue
		copyResidue.Code = byte(unicode.ToUpper(rune(copyResidue.Code)))
		if copyResidue.Position <= 0 {
			copyResidue.Position = i + 1
		}
		copyResidue.Bonds = append([]int(nil), residue.Bonds...)

		switch seqType {
		case Protein:
			if aa, ok := lookupAminoAcid(copyResidue.Code); ok {
				copyResidue.Properties.Hydrophobicity = aa.Hydrophobicity
				copyResidue.Properties.Charge = aa.Charge
				copyResidue.Properties.MolWeight = aa.MolWeight
				copyResidue.Properties.PKa = aa.PKa
				copyResidue.Properties.Volume = aa.Volume
			}
		case DNA, RNA:
			copyResidue.Properties.MeltingContrib = meltingContribution(copyResidue.Code, seqType)
		}

		enriched[i] = copyResidue
	}

	if seqType.IsNucleotide() {
		gc := GCContent(enriched, gcWindow)
		for i := range enriched {
			enriched[i].Properties.GCWindow = gc[i]
		}
	}

	return enriched
}

func totalMolecularWeight(residues []Residue) float64 {
	var total float64
	for _, residue := range residues {
		total += residue.Properties.MolWeight
	}
	return total
}
