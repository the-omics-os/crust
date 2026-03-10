package moleculeviewer

import "testing"

func TestParseSMILES_Ethanol(t *testing.T) {
	mol, err := ParseSMILES("CCO")
	if err != nil {
		t.Fatalf("ParseSMILES returned error: %v", err)
	}
	if len(mol.Atoms) != 3 {
		t.Fatalf("expected 3 atoms, got %d", len(mol.Atoms))
	}
	if len(mol.Bonds) != 2 {
		t.Fatalf("expected 2 bonds, got %d", len(mol.Bonds))
	}
	if got := mol.Formula(); got != "C2H6O" {
		t.Fatalf("expected formula C2H6O, got %q", got)
	}
	if mol.Atoms[2].PartialCharge >= 0 {
		t.Fatalf("expected oxygen atom to carry a negative partial charge, got %.2f", mol.Atoms[2].PartialCharge)
	}
}

func TestParseSMILES_AromaticRing(t *testing.T) {
	mol, err := ParseSMILES("c1ccccc1O")
	if err != nil {
		t.Fatalf("ParseSMILES returned error: %v", err)
	}
	if len(mol.Atoms) != 7 {
		t.Fatalf("expected 7 atoms, got %d", len(mol.Atoms))
	}
	aromatic := 0
	for _, atom := range mol.Atoms {
		if atom.Aromatic {
			aromatic++
		}
	}
	if aromatic != 6 {
		t.Fatalf("expected 6 aromatic atoms, got %d", aromatic)
	}
}

func TestParseSMILES_BracketCharge(t *testing.T) {
	mol, err := ParseSMILES("[NH4+]")
	if err != nil {
		t.Fatalf("ParseSMILES returned error: %v", err)
	}
	if len(mol.Atoms) != 1 {
		t.Fatalf("expected 1 atom, got %d", len(mol.Atoms))
	}
	if mol.Atoms[0].Charge != 1 {
		t.Fatalf("expected +1 charge, got %d", mol.Atoms[0].Charge)
	}
	if mol.Atoms[0].Hydrogens != 4 {
		t.Fatalf("expected 4 hydrogens, got %d", mol.Atoms[0].Hydrogens)
	}
}

func TestParseMOL(t *testing.T) {
	mol, err := ParseMOL(sampleMolBlock())
	if err != nil {
		t.Fatalf("ParseMOL returned error: %v", err)
	}
	if got := mol.Name; got != "Ethanol" {
		t.Fatalf("expected name Ethanol, got %q", got)
	}
	if got := mol.Formula(); got != "C2H6O" {
		t.Fatalf("expected formula C2H6O, got %q", got)
	}
	if !mol.HasCoordinates() {
		t.Fatal("expected coordinates from MOL block to be preserved")
	}
}

func TestMoleculeSearch(t *testing.T) {
	mol, err := ParseSMILES("CCO")
	if err != nil {
		t.Fatalf("ParseSMILES returned error: %v", err)
	}
	result := mol.Search("hydroxyl")
	if len(result.AtomIndices) == 0 {
		t.Fatalf("expected hydroxyl search to return atoms, got %+v", result)
	}
	foundOxygen := false
	for _, idx := range result.AtomIndices {
		if mol.Atoms[idx].Symbol == "O" {
			foundOxygen = true
			break
		}
	}
	if !foundOxygen {
		t.Fatalf("expected hydroxyl search to include oxygen, got %+v", result)
	}
	if len(result.Groups) != 1 || result.Groups[0] != "hydroxyl" {
		t.Fatalf("expected hydroxyl group annotation, got %+v", result.Groups)
	}

	result = mol.Search("O3")
	if len(result.AtomIndices) != 1 || result.AtomIndices[0] != 2 {
		t.Fatalf("expected indexed atom search to find atom 3, got %+v", result)
	}
}

func sampleMolBlock() string {
	return `Ethanol
  Codex

  3  2  0  0  0  0            999 V2000
    0.0000    0.0000    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    1.2094    0.0000    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    2.4188    0.0000    0.0000 O   0  0  0  0  0  0  0  0  0  0  0  0
  1  2  1  0  0  0  0
  2  3  1  0  0  0  0
M  END
`
}
