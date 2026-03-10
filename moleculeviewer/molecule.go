package moleculeviewer

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
)

// Atom represents one atom in a molecule graph.
type Atom struct {
	Index         int
	Symbol        string
	Charge        int
	Hydrogens     int
	PartialCharge float64
	Aromatic      bool
	Scaffold      bool
	Coords        [2]float64
	Neighbors     []int
}

// Bond connects two atoms in the molecule graph.
type Bond struct {
	Index    int
	From     int
	To       int
	Order    int
	Aromatic bool
}

// Molecule is the typed data model rendered by the component.
type Molecule struct {
	Atoms  []Atom
	Bonds  []Bond
	Name   string
	SMILES string
}

// FunctionalGroup is a coarse structural motif detected from the molecule graph.
type FunctionalGroup struct {
	Name  string
	Atoms []int
}

// SearchResult captures atom hits for a query.
type SearchResult struct {
	Query       string
	AtomIndices []int
	Groups      []string
}

// Clone returns a defensive copy of the molecule.
func (m Molecule) Clone() Molecule {
	cloned := Molecule{
		Name:   m.Name,
		SMILES: m.SMILES,
		Atoms:  make([]Atom, len(m.Atoms)),
		Bonds:  make([]Bond, len(m.Bonds)),
	}
	for i, atom := range m.Atoms {
		atom.Neighbors = append([]int(nil), atom.Neighbors...)
		cloned.Atoms[i] = atom
	}
	copy(cloned.Bonds, m.Bonds)
	return cloned
}

// Normalize repairs indices, neighbors, and hydrogen estimates.
func (m *Molecule) Normalize() {
	if m == nil {
		return
	}

	for i := range m.Atoms {
		m.Atoms[i].Index = i
		m.Atoms[i].Symbol = canonicalSymbol(m.Atoms[i].Symbol)
		m.Atoms[i].Neighbors = m.Atoms[i].Neighbors[:0]
	}

	validBonds := m.Bonds[:0]
	for i := range m.Bonds {
		bond := m.Bonds[i]
		if bond.From < 0 || bond.From >= len(m.Atoms) || bond.To < 0 || bond.To >= len(m.Atoms) || bond.From == bond.To {
			continue
		}
		if bond.Order <= 0 {
			bond.Order = 1
		}
		bond.Index = len(validBonds)
		validBonds = append(validBonds, bond)
		m.Atoms[bond.From].Neighbors = append(m.Atoms[bond.From].Neighbors, bond.To)
		m.Atoms[bond.To].Neighbors = append(m.Atoms[bond.To].Neighbors, bond.From)
	}
	m.Bonds = validBonds

	for i := range m.Atoms {
		sort.Ints(m.Atoms[i].Neighbors)
		m.Atoms[i].Hydrogens = maxInt(0, estimateHydrogens(m.Atoms[i], m.atomValence(i)))
	}

	scaffold := m.ScaffoldAtoms()
	for i := range m.Atoms {
		m.Atoms[i].Scaffold = scaffold[i]
		m.Atoms[i].PartialCharge = estimatePartialCharge(m.Atoms[i])
	}
}

// HasCoordinates reports whether the molecule already has usable 2D coordinates.
func (m Molecule) HasCoordinates() bool {
	if len(m.Atoms) == 0 {
		return false
	}
	if len(m.Atoms) == 1 {
		return true
	}

	minX, maxX := m.Atoms[0].Coords[0], m.Atoms[0].Coords[0]
	minY, maxY := m.Atoms[0].Coords[1], m.Atoms[0].Coords[1]
	for _, atom := range m.Atoms[1:] {
		minX = math.Min(minX, atom.Coords[0])
		maxX = math.Max(maxX, atom.Coords[0])
		minY = math.Min(minY, atom.Coords[1])
		maxY = math.Max(maxY, atom.Coords[1])
	}
	return math.Abs(maxX-minX) > 1e-6 || math.Abs(maxY-minY) > 1e-6
}

// Formula returns a simple Hill-style molecular formula.
func (m Molecule) Formula() string {
	if len(m.Atoms) == 0 {
		return ""
	}

	counts := map[string]int{}
	for _, atom := range m.Atoms {
		symbol := canonicalSymbol(atom.Symbol)
		if symbol == "" {
			continue
		}
		counts[symbol]++
		if symbol != "H" && atom.Hydrogens > 0 {
			counts["H"] += atom.Hydrogens
		}
	}

	var order []string
	if counts["C"] > 0 {
		order = append(order, "C")
		if counts["H"] > 0 {
			order = append(order, "H")
		}
	}

	var others []string
	for symbol := range counts {
		if symbol == "C" || (symbol == "H" && counts["C"] > 0) {
			continue
		}
		others = append(others, symbol)
	}
	sort.Strings(others)
	order = append(order, others...)

	var b strings.Builder
	for _, symbol := range order {
		b.WriteString(symbol)
		if counts[symbol] > 1 {
			b.WriteString(strconv.Itoa(counts[symbol]))
		}
	}
	return b.String()
}

