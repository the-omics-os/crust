package moleculeviewer

import (
	"math"
	"testing"
)

func TestNicotineProjectionHasSeparatedRingSystems(t *testing.T) {
	mol, err := ParseMolBlock(nicotineMolBlockForScratch)
	if err != nil {
		t.Fatalf("ParseMolBlock: %v", err)
	}
	mol = mol.WithoutHydrogens()
	mol.finalize()

	linker := math.Hypot(
		mol.Atoms[6].Coords[0]-mol.Atoms[2].Coords[0],
		mol.Atoms[6].Coords[1]-mol.Atoms[2].Coords[1],
	)
	if linker < 1.35 {
		t.Fatalf("expected inter-ring linker bond to be opened for readability, got length %.2f", linker)
	}

	m := New(WithMolecule(mol), WithWidth(96), WithHeight(26))
	proj := m.projectAtoms(78, 14)
	if proj.collisions != 0 {
		t.Fatalf("expected nicotine projection to be collision-free, got %d collisions", proj.collisions)
	}

	// Ring centroids should not collapse onto the same terminal row.
	topRingY := averageY(proj.positions, []int{0, 2, 3, 4, 5})
	bottomRingY := averageY(proj.positions, []int{1, 6, 8, 9, 10, 11})
	if math.Abs(topRingY-bottomRingY) < 2 {
		t.Fatalf("expected ring systems to separate on screen, got centroid rows %.2f and %.2f", topRingY, bottomRingY)
	}
}

func averageY(positions [][2]int, atoms []int) float64 {
	total := 0.0
	for _, idx := range atoms {
		total += float64(positions[idx][1])
	}
	return total / float64(len(atoms))
}

const nicotineMolBlockForScratch = `89594
  -OEChem-03092621202D

 26 27  0     1  0  0  0  0  0999 V2000
    2.9511    1.3184    0.0000 N   0  0  3  0  0  0  0  0  0  0  0  0
    4.6261   -1.7694    0.0000 N   0  0  0  0  0  0  0  0  0  0  0  0
    3.7601    0.7306    0.0000 C   0  0  1  0  0  0  0  0  0  0  0  0
    4.5691    1.3184    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    4.2601    2.2694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    3.2601    2.2694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    3.7601   -0.2694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    2.0000    1.0094    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    2.8941   -0.7694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    4.6261   -0.7694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    2.8941   -1.7694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    3.7601   -2.2694    0.0000 C   0  0  0  0  0  0  0  0  0  0  0  0
    4.3125    0.4491    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    4.8791    0.7814    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    5.1355    1.5705    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    4.8665    2.3983    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    4.1953    2.8860    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    3.3249    2.8860    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    2.6536    2.3983    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    1.8084    1.5990    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    1.4103    0.8178    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    2.1916    0.4197    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    2.3571   -0.4594    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    5.1630   -0.4594    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    2.3571   -2.0794    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
    3.7601   -2.8894    0.0000 H   0  0  0  0  0  0  0  0  0  0  0  0
  1  3  1  0  0  0  0
  1  6  1  0  0  0  0
  1  8  1  0  0  0  0
  2 10  1  0  0  0  0
  2 12  2  0  0  0  0
  3  4  1  0  0  0  0
  3  7  1  1  0  0  0
  3 13  1  0  0  0  0
  4  5  1  0  0  0  0
  4 14  1  0  0  0  0
  4 15  1  0  0  0  0
  5  6  1  0  0  0  0
  5 16  1  0  0  0  0
  5 17  1  0  0  0  0
  6 18  1  0  0  0  0
  6 19  1  0  0  0  0
  7  9  1  0  0  0  0
  7 10  2  0  0  0  0
  8 20  1  0  0  0  0
  8 21  1  0  0  0  0
  8 22  1  0  0  0  0
  9 11  2  0  0  0  0
  9 23  1  0  0  0  0
 10 24  1  0  0  0  0
 11 12  1  0  0  0  0
 11 25  1  0  0  0  0
 12 26  1  0  0  0  0
M  END
`
