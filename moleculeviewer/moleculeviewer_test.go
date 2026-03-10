package moleculeviewer

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

func TestNewWithSMILES(t *testing.T) {
	m := New(WithSMILES("CCO"), WithName("Ethanol"))
	if m.Err() != nil {
		t.Fatalf("expected no load error, got %v", m.Err())
	}
	if m.title != "Ethanol" {
		t.Fatalf("expected title Ethanol, got %q", m.title)
	}
	if m.selectedAtom != 0 {
		t.Fatalf("expected initial selection at atom 0, got %d", m.selectedAtom)
	}
}

func TestNavigationUsesCoordinates(t *testing.T) {
	m := New(WithMOL(sampleMolBlock()))

	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = updated.(Model)
	if m.selectedAtom != 1 {
		t.Fatalf("expected focus to move to atom 1, got %d", m.selectedAtom)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
	m = updated.(Model)
	if m.selectedAtom != 2 {
		t.Fatalf("expected focus to move to atom 2, got %d", m.selectedAtom)
	}
}

func TestSearchModeFocusesMatches(t *testing.T) {
	m := New(WithSMILES("CCO"))

	updated, _ := m.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	m = updated.(Model)
	if !m.searching {
		t.Fatal("expected search mode to open")
	}

	for _, key := range []tea.KeyPressMsg{
		{Text: "h", Code: 'h'},
		{Text: "y", Code: 'y'},
		{Text: "d", Code: 'd'},
		{Text: "r", Code: 'r'},
		{Text: "o", Code: 'o'},
		{Text: "x", Code: 'x'},
		{Text: "y", Code: 'y'},
		{Text: "l", Code: 'l'},
	} {
		updated, _ = m.Update(key)
		m = updated.(Model)
	}

	updated, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = updated.(Model)
	if m.searching {
		t.Fatal("expected search mode to close after enter")
	}
	if m.selectedAtom != 2 {
		t.Fatalf("expected hydroxyl search to focus oxygen atom, got %d", m.selectedAtom)
	}
	if !m.searchMatches[2] {
		t.Fatal("expected focused atom to remain marked as a search match")
	}
}

func TestSubmitAndCancel(t *testing.T) {
	m := New(WithSMILES("CCO"))

	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected submit command")
	}
	msg := cmd()
	submit, ok := msg.(crust.SubmitMsg)
	if !ok {
		t.Fatalf("expected crust.SubmitMsg, got %T", msg)
	}
	if submit.Component != "molecule_viewer" {
		t.Fatalf("expected component molecule_viewer, got %q", submit.Component)
	}
	if submit.Data["atom_label"] != "C1" {
		t.Fatalf("expected initial atom label C1, got %v", submit.Data["atom_label"])
	}

	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if cmd == nil {
		t.Fatal("expected cancel command")
	}
	if _, ok := cmd().(crust.CancelMsg); !ok {
		t.Fatalf("expected crust.CancelMsg, got %T", cmd())
	}
}

func TestRenderIncludesDiagramAndFallback(t *testing.T) {
	m := New(WithMOL(sampleMolBlock()), WithName("Ethanol"), WithWidth(96), WithHeight(24))
	rendered := stripANSI(m.Render())
	for _, want := range []string{"Ethanol", "C1", "Neighbors:"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected render to contain %q", want)
		}
	}

	m.SetWidth(50)
	fallback := stripANSI(m.Render())
	if !strings.Contains(fallback, "Layout: Adjacency view") {
		t.Fatalf("expected narrow render to fall back to adjacency rendering, got:\n%s", fallback)
	}
}

func TestHelpToggle(t *testing.T) {
	m := New(WithSMILES("CCO"))
	updated, _ := m.Update(tea.KeyPressMsg{Text: "?", Code: '?'})
	m = updated.(Model)
	if !m.showHelp {
		t.Fatal("expected help to open")
	}
	help := stripANSI(m.Render())
	if !strings.Contains(help, "Arrow keys navigate the graph") {
		t.Fatalf("expected help text in render output, got:\n%s", help)
	}
}