// BondBetween returns the bond connecting atoms a and b, if present.
func (m Molecule) BondBetween(a, b int) (Bond, bool) {
	for _, bond := range m.Bonds {
		if (bond.From == a && bond.To == b) || (bond.From == b && bond.To == a) {
			return bond, true
		}
	}
	return Bond{}, false
}

// FunctionalGroups detects common medicinal-chemistry motifs.
func (m Molecule) FunctionalGroups() []FunctionalGroup {
	type key struct {
		name string
		sig  string
	}

	seen := map[key]bool{}
	var groups []FunctionalGroup

	add := func(name string, atoms ...int) {
		if len(atoms) == 0 {
			return
		}
		cp := append([]int(nil), atoms...)
		sort.Ints(cp)
		parts := make([]string, len(cp))
		for i, idx := range cp {
			parts[i] = strconv.Itoa(idx)
		}
		k := key{name: name, sig: strings.Join(parts, ",")}
		if seen[k] {
			return
		}
		seen[k] = true
		groups = append(groups, FunctionalGroup{Name: name, Atoms: cp})
	}

	for _, atom := range m.Atoms {
		switch atom.Symbol {
		case "O":
			if carbonylCarbon, ok := m.carriedCarbonyl(atom.Index); ok {
				add("carbonyl", carbonylCarbon, atom.Index)
			}
			if m.isHydroxyl(atom.Index) {
				add("hydroxyl", atom.Index)
			}
			if m.isEther(atom.Index) {
				add("ether", atom.Index)
			}
		case "N":
			if m.isAmine(atom.Index) {
				add("amine", atom.Index)
			}
		case "S":
			if m.isThiol(atom.Index) {
				add("thiol", atom.Index)
			}
		case "P":
			if m.isPhosphate(atom.Index) {
				add("phosphate", atom.Index)
			}
		case "F", "Cl", "Br", "I":
			add("halide", atom.Index)
		}
	}

	for _, bond := range m.Bonds {
		if bond.Order != 2 {
			continue
		}
		a := m.Atoms[bond.From]
		b := m.Atoms[bond.To]
		if a.Symbol == "C" && b.Symbol == "O" {
			add("carbonyl", bond.From, bond.To)
			if m.hasNeighborWithSymbol(bond.From, "N", bond.To) {
				add("amide", bond.From, bond.To, m.firstNeighborWithSymbol(bond.From, "N", bond.To))
			}
			if m.hasNeighborWithSymbol(bond.From, "O", bond.To) {
				add("carboxyl", bond.From, bond.To, m.firstNeighborWithSymbol(bond.From, "O", bond.To))
			}
		}
		if b.Symbol == "C" && a.Symbol == "O" {
			add("carbonyl", bond.From, bond.To)
			if m.hasNeighborWithSymbol(bond.To, "N", bond.From) {
				add("amide", bond.From, bond.To, m.firstNeighborWithSymbol(bond.To, "N", bond.From))
			}
			if m.hasNeighborWithSymbol(bond.To, "O", bond.From) {
				add("carboxyl", bond.From, bond.To, m.firstNeighborWithSymbol(bond.To, "O", bond.From))
			}
		}
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Name == groups[j].Name {
			return len(groups[i].Atoms) < len(groups[j].Atoms)
		}
		return groups[i].Name < groups[j].Name
	})
	return groups
}

