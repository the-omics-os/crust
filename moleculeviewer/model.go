// Package moleculeviewer provides an interactive small-molecule viewer for
// Bubble Tea applications.
package moleculeviewer

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

// ViewMode controls the current coloring plane.
type ViewMode int

const (
	ViewModeIdentity ViewMode = iota
	ViewModeHetero
	ViewModeAromaticity
	ViewModePartialCharge
	ViewModeScaffold
)

const defaultTitle = "Small Molecule Viewer"

// Model is the Bubble Tea model for the small molecule viewer.
type Model struct {
	title         string
	molecule      Molecule
	theme         Theme
	width         int
	height        int
	mode          ViewMode
	selectedAtom  int
	lastAtom      int
	hoverBond     int
	showHelp      bool
	searching     bool
	searchBuffer  string
	searchMatches map[int]bool
	lastSearch    string
	status        string
	loadErr       error
}

// New creates a SmallMoleculeViewer with the given options.
func New(opts ...Option) Model {
	m := Model{
		title:         defaultTitle,
		theme:         DefaultTheme(),
		width:         92,
		height:        28,
		mode:          ViewModeIdentity,
		selectedAtom:  -1,
		lastAtom:      -1,
		hoverBond:     -1,
		searchMatches: map[int]bool{},
	}
	for _, opt := range opts {
		opt(&m)
	}
	if len(m.molecule.Atoms) > 0 && m.selectedAtom < 0 {
		m.selectedAtom = 0
	}
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		if msg.Width > 0 {
			m.width = msg.Width
		}
		if msg.Height > 0 {
			m.height = msg.Height
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.searching {
			return m.updateSearch(msg)
		}
		if m.showHelp {
			switch {
			case msg.String() == "?" || msg.Text == "?":
				m.showHelp = false
			case msg.String() == "esc":
				m.showHelp = false
			}
			return m, nil
		}
		switch {
		case msg.String() == "?" || msg.Text == "?":
			m.showHelp = true
			return m, nil
		case msg.String() == "tab":
			m.mode = (m.mode + 1) % 5
			m.status = fmt.Sprintf("Plane: %s", m.modeLabel())
			return m, nil
		case msg.String() == "shift+tab":
			if m.mode == 0 {
				m.mode = ViewModeScaffold
			} else {
				m.mode--
			}
			m.status = fmt.Sprintf("Plane: %s", m.modeLabel())
			return m, nil
		case msg.String() == "/" || msg.Text == "/":
			m.searching = true
			m.searchBuffer = m.lastSearch
			m.status = "Search mode"
			return m, nil
		case msg.String() == "enter":
			return m, m.submitCmd()
		case msg.String() == "esc":
			return m, cancelCmd()
		case msg.String() == "left" || msg.String() == "right" || msg.String() == "up" || msg.String() == "down":
			if moved := m.moveSelection(msg.String()); moved {
				return m, nil
			}
			m.status = "No bonded atom in that direction"
			return m, nil
		}
	}

	return m, nil
}

