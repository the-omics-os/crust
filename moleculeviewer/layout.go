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
		coords := make([][2]float64, len(m.Atoms))
		for i, atom := range m.Atoms {
			coords[i] = atom.Coords
		}
		stretchRingSystemLinkers(m, coords)
		for i := range m.Atoms {
			m.Atoms[i].Coords = coords[i]
		}
		normalizeCoordinates(m)
		return
	}

	n := len(m.Atoms)
	coords := make([][2]float64, n)
	placed := make([]bool, n)

	// Phase 1: Place ring atoms as regular polygons
	placeAllRings(m, coords, placed)

	// Phase 2: Place chain atoms extending outward from placed atoms
	visited := make([]bool, n)
	xOffset := 0.0
	for root := 0; root < n; root++ {
		if visited[root] {
			continue
		}
		component := collectComponent(m, root, visited)
		if len(component) == 0 {
			continue
		}

		// Find a starting atom for chain placement: prefer an already-placed ring atom
		start := -1
		for _, idx := range component {
			if placed[idx] {
				start = idx
				break
			}
		}
		if start < 0 {
			// No ring atoms in this component; place from the first atom
			start = component[0]
			coords[start] = [2]float64{0, 0}
			placed[start] = true
		}

		placeChains(m, start, coords, placed)

		// Shift disconnected components so they don't overlap
		compMinX, compMaxX := componentBounds(coords, component)
		if root > 0 || compMinX > 0 {
			shift := xOffset - compMinX
			for _, idx := range component {
				coords[idx][0] += shift
			}
			compMaxX += (xOffset - compMinX)
		}
		xOffset = compMaxX + 3.0
	}

	// Phase 3: Gentle force-directed refinement to resolve overlaps
	// Pin ring atoms to preserve their geometry, only relax chains
	ringSet := make([]bool, n)
	for _, ring := range m.Rings() {
		for _, idx := range ring.Atoms {
			ringSet[idx] = true
		}
	}
	stretchRingSystemLinkers(m, coords)
	relaxWithPins(m, coords, ringSet)

	for i := range m.Atoms {
		m.Atoms[i].Coords = coords[i]
	}
	normalizeCoordinates(m)
}

// placeAllRings places all ring systems as regular polygons with fused ring awareness.
func placeAllRings(m *Molecule, coords [][2]float64, placed []bool) {
	rings := m.Rings()
	if len(rings) == 0 {
		return
	}

	// Sort: smaller rings first (5-membered before 6-membered), then by first atom index
	sort.Slice(rings, func(i, j int) bool {
		if len(rings[i].Atoms) != len(rings[j].Atoms) {
			return len(rings[i].Atoms) < len(rings[j].Atoms)
		}
		return rings[i].Atoms[0] < rings[j].Atoms[0]
	})

	ringPlaced := make([]bool, len(rings))

	// Place the first ring centered at origin
	placeRingPolygon(rings[0].Atoms, coords, placed, [2]float64{0, 0}, math.Pi/2)
	ringPlaced[0] = true

	// Iteratively place fused rings that share an edge with an already-placed ring
	for changed := true; changed; {
		changed = false
		for i, ring := range rings {
			if ringPlaced[i] {
				continue
			}
			if placeFusedRing(ring.Atoms, coords, placed) {
				ringPlaced[i] = true
				changed = true
			}
		}
	}

	// Place any remaining isolated rings (not fused with already-placed ones)
	xOff := 0.0
	for i, ring := range rings {
		if ringPlaced[i] {
			continue
		}
		// Find X offset beyond all placed atoms
		for _, idx := range ring.Atoms {
			if placed[idx] {
				continue
			}
		}
		placeRingPolygon(ring.Atoms, coords, placed, [2]float64{xOff + 3, 0}, math.Pi/2)
		ringPlaced[i] = true
		for _, idx := range ring.Atoms {
			if coords[idx][0] > xOff {
				xOff = coords[idx][0]
			}
		}
	}
}