// Search performs element, motif, and bond-pattern matching.
func (m Molecule) Search(query string) SearchResult {
	trimmed := strings.TrimSpace(query)
	result := SearchResult{Query: trimmed}
	if trimmed == "" {
		return result
	}

	lower := strings.ToLower(trimmed)
	matches := map[int]bool{}
	groupSet := map[string]bool{}

	if order, left, right, ok := parseBondQuery(lower); ok {
		for _, bond := range m.Bonds {
			if bond.Order != order {
				continue
			}
			a := strings.ToLower(m.Atoms[bond.From].Symbol)
			b := strings.ToLower(m.Atoms[bond.To].Symbol)
			if (a == left && b == right) || (a == right && b == left) {
				matches[bond.From] = true
				matches[bond.To] = true
			}
		}
	}

	if symbol := canonicalSymbol(trimmed); symbol != "" {
		for _, atom := range m.Atoms {
			if atom.Symbol == symbol {
				matches[atom.Index] = true
			}
		}
	}

	if idx, ok := parseIndexedAtomQuery(trimmed); ok && idx >= 0 && idx < len(m.Atoms) {
		matches[idx] = true
	}

	switch lower {
	case "aromatic", "ring":
		for _, atom := range m.Atoms {
			if atom.Aromatic || m.atomIsScaffold(atom.Index) {
				matches[atom.Index] = true
			}
		}
	case "hetero", "heteroatom":
		for _, atom := range m.Atoms {
			if atom.Symbol != "C" && atom.Symbol != "H" {
				matches[atom.Index] = true
			}
		}
	case "scaffold", "core":
		for idx := range m.ScaffoldAtoms() {
			matches[idx] = true
		}
	case "r-group", "rgroup", "sidechain":
		scaffold := m.ScaffoldAtoms()
		for _, atom := range m.Atoms {
			if !scaffold[atom.Index] {
				matches[atom.Index] = true
			}
		}
	case "positive", "cation":
		for _, atom := range m.Atoms {
			if partialChargeClass(atom) > 0 {
				matches[atom.Index] = true
			}
		}
	case "negative", "anion":
		for _, atom := range m.Atoms {
			if partialChargeClass(atom) < 0 {
				matches[atom.Index] = true
			}
		}
	}

	for _, group := range m.FunctionalGroups() {
		if strings.Contains(group.Name, lower) {
			groupSet[group.Name] = true
			for _, idx := range group.Atoms {
				matches[idx] = true
			}
		}
	}

	for idx := range matches {
		result.AtomIndices = append(result.AtomIndices, idx)
	}
	sort.Ints(result.AtomIndices)

	for name := range groupSet {
		result.Groups = append(result.Groups, name)
	}
	sort.Strings(result.Groups)

	return result
}

// ScaffoldAtoms returns a Murcko-like scaffold approximation produced by leaf pruning.
func (m Molecule) ScaffoldAtoms() map[int]bool {
	n := len(m.Atoms)
	if n == 0 {
		return map[int]bool{}
	}

	degrees := make([]int, n)
	for i, atom := range m.Atoms {
		degrees[i] = len(atom.Neighbors)
	}

	removed := make([]bool, n)
	queue := make([]int, 0, n)
	for i, degree := range degrees {
		if degree <= 1 {
			queue = append(queue, i)
		}
	}

	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		if removed[idx] {
			continue
		}
		removed[idx] = true
		for _, neighbor := range m.Atoms[idx].Neighbors {
			if removed[neighbor] {
				continue
			}
			degrees[neighbor]--
			if degrees[neighbor] == 1 {
				queue = append(queue, neighbor)
			}
		}
	}

	out := map[int]bool{}
	for i := range m.Atoms {
		if !removed[i] {
			out[i] = true
		}
	}

	if len(out) == 0 {
		for _, atom := range m.Atoms {
			if len(atom.Neighbors) > 1 {
				out[atom.Index] = true
			}
		}
		if len(out) == 0 && len(m.Atoms) > 0 {
			out[0] = true
		}
	}

	return out
}

func (m Molecule) atomValence(idx int) float64 {
	total := 0.0
	for _, bond := range m.Bonds {
		if bond.From != idx && bond.To != idx {
			continue
		}
		if bond.Aromatic {
			total += 1.5
			continue
		}
		total += float64(maxInt(1, bond.Order))
	}
	return total
}

func estimateHydrogens(atom Atom, valence float64) int {
	if atom.Symbol == "H" {
		return 0
	}

	target := typicalValence(atom)
	if atom.Charge > 0 && (atom.Symbol == "N" || atom.Symbol == "P") {
		target += atom.Charge
	}
	if atom.Charge < 0 && (atom.Symbol == "O" || atom.Symbol == "S" || atom.Symbol == "N") {
		target += atom.Charge
	}

	hydrogens := int(math.Round(float64(target) - valence))
	if hydrogens < 0 {
		return 0
	}
	return hydrogens
}

func typicalValence(atom Atom) int {
	switch atom.Symbol {
	case "C":
		return 4
	case "N":
		return 3
	case "O":
		return 2
	case "S":
		if atom.Aromatic {
			return 2
		}
		return 2
	case "P":
		return 3
	case "F", "Cl", "Br", "I", "H":
		return 1
	case "B":
		return 3
	case "Si":
		return 4
	default:
		if atom.Aromatic {
			return 3
		}
		return maxInt(1, len(atom.Neighbors)+atom.Hydrogens)
	}
}

