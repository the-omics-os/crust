package sequenceviewer

import "testing"

func TestAminoAcidTableHasTwentyResidues(t *testing.T) {
	if len(aminoAcidTable) != 20 {
		t.Fatalf("expected 20 amino acids, got %d", len(aminoAcidTable))
	}
}

func TestKnownHydrophobicityValues(t *testing.T) {
	ile, ok := lookupAminoAcid('I')
	if !ok {
		t.Fatal("expected isoleucine in amino acid table")
	}
	if ile.Hydrophobicity != 4.5 {
		t.Fatalf("expected isoleucine hydrophobicity 4.5, got %f", ile.Hydrophobicity)
	}

	arg, ok := lookupAminoAcid('R')
	if !ok {
		t.Fatal("expected arginine in amino acid table")
	}
	if arg.Hydrophobicity != -4.5 {
		t.Fatalf("expected arginine hydrophobicity -4.5, got %f", arg.Hydrophobicity)
	}
}

func TestHydrophobicityRange(t *testing.T) {
	for code, aa := range aminoAcidTable {
		if aa.Hydrophobicity < minHydrophobicity || aa.Hydrophobicity > maxHydrophobicity {
			t.Fatalf("amino acid %c has hydrophobicity out of range: %f", code, aa.Hydrophobicity)
		}
	}
}

func TestThreeLetterLookup(t *testing.T) {
	aa, ok := aminoAcidFromThreeLetter("Lys")
	if !ok {
		t.Fatal("expected Lys lookup to succeed")
	}
	if aa.Code != 'K' {
		t.Fatalf("expected lysine code K, got %c", aa.Code)
	}
}