// placeRingPolygon places ring atoms as a regular polygon centered at center.
// startAngle is the angle of the first atom from center.
func placeRingPolygon(atoms []int, coords [][2]float64, placed []bool, center [2]float64, startAngle float64) {
	n := len(atoms)
	if n == 0 {
		return
	}
	// Radius chosen so that chord length (bond length) = 1.0
	// chord = 2 * R * sin(π/n), so R = 1 / (2 * sin(π/n))
	radius := 1.0 / (2 * math.Sin(math.Pi/float64(n)))
	step := 2 * math.Pi / float64(n)

	for i, idx := range atoms {
		angle := startAngle + float64(i)*step
		coords[idx] = [2]float64{
			center[0] + radius*math.Cos(angle),
			center[1] + radius*math.Sin(angle),
		}
		placed[idx] = true
	}
}

// placeFusedRing places a ring that shares an edge with already-placed atoms.
// Returns true if placement was successful (found shared edge).
func placeFusedRing(atoms []int, coords [][2]float64, placed []bool) bool {
	n := len(atoms)

	// Find a shared edge: two consecutive atoms in the ring that are both already placed
	sharedStart := -1
	for i := 0; i < n; i++ {
		a := atoms[i]
		b := atoms[(i+1)%n]
		if placed[a] && placed[b] {
			sharedStart = i
			break
		}
	}
	if sharedStart < 0 {
		return false
	}

	a := atoms[sharedStart]
	b := atoms[(sharedStart+1)%n]

	// Compute the center of the new ring polygon:
	// The shared edge midpoint, then extend perpendicular to the side AWAY from existing ring center
	midX := (coords[a][0] + coords[b][0]) / 2
	midY := (coords[a][1] + coords[b][1]) / 2
	edgeDX := coords[b][0] - coords[a][0]
	edgeDY := coords[b][1] - coords[a][1]
	edgeLen := math.Hypot(edgeDX, edgeDY)
	if edgeLen < 1e-9 {
		return false
	}

	// Perpendicular directions (two options)
	perpX := -edgeDY / edgeLen
	perpY := edgeDX / edgeLen

	// Choose the perpendicular direction that points AWAY from existing placed atoms
	// Compute the average position of already-placed ring atoms (other than the shared edge)
	avgX, avgY := 0.0, 0.0
	count := 0
	for _, idx := range atoms {
		if placed[idx] && idx != a && idx != b {
			avgX += coords[idx][0]
			avgY += coords[idx][1]
			count++
		}
	}

	// If no other atoms are placed, look at all placed atoms nearby
	if count == 0 {
		for i, p := range placed {
			if p && i != a && i != b {
				avgX += coords[i][0]
				avgY += coords[i][1]
				count++
			}
		}
	}

	if count > 0 {
		avgX /= float64(count)
		avgY /= float64(count)
		// Choose the perpendicular that points AWAY from the average
		toAvgX := avgX - midX
		toAvgY := avgY - midY
		dot := toAvgX*perpX + toAvgY*perpY
		if dot > 0 {
			perpX = -perpX
			perpY = -perpY
		}
	}

	// Distance from edge midpoint to polygon center
	radius := 1.0 / (2 * math.Sin(math.Pi/float64(n)))
	halfChord := edgeLen / 2
	apothem := math.Sqrt(math.Max(0, radius*radius-halfChord*halfChord))

	centerX := midX + perpX*apothem
	centerY := midY + perpY*apothem

	// Compute the angle from center to atom A
	startAngle := math.Atan2(coords[a][1]-centerY, coords[a][0]-centerX)
	step := 2 * math.Pi / float64(n)

	// Determine winding: A should be at position sharedStart, B at sharedStart+1
	angleToB := math.Atan2(coords[b][1]-centerY, coords[b][0]-centerX)
	expectedB := startAngle + step
	altB := startAngle - step

	diffCW := math.Abs(normalizeAngle(angleToB - expectedB))
	diffCCW := math.Abs(normalizeAngle(angleToB - altB))
	if diffCCW < diffCW {
		step = -step
	}

	// Place unplaced atoms
	for i, idx := range atoms {
		if placed[idx] {
			continue
		}
		offset := i - sharedStart
		if offset < 0 {
			offset += n
		}
		angle := startAngle + float64(offset)*step
		coords[idx] = [2]float64{
			centerX + radius*math.Cos(angle),
			centerY + radius*math.Sin(angle),
		}
		placed[idx] = true
	}
	return true
}