func partialChargeClass(atom Atom) int {
	switch {
	case atom.Charge > 0:
		return 1
	case atom.Charge < 0:
		return -1
	}

	switch atom.Symbol {
	case "N", "P":
		if atom.Hydrogens > 0 {
			return 1
		}
	case "O", "S", "F", "Cl", "Br", "I":
		return -1
	}
	return 0
}

func canonicalSymbol(symbol string) string {
	if symbol == "" {
		return ""
	}
	lower := strings.ToLower(symbol)
	switch lower {
	case "cl":
		return "Cl"
	case "br":
		return "Br"
	case "si":
		return "Si"
	case "na":
		return "Na"
	case "ca":
		return "Ca"
	case "li":
		return "Li"
	case "mg":
		return "Mg"
	case "zn":
		return "Zn"
	case "fe":
		return "Fe"
	case "al":
		return "Al"
	}
	return strings.ToUpper(lower[:1]) + lower[1:]
}

func atomDisplayName(symbol string) string {
	switch canonicalSymbol(symbol) {
	case "H":
		return "Hydrogen"
	case "C":
		return "Carbon"
	case "N":
		return "Nitrogen"
	case "O":
		return "Oxygen"
	case "S":
		return "Sulfur"
	case "P":
		return "Phosphorus"
	case "F":
		return "Fluorine"
	case "Cl":
		return "Chlorine"
	case "Br":
		return "Bromine"
	case "I":
		return "Iodine"
	case "B":
		return "Boron"
	case "Si":
		return "Silicon"
	case "Na":
		return "Sodium"
	case "K":
		return "Potassium"
	case "Ca":
		return "Calcium"
	case "Mg":
		return "Magnesium"
	case "Zn":
		return "Zinc"
	case "Fe":
		return "Iron"
	default:
		return canonicalSymbol(symbol)
	}
}

type elementInfo struct {
	Name string
	Kind string
}

func lookupElement(symbol string) elementInfo {
	canonical := canonicalSymbol(symbol)
	switch canonical {
	case "C":
		return elementInfo{Name: "Carbon", Kind: "carbon"}
	case "H":
		return elementInfo{Name: "Hydrogen", Kind: "hydrogen"}
	case "F", "Cl", "Br", "I":
		return elementInfo{Name: atomDisplayName(canonical), Kind: "halogen"}
	case "Na", "K", "Ca", "Mg", "Zn", "Fe", "Al":
		return elementInfo{Name: atomDisplayName(canonical), Kind: "metal"}
	default:
		return elementInfo{Name: atomDisplayName(canonical), Kind: "hetero"}
	}
}

func parseIndexedAtomQuery(query string) (int, bool) {
	if query == "" {
		return 0, false
	}
	digits := -1
	for i, r := range query {
		if r >= '0' && r <= '9' {
			digits = i
			break
		}
	}
	if digits < 0 {
		return 0, false
	}
	value, err := strconv.Atoi(query[digits:])
	if err != nil {
		return 0, false
	}
	if value <= 0 {
		return 0, false
	}
	return value - 1, true
}

func parseBondQuery(query string) (order int, left, right string, ok bool) {
	for _, part := range []struct {
		op    string
		order int
	}{
		{"#", 3},
		{"=", 2},
		{"-", 1},
	} {
		if !strings.Contains(query, part.op) {
			continue
		}
		fields := strings.Split(query, part.op)
		if len(fields) != 2 {
			continue
		}
		left = strings.ToLower(canonicalSymbol(strings.TrimSpace(fields[0])))
		right = strings.ToLower(canonicalSymbol(strings.TrimSpace(fields[1])))
		if left == "" || right == "" {
			return 0, "", "", false
		}
		return part.order, left, right, true
	}
	return 0, "", "", false
}

func (m Molecule) carriedCarbonyl(oxygenIdx int) (int, bool) {
	for _, bond := range m.Bonds {
		if bond.Order != 2 {
			continue
		}
		if bond.From == oxygenIdx && m.Atoms[bond.To].Symbol == "C" {
			return bond.To, true
		}
		if bond.To == oxygenIdx && m.Atoms[bond.From].Symbol == "C" {
			return bond.From, true
		}
	}
	return 0, false
}