func TestBondGlyphUsesMinimalAlphabet(t *testing.T) {
	m := New()
	for _, tc := range []struct {
		dx   int
		dy   int
		want string
	}{
		{dx: 4, dy: 0, want: "─"},
		{dx: 0, dy: 4, want: "│"},
		{dx: 4, dy: 3, want: "╲"},
		{dx: 4, dy: -3, want: "╱"},
	} {
		if got := m.bondGlyph(tc.dx, tc.dy, Bond{Order: 3, Aromatic: true}); got != tc.want {
			t.Fatalf("bondGlyph(%d,%d) = %q, want %q", tc.dx, tc.dy, got, tc.want)
		}
	}
}

func TestIdentityBondColorTracksBondOrder(t *testing.T) {
	m := New()
	if got := fmt.Sprint(m.bondColor(Bond{Order: 1})); got != fmt.Sprint(m.theme.Bond) {
		t.Fatalf("single bond color = %v, want %v", got, m.theme.Bond)
	}
	if got := fmt.Sprint(m.bondColor(Bond{Order: 2})); got != fmt.Sprint(m.theme.DoubleBond) {
		t.Fatalf("double bond color = %v, want %v", got, m.theme.DoubleBond)
	}
	if got := fmt.Sprint(m.bondColor(Bond{Order: 3})); got != fmt.Sprint(m.theme.TripleBond) {
		t.Fatalf("triple bond color = %v, want %v", got, m.theme.TripleBond)
	}
	if got := fmt.Sprint(m.bondColor(Bond{Order: 1, Aromatic: true})); got != fmt.Sprint(m.theme.AromaticBond) {
		t.Fatalf("aromatic bond color = %v, want %v", got, m.theme.AromaticBond)
	}
}

func TestProjectionPrefersGridAlignedOrientation(t *testing.T) {
	mol := Molecule{
		Atoms: []Atom{
			{Symbol: "C", Coords: [2]float64{0, 0}},
			{Symbol: "C", Coords: [2]float64{1, 0.2}},
			{Symbol: "O", Coords: [2]float64{2, 0.4}},
		},
		Bonds: []Bond{
			{From: 0, To: 1, Order: 1},
			{From: 1, To: 2, Order: 1},
		},
	}
	mol.Normalize()

	m := New(WithMolecule(mol), WithWidth(72), WithHeight(20))
	projection := m.projectAtoms(68, 12)
	if projection.collisions != 0 {
		t.Fatalf("expected collision-free projection, got %d collisions", projection.collisions)
	}
	sameRow := projection.positions[0][1] == projection.positions[1][1] && projection.positions[1][1] == projection.positions[2][1]
	sameCol := projection.positions[0][0] == projection.positions[1][0] && projection.positions[1][0] == projection.positions[2][0]
	if !sameRow && !sameCol {
		t.Fatalf("expected grid-aligned projection, got positions %+v", projection.positions)
	}
}

func TestLayoutMoleculeSeparatesBranchedChildren(t *testing.T) {
	mol, err := ParseSMILES("CC(C)(C)O")
	if err != nil {
		t.Fatalf("ParseSMILES returned error: %v", err)
	}
	LayoutMolecule(&mol)

	center := -1
	for i, atom := range mol.Atoms {
		if atom.Symbol == "C" && len(atom.Neighbors) == 4 {
			center = i
			break
		}
	}
	if center < 0 {
		t.Fatal("expected a tetra-substituted carbon center")
	}

	var angles []float64
	for _, neighbor := range mol.Atoms[center].Neighbors {
		dx := mol.Atoms[neighbor].Coords[0] - mol.Atoms[center].Coords[0]
		dy := mol.Atoms[neighbor].Coords[1] - mol.Atoms[center].Coords[1]
		angles = append(angles, math.Atan2(dy, dx))
	}

	minSep := math.Pi
	for i := 0; i < len(angles); i++ {
		for j := i + 1; j < len(angles); j++ {
			sep := angularDistance(angles[i], angles[j])
			if sep < minSep {
				minSep = sep
			}
		}
	}
	if minSep < 0.35 {
		t.Fatalf("expected branched children to occupy distinct sectors, smallest separation %.2f rad", minSep)
	}
}

func stripANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}
