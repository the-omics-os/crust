package moleculeviewer

import (
	"math"
	"sort"
)

const (
	connectivityTooClose = 0.63
	connectivityBondTol  = 0.045
)

var covalentRadii = map[string]float64{
	"H":  0.40,
	"C":  0.76,
	"O":  0.66,
	"N":  0.71,
	"P":  1.07,
	"S":  1.05,
	"Se": 1.20,
	"K":  2.03,
	"Ca": 1.76,
	"Mg": 1.41,
	"Cl": 1.02,
	"Na": 1.66,
	"Cu": 1.32,
	"Zn": 1.22,
	"Co": 1.50,
	"Fe": 1.52,
	"Mn": 1.61,
	"Cr": 1.39,
	"Si": 1.11,
	"Be": 0.96,
	"F":  0.57,
	"Br": 1.20,
	"I":  1.39,
}

var maxPerceivedBonds = map[string]int{
	"H":  1,
	"C":  4,
	"O":  2,
	"F":  1,
	"Cl": 1,
	"Br": 1,
	"I":  1,
}

type perceivedBondCandidate struct {
	from int
	to   int
	dist float64
}

// RepairConnectivityFromCoords supplements missing bonds using source-space coordinates.
// It preserves existing bond orders/aromaticity and only adds single bonds that satisfy
// gochem-style covalent-radius heuristics.
func (m *Molecule) RepairConnectivityFromCoords() bool {
	if m == nil || len(m.Atoms) < 2 || !m.HasCoordinates() {
		return false
	}

	existing := make(map[[2]int]Bond, len(m.Bonds))
	degrees := make([]int, len(m.Atoms))
	for _, bond := range m.Bonds {
		key := bondKey(bond.From, bond.To)
		existing[key] = bond
		if bond.From >= 0 && bond.From < len(degrees) {
			degrees[bond.From]++
		}
		if bond.To >= 0 && bond.To < len(degrees) {
			degrees[bond.To]++
		}
	}

	candidates := perceiveBondCandidates(m)
	if len(candidates) == 0 {
		return false
	}

	added := false
	for _, candidate := range candidates {
		key := bondKey(candidate.from, candidate.to)
		if _, ok := existing[key]; ok {
			continue
		}
		if cappedOut(m.Atoms[candidate.from], degrees[candidate.from]) || cappedOut(m.Atoms[candidate.to], degrees[candidate.to]) {
			continue
		}
		m.Bonds = append(m.Bonds, Bond{
			From:  candidate.from,
			To:    candidate.to,
			Order: 1,
		})
		existing[key] = m.Bonds[len(m.Bonds)-1]
		degrees[candidate.from]++
		degrees[candidate.to]++
		added = true
	}

	return added
}

func perceiveBondCandidates(m *Molecule) []perceivedBondCandidate {
	candidates := make([]perceivedBondCandidate, 0, len(m.Atoms))
	for i := 0; i < len(m.Atoms); i++ {
		r1, ok := covalentRadii[m.Atoms[i].Symbol]
		if !ok {
			continue
		}
		for j := i + 1; j < len(m.Atoms); j++ {
			r2, ok := covalentRadii[m.Atoms[j].Symbol]
			if !ok {
				continue
			}
			dist := atomDistance(m.Atoms[i], m.Atoms[j])
			if dist <= connectivityTooClose || dist >= r1+r2+connectivityBondTol {
				continue
			}
			candidates = append(candidates, perceivedBondCandidate{
				from: i,
				to:   j,
				dist: dist,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		if math.Abs(candidates[i].dist-candidates[j].dist) < 1e-9 {
			if candidates[i].from == candidates[j].from {
				return candidates[i].to < candidates[j].to
			}
			return candidates[i].from < candidates[j].from
		}
		return candidates[i].dist < candidates[j].dist
	})
	return candidates
}

func atomDistance(a, b Atom) float64 {
	dx := a.Coords[0] - b.Coords[0]
	dy := a.Coords[1] - b.Coords[1]
	return math.Hypot(dx, dy)
}

func cappedOut(atom Atom, degree int) bool {
	max, ok := maxPerceivedBonds[atom.Symbol]
	return ok && max > 0 && degree >= max
}

func bondKey(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}