func normalizeAngle(a float64) float64 {
	for a > math.Pi {
		a -= 2 * math.Pi
	}
	for a < -math.Pi {
		a += 2 * math.Pi
	}
	return a
}

// placeChains places all unplaced atoms by BFS from all already-placed atoms.
func placeChains(m *Molecule, start int, coords [][2]float64, placed []bool) {
	type queueEntry struct {
		parent        int
		atom          int
		incomingAngle float64
	}

	// Seed BFS with ALL placed atoms so chain atoms off any ring atom are reached
	var queue []queueEntry
	for i, p := range placed {
		if p {
			queue = append(queue, queueEntry{parent: -1, atom: i, incomingAngle: 0})
		}
	}
	// Also add the start atom if not yet placed (for chain-only components)
	if !placed[start] {
		coords[start] = [2]float64{0, 0}
		placed[start] = true
		queue = append(queue, queueEntry{parent: -1, atom: start, incomingAngle: 0})
	}

	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]

		var children []int
		for _, neighbor := range m.Atoms[entry.atom].Neighbors {
			if !placed[neighbor] {
				children = append(children, neighbor)
			}
		}
		if len(children) == 0 {
			continue
		}

		sort.Slice(children, func(i, j int) bool {
			left := branchSize(m, children[i], entry.atom, placed)
			right := branchSize(m, children[j], entry.atom, placed)
			if left != right {
				return left > right
			}
			return len(m.Atoms[children[i]].Neighbors) > len(m.Atoms[children[j]].Neighbors)
		})

		outAngle := computeOutwardAngle(m, entry.atom, coords, placed, entry.incomingAngle)
		angles := chooseChainAngles(m, entry.atom, children, coords, placed, outAngle)
		for i, child := range children {
			if placed[child] {
				continue
			}
			angle := angles[minInt(i, len(angles)-1)]
			bondLength := 1.0 - 0.08*float64(maxInt(0, m.bondOrderBetween(entry.atom, child)-1))
			coords[child] = [2]float64{
				coords[entry.atom][0] + math.Cos(angle)*bondLength,
				coords[entry.atom][1] + math.Sin(angle)*bondLength,
			}
			placed[child] = true
			queue = append(queue, queueEntry{parent: entry.atom, atom: child, incomingAngle: angle})
		}
	}
}

// computeOutwardAngle finds the angle pointing away from already-placed neighbors.
func computeOutwardAngle(m *Molecule, atom int, coords [][2]float64, placed []bool, fallback float64) float64 {
	var occupied []float64
	for _, neighbor := range m.Atoms[atom].Neighbors {
		if placed[neighbor] {
			dx := coords[neighbor][0] - coords[atom][0]
			dy := coords[neighbor][1] - coords[atom][1]
			occupied = append(occupied, math.Atan2(dy, dx))
		}
	}
	if len(occupied) == 0 {
		return fallback
	}
	return openSectorAngle(occupied, fallback)
}

func chooseChainAngles(m *Molecule, atom int, children []int, coords [][2]float64, placed []bool, baseAngle float64) []float64 {
	if len(children) == 0 {
		return nil
	}

	candidates := candidateChainAngles(baseAngle)
	var occupied []float64
	for _, neighbor := range m.Atoms[atom].Neighbors {
		if !placed[neighbor] {
			continue
		}
		dx := coords[neighbor][0] - coords[atom][0]
		dy := coords[neighbor][1] - coords[atom][1]
		occupied = append(occupied, math.Atan2(dy, dx))
	}

	chosen := make([]float64, 0, len(children))
	for idx := range children {
		bestAngle := baseAngle
		bestScore := math.Inf(1)
		for _, candidate := range candidates {
			score := scoreChainAngle(candidate, coords[atom], coords, placed, occupied, chosen)
			if idx == 0 {
				score += angularDistance(candidate, baseAngle) * 0.45
			}
			if score < bestScore {
				bestScore = score
				bestAngle = candidate
			}
		}
		chosen = append(chosen, bestAngle)
	}
	return chosen
}

