package sequenceviewer

import (
	"math"
	"testing"
)

func closeFloat(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

func TestGCContentKnownValues(t *testing.T) {
	residues := buildResiduesFromSequence("ATGCG", DNA, 3)
	got := GCContent(residues, 3)
	want := []float64{1.0 / 3.0, 1.0 / 3.0, 2.0 / 3.0, 1.0, 1.0}
	if len(got) != len(want) {
		t.Fatalf("expected %d values, got %d", len(want), len(got))
	}
	for i := range want {
		if !closeFloat(got[i], want[i]) {
			t.Fatalf("value %d = %f, want %f", i, got[i], want[i])
		}
	}
}

func TestFindORFsKnownSequence(t *testing.T) {
	residues := buildResiduesFromSequence("AAATGAAATAGCCC", DNA, 5)
	orfs := FindORFs(residues, 2)
	if len(orfs) != 1 {
		t.Fatalf("expected 1 ORF, got %d", len(orfs))
	}
	if orfs[0].Frame != 3 || orfs[0].Start != 3 || orfs[0].End != 11 {
		t.Fatalf("unexpected ORF %+v", orfs[0])
	}
}

func TestFindRestrictionSitesKnownSequence(t *testing.T) {
	residues := buildResiduesFromSequence("AAGAATTCGG", DNA, 5)
	sites := FindRestrictionSites(residues, []RestrictionEnzyme{{Name: "EcoRI", Site: "GAATTC"}})
	if len(sites) != 1 {
		t.Fatalf("expected 1 restriction site, got %d", len(sites))
	}
	if sites[0].Start != 3 || sites[0].End != 8 {
		t.Fatalf("unexpected site %+v", sites[0])
	}
}

func TestEstimateTm(t *testing.T) {
	residues := buildResiduesFromSequence("ATGC", DNA, 4)
	if got := EstimateTm(residues); !closeFloat(got, 12) {
		t.Fatalf("expected Tm 12, got %f", got)
	}
}

func TestEstimatePI(t *testing.T) {
	basic := buildResiduesFromSequence("KKK", Protein, 5)
	acidic := buildResiduesFromSequence("DDD", Protein, 5)

	if got := EstimatePI(basic); got <= 9 {
		t.Fatalf("expected basic pI > 9, got %f", got)
	}
	if got := EstimatePI(acidic); got >= 4.5 {
		t.Fatalf("expected acidic pI < 4.5, got %f", got)
	}
}

func TestCommonRestrictionEnzymesContainsEcoRI(t *testing.T) {
	enzymes := CommonRestrictionEnzymes()
	found := false
	for _, enzyme := range enzymes {
		if enzyme.Name == "EcoRI" && enzyme.Site == "GAATTC" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected EcoRI in common enzyme list")
	}
}
