package moleculeviewer

import "math"

const terminalCharAspect = 1.85

type diagramProjection struct {
	positions  [][2]int
	collisions int
	score      float64
	scale      float64
}

func (m Model) projectAtoms(width, height int) diagramProjection {
	n := len(m.molecule.Atoms)
	if n == 0 {
		return diagramProjection{}
	}

	if n == 1 {
		return diagramProjection{
			positions: [][2]int{{width / 2, height / 2}},
			score:     0,
			scale:     1,
		}
	}

	coords := make([][2]float64, n)
	for i, atom := range m.molecule.Atoms {
		coords[i] = atom.Coords
	}

	best := diagramProjection{score: math.Inf(1)}
	for _, angle := range projectionAngles() {
		rotated := rotateCoords(coords, angle)
		projected := quantizeProjection(rotated, width, height)
		projected.score = scoreProjection(m.molecule, rotated, projected.positions, width, height) - projected.scale*0.12
		projected.collisions = countAtomCollisions(projected.positions)
		if projected.score < best.score {
			best = projected
		}
	}

	if !math.IsInf(best.score, 0) {
		return best
	}
	return quantizeProjection(coords, width, height)
}

func projectionAngles() []float64 {
	angles := make([]float64, 0, 12)
	for deg := 0; deg < 180; deg += 15 {
		angles = append(angles, float64(deg)*math.Pi/180)
	}
	return angles
}

func rotateCoords(coords [][2]float64, angle float64) [][2]float64 {
	if math.Abs(angle) < 1e-9 {
		return append([][2]float64(nil), coords...)
	}

	minX, maxX := coords[0][0], coords[0][0]
	minY, maxY := coords[0][1], coords[0][1]
	for _, coord := range coords[1:] {
		minX = math.Min(minX, coord[0])
		maxX = math.Max(maxX, coord[0])
		minY = math.Min(minY, coord[1])
		maxY = math.Max(maxY, coord[1])
	}
	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2

	sinA, cosA := math.Sin(angle), math.Cos(angle)
	out := make([][2]float64, len(coords))
	for i, coord := range coords {
		x := coord[0] - centerX
		y := coord[1] - centerY
		out[i][0] = x*cosA - y*sinA
		out[i][1] = x*sinA + y*cosA
	}
	return out
}

func quantizeProjection(coords [][2]float64, width, height int) diagramProjection {
	n := len(coords)
	if n == 0 {
		return diagramProjection{}
	}

	minX, maxX := coords[0][0], coords[0][0]
	minY, maxY := coords[0][1], coords[0][1]
	for _, coord := range coords[1:] {
		minX = math.Min(minX, coord[0])
		maxX = math.Max(maxX, coord[0])
		minY = math.Min(minY, coord[1])
		maxY = math.Max(maxY, coord[1])
	}

	spanX := math.Max(maxX-minX, 1e-6)
	spanY := math.Max(maxY-minY, 1e-6)
	availW := float64(maxInt(8, width-6))
	availH := float64(maxInt(4, height-4))
	scale := math.Min(availW/spanX, availH*terminalCharAspect/spanY)
	if scale < 1 {
		scale = 1
	}

	centerX := (minX + maxX) / 2
	centerY := (minY + maxY) / 2
	screenX := width / 2
	screenY := height / 2
	out := make([][2]int, n)
	for i, coord := range coords {
		out[i][0] = int(math.Round((coord[0]-centerX)*scale)) + screenX
		out[i][1] = screenY - int(math.Round((coord[1]-centerY)*scale/terminalCharAspect))
	}

	return diagramProjection{
		positions: out,
		scale:     scale,
	}
}

func scoreProjection(mol Molecule, coords [][2]float64, positions [][2]int, width, height int) float64 {
	if len(positions) == 0 {
		return math.Inf(1)
	}

	score := 0.0
	atomOccupancy := make(map[[2]int]int, len(positions))
	for _, pos := range positions {
		atomOccupancy[[2]int{pos[0], pos[1]}]++
		if pos[0] < 1 || pos[0] >= width-1 || pos[1] < 0 || pos[1] >= height {
			score += 12
		}
	}

	score += float64(countAtomCollisions(positions)) * 22

	bonded := make(map[[2]int]bool, len(mol.Bonds))
	for _, bond := range mol.Bonds {
		bonded[bondKey(bond.From, bond.To)] = true

		a := positions[bond.From]
		b := positions[bond.To]
		points := rasterLine(a[0], a[1], b[0], b[1])
		if len(points) < 2 {
			score += 8
			continue
		}

		projectedLength := math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1]))
		if projectedLength < 2 {
			score += (2 - projectedLength) * 6
		}

		score += float64(rasterTurns(points)) * 1.35
		score += orientationPenalty(coords[bond.From], coords[bond.To]) * 0.75

		for idx, point := range points {
			if idx == 0 || idx == len(points)-1 {
				continue
			}
			if atomOccupancy[[2]int{point[0], point[1]}] > 0 {
				score += 5
			}
		}
	}

	for i := 0; i < len(positions); i++ {
		for j := i + 1; j < len(positions); j++ {
			if bonded[bondKey(i, j)] {
				continue
			}
			dist := math.Abs(float64(positions[i][0]-positions[j][0])) + math.Abs(float64(positions[i][1]-positions[j][1]))
			if dist < 1.5 {
				score += (1.5 - dist) * 4
			}
		}
	}

	return score
}

func countAtomCollisions(positions [][2]int) int {
	seen := map[[2]int]int{}
	collisions := 0
	for _, pos := range positions {
		key := [2]int{pos[0], pos[1]}
		seen[key]++
		if seen[key] > 1 {
			collisions++
		}
	}
	return collisions
}

func rasterTurns(points [][2]int) int {
	if len(points) < 3 {
		return 0
	}
	turns := 0
	lastDX := points[1][0] - points[0][0]
	lastDY := points[1][1] - points[0][1]
	for i := 2; i < len(points); i++ {
		dx := points[i][0] - points[i-1][0]
		dy := points[i][1] - points[i-1][1]
		if dx != lastDX || dy != lastDY {
			turns++
			lastDX, lastDY = dx, dy
		}
	}
	return turns
}

func orientationPenalty(a, b [2]float64) float64 {
	angle := math.Atan2(b[1]-a[1], b[0]-a[0])
	if angle < 0 {
		angle = -angle
	}
	targets := []float64{0, math.Pi / 4, math.Pi / 2, 3 * math.Pi / 4, math.Pi}
	best := math.Pi
	for _, target := range targets {
		diff := math.Abs(angle - target)
		if diff > math.Pi/2 {
			diff = math.Pi - diff
		}
		if diff < best {
			best = diff
		}
	}
	return best
}