func (m Model) updateSearch(km tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch km.String() {
	case "esc":
		m.searching = false
		m.status = "Search dismissed"
		return m, nil
	case "enter":
		m.applySearch()
		m.searching = false
		return m, nil
	case "backspace":
		if len(m.searchBuffer) > 0 {
			_, size := lastRune(m.searchBuffer)
			m.searchBuffer = m.searchBuffer[:len(m.searchBuffer)-size]
		}
		return m, nil
	case "space":
		m.searchBuffer += " "
		return m, nil
	}

	if km.Text != "" && km.Mod == 0 {
		m.searchBuffer += km.Text
		return m, nil
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the viewer as a plain string.
func (m Model) Render() string {
	return m.render()
}

// SetMolecule replaces the active molecule with a defensive copy.
func (m *Model) SetMolecule(mol Molecule) {
	m.setMolecule(mol)
}

// SetSMILES parses and loads a SMILES string.
func (m *Model) SetSMILES(smiles string) error {
	mol, err := ParseSMILES(smiles)
	if err != nil {
		m.loadErr = err
		return err
	}
	m.loadErr = nil
	m.setMolecule(mol)
	return nil
}

// SetMOL parses and loads a MOL block.
func (m *Model) SetMOL(molfile string) error {
	mol, err := ParseMOL(molfile)
	if err != nil {
		m.loadErr = err
		return err
	}
	m.loadErr = nil
	m.setMolecule(mol)
	return nil
}

// SetSDF parses and loads the first record in an SDF payload.
func (m *Model) SetSDF(sdf string) error {
	mol, err := ParseSDF(sdf)
	if err != nil {
		m.loadErr = err
		return err
	}
	m.loadErr = nil
	m.setMolecule(mol)
	return nil
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(width int) { m.width = width }

// SetHeight updates the rendering height.
func (m *Model) SetHeight(height int) { m.height = height }

// Molecule returns a defensive copy of the active molecule.
func (m Model) Molecule() Molecule { return m.molecule.Clone() }

// Err returns the last load error, if any.
func (m Model) Err() error { return m.loadErr }

// FocusedAtom returns the currently selected atom.
func (m Model) FocusedAtom() (Atom, bool) {
	if m.selectedAtom < 0 || m.selectedAtom >= len(m.molecule.Atoms) {
		return Atom{}, false
	}
	return m.molecule.Atoms[m.selectedAtom], true
}

// CurrentMode returns the current view mode.
func (m Model) CurrentMode() ViewMode { return m.mode }

func (m *Model) setMolecule(mol Molecule) {
	m.molecule = mol.Clone()
	m.molecule.finalize()
	if m.molecule.Name != "" && (m.title == "" || m.title == defaultTitle) {
		m.title = m.molecule.Name
	}
	if len(m.molecule.Atoms) > 0 {
		m.selectedAtom = 0
	}
	m.lastAtom = -1
	m.hoverBond = -1
	m.status = fmt.Sprintf("Loaded %d atoms and %d bonds", len(m.molecule.Atoms), len(m.molecule.Bonds))
	m.clearSearchMatches()
}

func (m *Model) applySearch() {
	query := strings.TrimSpace(strings.ToLower(m.searchBuffer))
	m.lastSearch = query
	m.clearSearchMatches()
	if query == "" {
		m.status = "Search cleared"
		return
	}
	result := m.molecule.Search(query)
	if len(result.AtomIndices) == 0 {
		m.status = fmt.Sprintf("No atoms matched %q", query)
		return
	}
	for _, idx := range result.AtomIndices {
		m.searchMatches[idx] = true
	}
	m.lastAtom = m.selectedAtom
	m.selectedAtom = m.pickBestSearchMatch(query, result.AtomIndices)
	m.hoverBond = m.findBondIndex(m.lastAtom, m.selectedAtom)
	groupNote := ""
	if len(result.Groups) > 0 {
		groupNote = fmt.Sprintf(" | groups: %s", strings.Join(result.Groups, ", "))
	}
	m.status = fmt.Sprintf("%d match(es) for %q; focused %s%s", len(result.AtomIndices), query, m.molecule.AtomLabel(m.selectedAtom), groupNote)
}

func (m *Model) clearSearchMatches() {
	m.searchMatches = map[int]bool{}
}

func (m *Model) moveSelection(direction string) bool {
	if len(m.molecule.Atoms) == 0 {
		return false
	}
	if m.selectedAtom < 0 || m.selectedAtom >= len(m.molecule.Atoms) {
		m.selectedAtom = 0
		return true
	}

	current := m.selectedAtom
	best := -1
	bestScore := math.Inf(-1)
	fallback := -1
	fallbackScore := math.Inf(-1)
	targetX, targetY := directionVector(direction)
	origin := m.molecule.Atoms[current].Coords
	neighbors := append([]int(nil), m.molecule.Atoms[current].Neighbors...)
	if m.molecule.Atoms[current].Symbol != "H" {
		var heavy []int
		for _, neighbor := range neighbors {
			if m.molecule.Atoms[neighbor].Symbol != "H" {
				heavy = append(heavy, neighbor)
			}
		}
		if len(heavy) > 0 {
			neighbors = heavy
		}
	}

	for _, neighbor := range neighbors {
		coords := m.molecule.Atoms[neighbor].Coords
		dx := coords[0] - origin[0]
		dy := coords[1] - origin[1]
		dist := math.Hypot(dx, dy)
		if dist < 1e-6 {
			continue
		}
		dot := (dx*targetX + dy*targetY) / dist
		score := dot*4 - dist*0.35
		if score > fallbackScore {
			fallback = neighbor
			fallbackScore = score
		}
		if dot <= 0.15 {
			continue
		}
		if score > bestScore {
			best = neighbor
			bestScore = score
		}
	}

	if best < 0 {
		best = fallback
	}
	if best < 0 || best == current {
		return false
	}

	m.lastAtom = current
	m.selectedAtom = best
	m.hoverBond = m.findBondIndex(current, best)
	if bond, ok := m.molecule.BondBetween(current, best); ok {
		m.status = fmt.Sprintf("Focused %s via %s bond", m.molecule.AtomLabel(best), bondDescriptor(bond))
	} else {
		m.status = fmt.Sprintf("Focused %s", m.molecule.AtomLabel(best))
	}
	return true
}

func (m Model) submitCmd() tea.Cmd {
	if m.selectedAtom < 0 || m.selectedAtom >= len(m.molecule.Atoms) {
		return nil
	}
	atom := m.molecule.Atoms[m.selectedAtom]
	payload := map[string]any{
		"name":              m.molecule.Name,
		"smiles":            m.molecule.SMILES,
		"formula":           m.molecule.Formula(),
		"atom_index":        atom.Index,
		"atom_symbol":       atom.Symbol,
		"atom_label":        m.molecule.AtomLabel(m.selectedAtom),
		"formal_charge":     atom.Charge,
		"implicit_h":        atom.Hydrogens,
		"partial_charge":    atom.PartialCharge,
		"aromatic":          atom.Aromatic,
		"scaffold":          atom.Scaffold,
		"functional_groups": m.groupsForAtom(m.selectedAtom),
		"neighbors":         append([]int(nil), atom.Neighbors...),
	}
	if m.hoverBond >= 0 && m.hoverBond < len(m.molecule.Bonds) {
		bond := m.molecule.Bonds[m.hoverBond]
		payload["bond_index"] = bond.Index
		payload["bond_from"] = bond.From
		payload["bond_to"] = bond.To
		payload["bond_order"] = bond.Order
		payload["bond_aromatic"] = bond.Aromatic
	}
	return func() tea.Msg {
		return crust.SubmitMsg{
			Component: "molecule_viewer",
			Data:      payload,
		}
	}
}

func cancelCmd() tea.Cmd {
	return func() tea.Msg {
		return crust.CancelMsg{
			Component: "molecule_viewer",
			Reason:    "user cancelled",
		}
	}
}

func (m Model) groupsForAtom(index int) []string {
	seen := map[string]bool{}
	var names []string
	for _, group := range m.molecule.FunctionalGroups() {
		for _, atom := range group.Atoms {
			if atom == index && !seen[group.Name] {
				seen[group.Name] = true
				names = append(names, group.Name)
				break
			}
		}
	}
	return names
}

func (m Model) pickBestSearchMatch(query string, matches []int) int {
	best := matches[0]
	bestScore := math.Inf(-1)
	for _, idx := range matches {
		atom := m.molecule.Atoms[idx]
		score := 0.0
		label := strings.ToLower(m.molecule.AtomLabel(idx))
		name := strings.ToLower(m.molecule.atomName(idx))
		symbol := strings.ToLower(atom.Symbol)
		switch {
		case strings.Contains(label, query):
			score += 5
		case strings.Contains(name, query):
			score += 4
		case strings.Contains(symbol, query):
			score += 4
		}
		if atom.Symbol != "C" && atom.Symbol != "H" {
			score += 2
		}
		if atom.Charge != 0 || math.Abs(atom.PartialCharge) > 0.15 {
			score += 1
		}
		if atom.Aromatic {
			score += 0.25
		}
		if score > bestScore {
			best = idx
			bestScore = score
		}
	}
	return best
}

func (m Model) useAdjacencyView(width int) bool {
	if width < 64 {
		return true
	}
	if m.height > 0 && m.height < 18 {
		return true
	}
	return len(m.molecule.Atoms) > 48
}

func (m Model) diagramHeight() int {
	if m.height <= 0 {
		return 14
	}
	height := m.height - 10
	if height < 8 {
		height = 8
	}
	if height > 18 {
		height = 18
	}
	return height
}

func (m Model) modeLabel() string {
	switch m.mode {
	case ViewModeHetero:
		return "heteroatoms"
	case ViewModeAromaticity:
		return "aromaticity"
	case ViewModePartialCharge:
		return "partial charge"
	case ViewModeScaffold:
		return "scaffold"
	default:
		return "identity"
	}
}

func (m Model) findBondIndex(a, b int) int {
	if a < 0 || b < 0 {
		return -1
	}
	for i, bond := range m.molecule.Bonds {
		if (bond.From == a && bond.To == b) || (bond.From == b && bond.To == a) {
			return i
		}
	}
	return -1
}

func directionVector(direction string) (float64, float64) {
	switch direction {
	case "left":
		return -1, 0
	case "right":
		return 1, 0
	case "up":
		return 0, 1
	default:
		return 0, -1
	}
}

func lastRune(s string) (rune, int) {
	return utf8.DecodeLastRuneInString(s)
}
