package moleculeviewer

import (
	"math"
	"sort"
)

// LayoutMolecule generates or normalizes 2D coordinates for the molecule.
func LayoutMolecule(m *Molecule) {
	if m == nil || len(m.Atoms) == 0 {
		return
	}

	m.Normalize()

	if m.HasCoordinates() {
		normalizeCoordinates(m)
		return
	}

	n := len(m.Atoms)
	coords := make([][2]float64, n)
	placed := make([]bool, n)
	visited := make([]bool, n)
	xOffset := 0.0

	for root := 0; root < n; root++ {
		if visited[root] {
			continue
		}

		component := []int{}
		componentNodes := collectComponent(m, root, visited)
		if len(componentNodes) == 0 {
			continue
		}

		coords[root] = [2]float64{0, 0}
		placeTree(m, root, -1, 0, coords, placed, &component)

		minX, maxX := componentBounds(coords, component)
		shift := xOffset - minX
		for _, idx := range component {
			coords[idx][0] += shift
		}
		xOffset += (maxX - minX) + 3.0
	}

	relaxCoordinates(m, coords)
	for i := range m.Atoms {
		m.Atoms[i].Coords = coords[i]
	}
	normalizeCoordinates(m)
}

func placeTree(m *Molecule, atomIdx, parent int, incomingAngle float64, coords [][2]float64, placed []bool, component *[]int) {
	if placed[atomIdx] {
		return
	}
	placed[atomIdx] = true
	*component = append(*component, atomIdx)

	var children []int
	for _, neighbor := range m.Atoms[atomIdx].Neighbors {
		if neighbor == parent || placed[neighbor] {
			continue
		}
		children = append(children, neighbor)
	}

	sort.Slice(children, func(i, j int) bool {
		left := m.Atoms[children[i]]
		right := m.Atoms[children[j]]
		if left.Aromatic != right.Aromatic {
			return left.Aromatic
		}
		return len(left.Neighbors) > len(right.Neighbors)
	})

	angles := childAngles(len(children), parent < 0, incomingAngle)
	for i, child := range children {
		angle := angles[minInt(i, len(angles)-1)]
		bondLength := 1.0 - 0.08*float64(maxInt(0, m.bondOrderBetween(atomIdx, child)-1))
		coords[child][0] = coords[atomIdx][0] + math.Cos(angle)*bondLength
		coords[child][1] = coords[atomIdx][1] + math.Sin(angle)*bondLength
		placeTree(m, child, atomIdx, angle, coords, placed, component)
	}
}

func childAngles(count int, isRoot bool, incomingAngle float64) []float64 {
	if count <= 0 {
		return nil
	}
	if isRoot {
		switch count {
		case 1:
			return []float64{0}
		case 2:
			return []float64{-math.Pi / 6, math.Pi / 6}
		case 3:
			return []float64{0, 2 * math.Pi / 3, -2 * math.Pi / 3}
		default:
			out := make([]float64, count)
			step := 2 * math.Pi / float64(count)
			for i := range out {
				out[i] = float64(i) * step
			}
			return out
		}
	}

	offsets := []float64{0, math.Pi / 3, -math.Pi / 3, 2 * math.Pi / 3, -2 * math.Pi / 3, math.Pi}
	out := make([]float64, count)
	for i := range out {
		offset := offsets[minInt(i, len(offsets)-1)]
		out[i] = incomingAngle + offset
	}
	return out
}

func collectComponent(m *Molecule, root int, visited []bool) []int {
	stack := []int{root}
	visited[root] = true
	var component []int
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		component = append(component, idx)
		for _, neighbor := range m.Atoms[idx].Neighbors {
			if visited[neighbor] {
				continue
			}
			visited[neighbor] = true
			stack = append(stack, neighbor)
		}
	}
	return component
}

func componentBounds(coords [][2]float64, atoms []int) (float64, float64) {
	if len(atoms) == 0 {
		return 0, 0
	}
	minX := coords[atoms[0]][0]
	maxX := coords[atoms[0]][0]
	for _, idx := range atoms[1:] {
		minX = math.Min(minX, coords[idx][0])
		maxX = math.Max(maxX, coords[idx][0])
	}
	return minX, maxX
}