func candidateChainAngles(baseAngle float64) []float64 {
	offsets := []float64{
		0,
		math.Pi / 3, -math.Pi / 3,
		math.Pi / 2, -math.Pi / 2,
		2 * math.Pi / 3, -2 * math.Pi / 3,
		math.Pi, math.Pi / 6, -math.Pi / 6,
		5 * math.Pi / 6, -5 * math.Pi / 6,
	}
	out := make([]float64, len(offsets))
	for i, offset := range offsets {
		out[i] = normalizeAngle(baseAngle + offset)
	}
	return out
}

func scoreChainAngle(candidate float64, origin [2]float64, coords [][2]float64, placed []bool, occupied, chosen []float64) float64 {
	target := [2]float64{
		origin[0] + math.Cos(candidate),
		origin[1] + math.Sin(candidate),
	}

	score := 0.0
	for i, isPlaced := range placed {
		if !isPlaced {
			continue
		}
		dist := math.Hypot(coords[i][0]-target[0], coords[i][1]-target[1])
		if dist < 0.55 {
			score += 100
			continue
		}
		score += 0.08 / (dist * dist)
	}

	for _, angle := range occupied {
		sep := angularDistance(candidate, angle)
		if sep < math.Pi/6 {
			score += 18 * (math.Pi/6 - sep)
		}
	}

	for _, angle := range chosen {
		sep := angularDistance(candidate, angle)
		if sep < math.Pi/4 {
			score += 14 * (math.Pi/4 - sep)
		}
	}

	return score
}

func openSectorAngle(occupied []float64, fallback float64) float64 {
	if len(occupied) == 0 {
		return normalizeAngle(fallback)
	}

	sorted := append([]float64(nil), occupied...)
	for i, angle := range sorted {
		sorted[i] = normalizeAngle(angle)
		if sorted[i] < 0 {
			sorted[i] += 2 * math.Pi
		}
	}
	sort.Float64s(sorted)

	bestGap := -1.0
	bestAngle := normalizeAngle(fallback)
	fallbackNorm := bestAngle
	if fallbackNorm < 0 {
		fallbackNorm += 2 * math.Pi
	}

	for i := 0; i < len(sorted); i++ {
		current := sorted[i]
		next := sorted[(i+1)%len(sorted)]
		if i == len(sorted)-1 {
			next += 2 * math.Pi
		}
		gap := next - current
		candidate := current + gap/2
		candidateNorm := math.Mod(candidate, 2*math.Pi)
		if gap > bestGap+1e-9 {
			bestGap = gap
			bestAngle = candidateNorm
			continue
		}
		if math.Abs(gap-bestGap) < 1e-9 && angularDistance(candidateNorm, fallbackNorm) < angularDistance(bestAngle, fallbackNorm) {
			bestAngle = candidateNorm
		}
	}

	return normalizeAngle(bestAngle)
}

func angularDistance(a, b float64) float64 {
	diff := math.Abs(normalizeAngle(a - b))
	if diff > math.Pi {
		return 2*math.Pi - diff
	}
	return diff
}

func branchSize(m *Molecule, start, parent int, placed []bool) int {
	visited := map[int]bool{parent: true}
	var walk func(int) int
	walk = func(current int) int {
		visited[current] = true
		total := 1
		for _, neighbor := range m.Atoms[current].Neighbors {
			if visited[neighbor] || placed[neighbor] {
				continue
			}
			total += walk(neighbor)
		}
		return total
	}
	return walk(start)
}