func (m Molecule) isHydroxyl(atomIdx int) bool {
	atom := m.Atoms[atomIdx]
	if atom.Symbol != "O" {
		return false
	}
	if len(atom.Neighbors) == 0 {
		return true
	}
	for _, bond := range m.Bonds {
		if (bond.From == atomIdx || bond.To == atomIdx) && bond.Order > 1 {
			return false
		}
	}
	return atom.Hydrogens > 0 || len(atom.Neighbors) == 1
}

func (m Molecule) isEther(atomIdx int) bool {
	atom := m.Atoms[atomIdx]
	if atom.Symbol != "O" || len(atom.Neighbors) != 2 {
		return false
	}
	for _, neighbor := range atom.Neighbors {
		if m.Atoms[neighbor].Symbol != "C" {
			return false
		}
	}
	for _, bond := range m.Bonds {
		if (bond.From == atomIdx || bond.To == atomIdx) && bond.Order > 1 {
			return false
		}
	}
	return true
}

func (m Molecule) isAmine(atomIdx int) bool {
	atom := m.Atoms[atomIdx]
	if atom.Symbol != "N" || atom.Aromatic {
		return false
	}
	for _, bond := range m.Bonds {
		if (bond.From == atomIdx || bond.To == atomIdx) && bond.Order > 1 {
			return false
		}
	}
	return len(atom.Neighbors) > 0
}

func (m Molecule) isThiol(atomIdx int) bool {
	atom := m.Atoms[atomIdx]
	if atom.Symbol != "S" || atom.Aromatic {
		return false
	}
	return atom.Hydrogens > 0 || len(atom.Neighbors) == 1
}

func (m Molecule) isPhosphate(atomIdx int) bool {
	atom := m.Atoms[atomIdx]
	if atom.Symbol != "P" {
		return false
	}
	hetero := 0
	for _, neighbor := range atom.Neighbors {
		switch m.Atoms[neighbor].Symbol {
		case "O", "S", "N":
			hetero++
		}
	}
	return hetero >= 3
}

func (m Molecule) hasNeighborWithSymbol(atomIdx int, symbol string, exclude int) bool {
	return m.firstNeighborWithSymbol(atomIdx, symbol, exclude) >= 0
}

func (m Molecule) firstNeighborWithSymbol(atomIdx int, symbol string, exclude int) int {
	for _, neighbor := range m.Atoms[atomIdx].Neighbors {
		if neighbor == exclude {
			continue
		}
		if m.Atoms[neighbor].Symbol == symbol {
			return neighbor
		}
	}
	return -1
}

func (m Molecule) atomIsScaffold(idx int) bool {
	return m.ScaffoldAtoms()[idx]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func formatBondOrder(bond Bond) string {
	if bond.Aromatic {
		return "aromatic"
	}
	switch bond.Order {
	case 1:
		return "single"
	case 2:
		return "double"
	case 3:
		return "triple"
	default:
		return fmt.Sprintf("order %d", bond.Order)
	}
}

// finalize normalizes graph metadata and ensures coordinates are present.
func (m *Molecule) finalize() {
	if m == nil {
		return
	}
	LayoutMolecule(m)
	scaffold := m.ScaffoldAtoms()
	for i := range m.Atoms {
		m.Atoms[i].Scaffold = scaffold[i]
		m.Atoms[i].PartialCharge = estimatePartialCharge(m.Atoms[i])
	}
}

// AtomLabel returns a short stable atom label like C3 or O11.
func (m Molecule) AtomLabel(index int) string {
	if index < 0 || index >= len(m.Atoms) {
		return "?"
	}
	atom := m.Atoms[index]
	return fmt.Sprintf("%s%d", atom.Symbol, atom.Index+1)
}

func (m Molecule) atomName(index int) string {
	if index < 0 || index >= len(m.Atoms) {
		return ""
	}
	return atomDisplayName(m.Atoms[index].Symbol)
}

func estimatePartialCharge(atom Atom) float64 {
	if atom.Charge != 0 {
		return float64(atom.Charge)
	}
	switch atom.Symbol {
	case "O":
		return -0.35
	case "N":
		if atom.Hydrogens > 0 {
			return 0.25
		}
		if atom.Aromatic {
			return -0.08
		}
		return -0.12
	case "S":
		return -0.18
	case "P":
		return 0.32
	case "F", "Cl", "Br", "I":
		return -0.10
	default:
		return 0
	}
}

func bondDescriptor(bond Bond) string {
	if bond.Aromatic {
		return "aromatic"
	}
	switch bond.Order {
	case 1:
		return "single"
	case 2:
		return "double"
	case 3:
		return "triple"
	default:
		return fmt.Sprintf("order %d", bond.Order)
	}
}