func relaxCoordinates(m *Molecule, coords [][2]float64) {
	if len(coords) <= 1 {
		return
	}

	const (
		iterations = 220
		springK    = 0.18
		repelK     = 0.015
		step       = 0.08
	)

	for iter := 0; iter < iterations; iter++ {
		forces := make([][2]float64, len(coords))

		for i := 0; i < len(coords); i++ {
			for j := i + 1; j < len(coords); j++ {
				dx := coords[j][0] - coords[i][0]
				dy := coords[j][1] - coords[i][1]
				dist2 := dx*dx + dy*dy
				if dist2 < 1e-6 {
					dx = 0.01 * float64((j-i)%3+1)
					dy = 0.01 * float64((i+j)%3+1)
					dist2 = dx*dx + dy*dy
				}
				dist := math.Sqrt(dist2)
				repel := repelK / dist2
				rx := dx / dist * repel
				ry := dy / dist * repel
				forces[i][0] -= rx
				forces[i][1] -= ry
				forces[j][0] += rx
				forces[j][1] += ry
			}
		}

		for _, bond := range m.Bonds {
			dx := coords[bond.To][0] - coords[bond.From][0]
			dy := coords[bond.To][1] - coords[bond.From][1]
			dist := math.Hypot(dx, dy)
			if dist < 1e-6 {
				dist = 1e-6
			}
			target := 1.0 - 0.08*float64(maxInt(0, bond.Order-1))
			if bond.Aromatic {
				target = 0.94
			}
			pull := (dist - target) * springK
			px := dx / dist * pull
			py := dy / dist * pull
			forces[bond.From][0] += px
			forces[bond.From][1] += py
			forces[bond.To][0] -= px
			forces[bond.To][1] -= py
		}

		for i := range coords {
			coords[i][0] += forces[i][0] * step
			coords[i][1] += forces[i][1] * step
		}
	}
}

func normalizeCoordinates(m *Molecule) {
	if len(m.Atoms) == 0 {
		return
	}

	if len(m.Atoms) == 1 {
		m.Atoms[0].Coords = [2]float64{0, 0}
		return
	}

	avgBond := 0.0
	for _, bond := range m.Bonds {
		dx := m.Atoms[bond.To].Coords[0] - m.Atoms[bond.From].Coords[0]
		dy := m.Atoms[bond.To].Coords[1] - m.Atoms[bond.From].Coords[1]
		avgBond += math.Hypot(dx, dy)
	}
	if len(m.Bonds) > 0 {
		avgBond /= float64(len(m.Bonds))
	}
	if avgBond < 1e-6 {
		avgBond = 1.0
	}

	minX, maxX := m.Atoms[0].Coords[0], m.Atoms[0].Coords[0]
	minY, maxY := m.Atoms[0].Coords[1], m.Atoms[0].Coords[1]
	for i := range m.Atoms {
		m.Atoms[i].Coords[0] /= avgBond
		m.Atoms[i].Coords[1] /= avgBond
		minX = math.Min(minX, m.Atoms[i].Coords[0])
		maxX = math.Max(maxX, m.Atoms[i].Coords[0])
		minY = math.Min(minY, m.Atoms[i].Coords[1])
		maxY = math.Max(maxY, m.Atoms[i].Coords[1])
	}

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	for i := range m.Atoms {
		m.Atoms[i].Coords[0] -= centerX
		m.Atoms[i].Coords[1] -= centerY
	}

	if (maxY - minY) > (maxX-minX)*1.4 {
		for i := range m.Atoms {
			x := m.Atoms[i].Coords[0]
			y := m.Atoms[i].Coords[1]
			m.Atoms[i].Coords[0] = -y
			m.Atoms[i].Coords[1] = x
		}
	}
}

func (m Molecule) bondOrderBetween(a, b int) int {
	bond, ok := m.BondBetween(a, b)
	if !ok {
		return 1
	}
	return maxInt(1, bond.Order)
}