// relaxWithPins runs force-directed refinement. Pinned atoms (rings) are fixed.
func relaxWithPins(m *Molecule, coords [][2]float64, pinned []bool) {
	if len(coords) <= 1 {
		return
	}

	const (
		iterations = 120
		springK    = 0.15
		repelK     = 0.012
		step       = 0.06
	)

	for iter := 0; iter < iterations; iter++ {
		forces := make([][2]float64, len(coords))

		// Repulsion between all atom pairs
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

		// Spring forces along bonds
		for _, bond := range m.Bonds {
			dx := coords[bond.To][0] - coords[bond.From][0]
			dy := coords[bond.To][1] - coords[bond.From][1]
			dist := math.Hypot(dx, dy)
			if dist < 1e-6 {
				dist = 1e-6
			}
			target := 1.0
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

		// Apply forces only to unpinned atoms
		for i := range coords {
			if pinned[i] {
				continue
			}
			coords[i][0] += forces[i][0] * step
			coords[i][1] += forces[i][1] * step
		}
	}
}

func stretchRingSystemLinkers(m *Molecule, coords [][2]float64) {
	if len(coords) == 0 {
		return
	}

	systems := ringSystems(m.Rings(), len(m.Atoms))
	if len(systems) < 2 {
		return
	}

	atomSystem := make([]int, len(m.Atoms))
	for i := range atomSystem {
		atomSystem[i] = -1
	}
	for systemID, atoms := range systems {
		for idx := range atoms {
			atomSystem[idx] = systemID
		}
	}

	for _, bond := range m.Bonds {
		if bond.Aromatic || bond.Order != 1 {
			continue
		}
		leftSystem := atomSystem[bond.From]
		rightSystem := atomSystem[bond.To]
		if leftSystem < 0 || rightSystem < 0 || leftSystem == rightSystem {
			continue
		}

		left, right, ok := splitComponentsAcrossBond(m, bond.From, bond.To)
		if !ok {
			continue
		}

		dx := coords[bond.To][0] - coords[bond.From][0]
		dy := coords[bond.To][1] - coords[bond.From][1]
		dist := math.Hypot(dx, dy)
		if dist < 1e-6 {
			continue
		}
		target := 1.55
		if dist >= target {
			continue
		}

		shift := target - dist
		ux := dx / dist
		uy := dy / dist
		shiftSet := right
		dir := 1.0
		if len(left) < len(right) {
			shiftSet = left
			dir = -1
		}

		for idx := range shiftSet {
			coords[idx][0] += ux * shift * dir
			coords[idx][1] += uy * shift * dir
		}
	}
}

func ringSystems(rings []Ring, atomCount int) []map[int]bool {
	if len(rings) == 0 {
		return nil
	}

	parent := make([]int, len(rings))
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(a, b int) {
		ra := find(a)
		rb := find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	atomToRing := make([]int, atomCount)
	for i := range atomToRing {
		atomToRing[i] = -1
	}
	for ringIdx, ring := range rings {
		for _, atomIdx := range ring.Atoms {
			if prev := atomToRing[atomIdx]; prev >= 0 {
				union(prev, ringIdx)
			} else {
				atomToRing[atomIdx] = ringIdx
			}
		}
	}

	grouped := map[int]map[int]bool{}
	for ringIdx, ring := range rings {
		root := find(ringIdx)
		if grouped[root] == nil {
			grouped[root] = map[int]bool{}
		}
		for _, atomIdx := range ring.Atoms {
			grouped[root][atomIdx] = true
		}
	}

	out := make([]map[int]bool, 0, len(grouped))
	for _, atoms := range grouped {
		out = append(out, atoms)
	}
	return out
}

func splitComponentsAcrossBond(m *Molecule, left, right int) (map[int]bool, map[int]bool, bool) {
	leftSet := map[int]bool{}
	rightSet := map[int]bool{}
	visitWithoutBond(m, left, left, right, leftSet)
	visitWithoutBond(m, right, left, right, rightSet)
	if !leftSet[right] && !rightSet[left] {
		return leftSet, rightSet, true
	}
	return nil, nil, false
}

func visitWithoutBond(m *Molecule, start, edgeA, edgeB int, seen map[int]bool) {
	stack := []int{start}
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[idx] {
			continue
		}
		seen[idx] = true
		for _, neighbor := range m.Atoms[idx].Neighbors {
			if (idx == edgeA && neighbor == edgeB) || (idx == edgeB && neighbor == edgeA) {
				continue
			}
			if !seen[neighbor] {
				stack = append(stack, neighbor)
			}
		}
	}
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

	// Scale to unit bond length (two separate passes to avoid stale bounds)
	for i := range m.Atoms {
		m.Atoms[i].Coords[0] /= avgBond
		m.Atoms[i].Coords[1] /= avgBond
	}

	minX, maxX := m.Atoms[0].Coords[0], m.Atoms[0].Coords[0]
	minY, maxY := m.Atoms[0].Coords[1], m.Atoms[0].Coords[1]
	for _, atom := range m.Atoms[1:] {
		minX = math.Min(minX, atom.Coords[0])
		maxX = math.Max(maxX, atom.Coords[0])
		minY = math.Min(minY, atom.Coords[1])
		maxY = math.Max(maxY, atom.Coords[1])
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
