package moleculeviewer

import (
	"fmt"
	"image/color"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

type canvasCell struct {
	value    string
	priority int
}

func (m Model) render() string {
	width := maxInt(54, m.width)
	height := maxInt(18, m.height)

	title := m.title
	if strings.TrimSpace(title) == "" {
		if m.molecule.Name != "" {
			title = m.molecule.Name
		} else if m.molecule.Formula() != "" {
			title = m.molecule.Formula()
		} else {
			title = defaultTitle
		}
	}

	switch {
	case m.loadErr != nil:
		return m.renderStateBox(width, height, title, []string{
			lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("Unable to load molecule"),
			lipgloss.NewStyle().Foreground(m.theme.Negative).Render(m.loadErr.Error()),
			"",
			lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Supported inputs: SMILES, MOL blocks, and SDF payloads."),
			lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Use WithSMILES / WithMOL / WithSDF or the setter methods."),
		})
	case len(m.molecule.Atoms) == 0:
		return m.renderStateBox(width, height, title, []string{
			lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("No molecule loaded"),
			"",
			lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Load a structure with WithSMILES, WithMOL, WithSDF, or SetMolecule."),
			lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Example: CN1C=NC2=C1C(=O)N(C(=O)N2C)C"),
		})
	case m.showHelp:
		return m.renderStateBox(width, height, title, []string{
			lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("Small Molecule Viewer"),
			"",
			lipgloss.NewStyle().Foreground(m.theme.Text).Render("Arrow keys navigate the graph by bonded-neighbor screen direction."),
			lipgloss.NewStyle().Foreground(m.theme.Text).Render("Tab cycles coloring planes: identity, heteroatoms, aromaticity, charge, scaffold."),
			lipgloss.NewStyle().Foreground(m.theme.Text).Render("/: opens the search prompt. Try O, aromatic, amide, scaffold, or C=O."),
			lipgloss.NewStyle().Foreground(m.theme.Text).Render("Enter emits crust.SubmitMsg with the focused atom and active bond."),
			lipgloss.NewStyle().Foreground(m.theme.Text).Render("Esc closes help first, then cancels the overlay."),
			"",
			lipgloss.NewStyle().Foreground(m.theme.Help).Render("If the molecule is too dense for the terminal, the view falls back to an adjacency list."),
		})
	default:
		diagram, fallback := m.renderDiagram(width-2, m.diagramHeight())
		return m.renderBox(width, fmt.Sprintf("%s  [%s]", title, m.modeLabel()), diagram, m.renderInspector(fallback))
	}
}

func (m Model) renderStateBox(width, height int, title string, lines []string) string {
	targetBody := maxInt(4, height-2)
	body := append([]string(nil), lines...)
	for len(body) < targetBody {
		body = append(body, "")
	}
	if len(body) > targetBody {
		body = body[:targetBody]
	}
	return m.renderBox(width, title, body, nil)
}

func (m Model) renderBox(width int, title string, upper, lower []string) string {
	innerWidth := maxInt(20, width-2)
	borderStyle := lipgloss.NewStyle().Foreground(m.theme.Border)
	lineStyle := lipgloss.NewStyle().Width(innerWidth).MaxWidth(innerWidth)

	title = trimPlain(title, maxInt(10, innerWidth-4))
	titleStyled := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render(" " + title + " ")
	pad := innerWidth - lipgloss.Width(titleStyled)
	if pad < 0 {
		pad = 0
	}

	lines := []string{
		borderStyle.Render("┌") + titleStyled + borderStyle.Render(strings.Repeat("─", pad)) + borderStyle.Render("┐"),
	}
	for _, line := range upper {
		lines = append(lines, borderStyle.Render("│")+lineStyle.Render(line)+borderStyle.Render("│"))
	}
	if len(lower) > 0 {
		lines = append(lines, borderStyle.Render("├"+strings.Repeat("─", innerWidth)+"┤"))
		for _, line := range lower {
			lines = append(lines, borderStyle.Render("│")+lineStyle.Render(line)+borderStyle.Render("│"))
		}
	}
	lines = append(lines, borderStyle.Render("└"+strings.Repeat("─", innerWidth)+"┘"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) renderDiagram(width, height int) ([]string, bool) {
	if m.useAdjacencyView(width) || height < 6 {
		return m.renderAdjacency(width, height), true
	}

	projection := m.projectAtoms(width, height)
	positions := projection.positions
	if projection.collisions > maxInt(2, len(m.molecule.Atoms)/5) {
		return m.renderAdjacency(width, height), true
	}

	canvas := make([][]canvasCell, height)
	for y := range canvas {
		canvas[y] = make([]canvasCell, width)
		for x := range canvas[y] {
			canvas[y][x] = canvasCell{value: " "}
		}
	}

	for _, bond := range m.molecule.Bonds {
		a := positions[bond.From]
		b := positions[bond.To]
		style := lipgloss.NewStyle().Foreground(m.bondColor(bond))
		startHidden := m.diagramLabel(bond.From) == ""
		endHidden := m.diagramLabel(bond.To) == ""
		points := rasterLine(a[0], a[1], b[0], b[1])
		for idx, p := range points {
			if idx == 0 && !startHidden {
				continue
			}
			if idx == len(points)-1 && !endHidden {
				continue
			}
			// Per-step glyph: use the local step direction, not the overall bond direction
			var sdx, sdy int
			if idx < len(points)-1 {
				sdx = points[idx+1][0] - p[0]
				sdy = points[idx+1][1] - p[1]
			} else if idx > 0 {
				sdx = p[0] - points[idx-1][0]
				sdy = p[1] - points[idx-1][1]
			}
			glyph := m.bondGlyph(sdx, sdy, bond)
			drawCanvas(canvas, p[0], p[1], style.Render(glyph), 1)
		}
	}

	for idx := range m.molecule.Atoms {
		label := m.diagramLabel(idx)
		if label == "" {
			continue
		}
		style := m.atomStyle(idx)
		startX := positions[idx][0] - (len([]rune(label))-1)/2
		for off, r := range []rune(label) {
			drawCanvas(canvas, startX+off, positions[idx][1], style.Render(string(r)), 3)
		}
	}

	lines := make([]string, height)
	for y := range canvas {
		var b strings.Builder
		for x := range canvas[y] {
			b.WriteString(canvas[y][x].value)
		}
		lines[y] = b.String()
	}
	return lines, false
}

func (m Model) renderAdjacency(width, height int) []string {
	start := clampInt(maxInt(0, m.selectedAtom-height/2), 0, maxInt(0, len(m.molecule.Atoms)-height))
	lines := make([]string, 0, height)
	for i := 0; i < height && start+i < len(m.molecule.Atoms); i++ {
		atom := m.molecule.Atoms[start+i]
		prefix := " "
		if atom.Index == m.selectedAtom {
			prefix = ">"
		} else if m.searchMatches[atom.Index] {
			prefix = "*"
		}

		var neighbors []string
		for _, neighbor := range atom.Neighbors {
			bond, _ := m.molecule.BondBetween(atom.Index, neighbor)
			neighbors = append(neighbors, fmt.Sprintf("%s (%s)", m.molecule.AtomLabel(neighbor), shortBond(bond)))
		}
		sort.Strings(neighbors)

		line := fmt.Sprintf("%s %-4s q=%+d h=%d -> %s", prefix, m.molecule.AtomLabel(atom.Index), atom.Charge, atom.Hydrogens, strings.Join(neighbors, ", "))
		lines = append(lines, trimPlain(line, width))
	}
	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func (m Model) renderInspector(fallback bool) []string {
	if m.selectedAtom < 0 || m.selectedAtom >= len(m.molecule.Atoms) {
		return []string{
			"No atom focused",
			"",
			"",
			"",
			lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Arrows move  Tab changes plane  / searches  ? help"),
		}
	}

	atom := m.molecule.Atoms[m.selectedAtom]
	line1 := fmt.Sprintf(
		"[%s] %s | charge %+d | partial %+0.2f | %d bonds",
		m.molecule.AtomLabel(atom.Index),
		atomDisplayName(atom.Symbol),
		atom.Charge,
		atom.PartialCharge,
		len(atom.Neighbors),
	)
	if atom.Aromatic {
		line1 += " | aromatic"
	}
	if atom.Scaffold {
		line1 += " | scaffold"
	} else {
		line1 += " | r-group"
	}

	var neighbors []string
	for _, neighbor := range atom.Neighbors {
		bond, _ := m.molecule.BondBetween(atom.Index, neighbor)
		neighbors = append(neighbors, fmt.Sprintf("%s (%s)", m.molecule.AtomLabel(neighbor), shortBond(bond)))
	}
	sort.Strings(neighbors)
	line2 := "Neighbors: none"
	if len(neighbors) > 0 {
		line2 = "Neighbors: " + strings.Join(neighbors, ", ")
	}

	var detail []string
	if m.hoverBond >= 0 && m.hoverBond < len(m.molecule.Bonds) {
		bond := m.molecule.Bonds[m.hoverBond]
		detail = append(detail, fmt.Sprintf(
			"Bond: %s-%s (%s)",
			m.molecule.AtomLabel(bond.From),
			m.molecule.AtomLabel(bond.To),
			bondDescriptor(bond),
		))
	}
	groups := m.groupsForAtom(m.selectedAtom)
	if len(groups) > 0 {
		detail = append(detail, "Groups: "+strings.Join(groups, ", "))
	} else {
		detail = append(detail, "Groups: none")
	}
	if fallback {
		detail = append(detail, "Layout: Adjacency view")
	}
	line3 := strings.Join(detail, " | ")

	searchLine := "Search: none"
	if m.lastSearch != "" {
		searchLine = fmt.Sprintf("Search: %s (%d hit%s)", m.lastSearch, len(m.searchMatches), pluralSuffix(len(m.searchMatches)))
	}
	if m.searching {
		prompt := fmt.Sprintf("Find: /%s█", m.searchBuffer)
		if hints := m.searchHints(); len(hints) > 0 {
			prompt += "  " + strings.Join(hints, ", ")
		}
		searchLine = prompt
	}

	status := m.status
	if status == "" {
		status = "Arrows move  Tab changes plane  / searches  Enter selects  ? help"
	}

	line5 := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Foreground(m.theme.Help).Render("Mode: "+m.modeLabel()),
		"  ",
		lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render(fmt.Sprintf("%d atoms  %d bonds  %s", len(m.molecule.Atoms), len(m.molecule.Bonds), m.molecule.Formula())),
		"  ",
		lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render(status),
	)

	return []string{line1, line2, line3, searchLine, line5}
}

func (m Model) atomStyle(index int) lipgloss.Style {
	atom := m.molecule.Atoms[index]
	style := lipgloss.NewStyle().Bold(true).Foreground(m.atomColor(atom))
	if m.searchMatches[index] {
		style = style.Underline(true).Foreground(m.theme.Search)
	}
	if index == m.selectedAtom {
		style = style.Background(m.theme.Selected).Foreground(lipgloss.Color("0")).Underline(false)
	}
	return style
}

func (m Model) diagramLabel(index int) string {
	if index < 0 || index >= len(m.molecule.Atoms) {
		return ""
	}
	atom := m.molecule.Atoms[index]
	selected := index == m.selectedAtom
	matched := m.searchMatches[index]

	if atom.Symbol == "H" && atom.Charge == 0 && !selected && !matched {
		return ""
	}

	if atom.Symbol == "C" && atom.Charge == 0 {
		if selected || matched {
			return "C"
		}
		return "C"
	}

	return atom.Symbol
}

func (m Model) atomColor(atom Atom) color.Color {
	switch m.mode {
	case ViewModeHetero:
		if atom.Symbol == "C" || atom.Symbol == "H" {
			return m.theme.TextMuted
		}
		return elementColor(atom.Symbol, m.theme)
	case ViewModeAromaticity:
		if atom.Aromatic {
			return m.theme.AromaticBond
		}
		return m.theme.TextMuted
	case ViewModePartialCharge:
		switch {
		case atom.PartialCharge > 0.1:
			return m.theme.Positive
		case atom.PartialCharge < -0.1:
			return m.theme.Negative
		default:
			return m.theme.Text
		}
	case ViewModeScaffold:
		if atom.Scaffold {
			return m.theme.Scaffold
		}
		return m.theme.RGroup
	default:
		return elementColor(atom.Symbol, m.theme)
	}
}

func (m Model) bondColor(bond Bond) color.Color {
	color := m.bondOrderColor(bond)
	switch m.mode {
	case ViewModeAromaticity:
		if bond.Aromatic {
			color = m.theme.AromaticBond
		} else {
			color = m.theme.TextMuted
		}
	case ViewModePartialCharge:
		left := m.molecule.Atoms[bond.From].PartialCharge
		right := m.molecule.Atoms[bond.To].PartialCharge
		if left > 0.1 || right > 0.1 {
			color = m.theme.Positive
		} else if left < -0.1 || right < -0.1 {
			color = m.theme.Negative
		}
	case ViewModeScaffold:
		if m.molecule.Atoms[bond.From].Scaffold && m.molecule.Atoms[bond.To].Scaffold {
			color = m.theme.Scaffold
		} else {
			color = m.theme.RGroup
		}
	}

	if m.searchMatches[bond.From] && m.searchMatches[bond.To] {
		color = m.theme.Search
	}
	if m.hoverBond >= 0 && bond.Index == m.hoverBond {
		color = m.theme.Help
	}
	return color
}

func (m Model) bondOrderColor(bond Bond) color.Color {
	if bond.Aromatic {
		return m.theme.AromaticBond
	}
	switch bond.Order {
	case 3:
		return m.theme.TripleBond
	case 2:
		return m.theme.DoubleBond
	default:
		return m.theme.Bond
	}
}

func elementColor(symbol string, theme Theme) color.Color {
	switch symbol {
	case "C":
		return theme.Carbon
	case "N":
		return theme.Nitrogen
	case "O":
		return theme.Oxygen
	case "S":
		return theme.Sulfur
	case "P":
		return theme.Phosphorus
	case "F", "Cl", "Br", "I":
		return theme.Halogen
	case "H":
		return theme.Hydrogen
	case "Na", "K", "Li", "Ca", "Mg", "Zn", "Fe":
		return theme.Metal
	default:
		return theme.Text
	}
}

func (m Model) bondGlyph(dx, dy int, bond Bond) string {
	horizontal := absInt(dx) >= absInt(dy)*2
	vertical := absInt(dy) >= absInt(dx)*2

	_ = bond
	if horizontal {
		return "─"
	}
	if vertical {
		return "│"
	}
	if dx*dy >= 0 {
		return "╲"
	}
	return "╱"
}

func rasterLine(x0, y0, x1, y1 int) [][2]int {
	var points [][2]int
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx := -1
	if x0 < x1 {
		sx = 1
	}
	sy := -1
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		points = append(points, [2]int{x0, y0})
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
	return points
}

func drawCanvas(canvas [][]canvasCell, x, y int, value string, priority int) {
	if y < 0 || y >= len(canvas) || x < 0 || x >= len(canvas[y]) {
		return
	}
	if priority >= canvas[y][x].priority {
		canvas[y][x] = canvasCell{value: value, priority: priority}
	}
}

func shortBond(bond Bond) string {
	if bond.Aromatic {
		return "aro"
	}
	switch bond.Order {
	case 2:
		return "="
	case 3:
		return "#"
	default:
		return "-"
	}
}

func trimPlain(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= max {
		return s
	}
	runes := []rune(s)
	if max == 1 {
		return string(runes[:1])
	}
	if len(runes) > max-1 {
		runes = runes[:max-1]
	}
	return string(runes) + "…"
}

func (m Model) searchHints() []string {
	prefix := strings.ToLower(strings.TrimSpace(m.searchBuffer))
	candidates := []string{"C", "N", "O", "Cl", "Br", "aromatic", "hetero", "carbonyl", "amide", "amine", "hydroxyl", "halide", "scaffold", "r-group", "positive", "negative", "C=O"}
	for _, group := range m.molecule.FunctionalGroups() {
		candidates = append(candidates, group.Name)
	}

	seen := map[string]bool{}
	var hits []string
	for _, candidate := range candidates {
		if seen[candidate] {
			continue
		}
		seen[candidate] = true
		if prefix == "" || strings.HasPrefix(strings.ToLower(candidate), prefix) {
			hits = append(hits, candidate)
		}
	}
	sort.Strings(hits)
	if len(hits) > 3 {
		hits = hits[:3]
	}
	return hits
}

func pluralSuffix(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
