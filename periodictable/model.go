// Package periodictable provides an interactive periodic table component for
// Bubble Tea applications.
//
// The component renders the full 118-element table, supports gap-aware cursor
// navigation, exposes multiple cell lenses, and returns crust.SubmitMsg /
// crust.CancelMsg from Update when the interaction completes.
package periodictable

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

const componentKey = "periodic_table"

type viewMode int

const (
	viewModeSymbol viewMode = iota
	viewModeMass
	viewModeElectronegativity
	viewModeElectronConfig
)

// Model is the Bubble Tea model for the periodic table.
type Model struct {
	width         int
	theme         Theme
	focusedNumber int
	highlights    map[string]struct{}
	mode          viewMode
	showHelp      bool
}

// New creates a PeriodicTable with the given options.
func New(opts ...Option) Model {
	m := Model{
		width:         118,
		theme:         DefaultTheme(),
		focusedNumber: 1,
		highlights:    map[string]struct{}{},
		mode:          viewModeSymbol,
	}
	for _, opt := range opts {
		opt(&m)
	}
	if _, ok := elementByNumber(m.focusedNumber); !ok {
		m.focusedNumber = 1
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
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key := msg.String(); key {
	case "left":
		m.moveHorizontal(-1)
	case "right":
		m.moveHorizontal(1)
	case "up":
		m.moveVertical(-1)
	case "down":
		m.moveVertical(1)
	case "tab":
		m.mode = m.mode.next(1)
	case "shift+tab":
		m.mode = m.mode.next(-1)
	case "enter":
		return m, submitCmd(m.focusedNumber)
	case "esc":
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		return m, cancelCmd()
	case "?":
		m.showHelp = !m.showHelp
	default:
		if len(key) == 1 && key[0] >= '1' && key[0] <= '7' {
			m.jumpToPeriod(int(key[0] - '0'))
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the table as a plain string.
func (m Model) Render() string {
	return m.render()
}

func (m Model) render() string {
	width := m.renderWidth()
	cellWidth := m.cellWidth(width)
	tableWidth := m.gridWidth(cellWidth)

	var parts []string
	parts = append(parts, m.renderHeader(width))
	parts = append(parts, m.renderGroupHeader(cellWidth))
	for _, row := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9} {
		parts = append(parts, m.renderRow(row, cellWidth))
	}
	parts = append(parts, lipgloss.NewStyle().Foreground(m.theme.Border).Render(strings.Repeat("-", tableWidth)))
	parts = append(parts, m.renderDetailPanel(width))
	if m.showHelp {
		parts = append(parts, m.renderHelp(width))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(w int) { m.width = w }

// SetHighlights replaces the externally highlighted element set.
func (m *Model) SetHighlights(symbols []string) {
	m.highlights = highlightSet(symbols)
}

// Selected returns the currently focused element.
func (m Model) Selected() *Element {
	element, ok := elementByNumber(m.focusedNumber)
	if !ok {
		return nil
	}
	copy := element
	return &copy
}

func (m Model) renderWidth() int {
	if m.width < 76 {
		return 76
	}
	return m.width
}

func (m Model) cellWidth(width int) int {
	cellWidth := (width - 20) / 18
	if cellWidth < 3 {
		return 3
	}
	if cellWidth > 6 {
		return 6
	}
	return cellWidth
}

func (m Model) gridWidth(cellWidth int) int {
	return 21 + (18 * cellWidth)
}

func (m Model) renderHeader(width int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Width(width).
		Align(lipgloss.Center)
	subtitleStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Width(width).
		Align(lipgloss.Center)

	current := m.Selected()
	focusLabel := "No selection"
	if current != nil {
		focusLabel = fmt.Sprintf("%s [%s] | %s lens", current.Name, current.Symbol, m.mode.label())
	}

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("Periodic Table"),
		subtitleStyle.Render(focusLabel),
	)
}

func (m Model) renderGroupHeader(cellWidth int) string {
	focusedGroup := 0
	if current := m.Selected(); current != nil {
		focusedGroup = current.Group
	}

	var cells []string
	cells = append(cells, "   ")
	for col := 1; col <= 18; col++ {
		style := lipgloss.NewStyle().
			Width(cellWidth).
			Align(lipgloss.Center).
			Foreground(m.theme.Border)
		if focusedGroup != 0 && col == focusedGroup {
			style = style.Foreground(m.theme.Cursor).Bold(true)
		}
		cells = append(cells, style.Render(strconv.Itoa(col)))
	}
	return strings.Join(cells, " ")
}

func (m Model) renderRow(row, cellWidth int) string {
	focusedRow := 0
	if current := m.Selected(); current != nil {
		focusedRow = current.Period
		switch current.Category {
		case categoryLanthanide:
			focusedRow = 8
		case categoryActinide:
			focusedRow = 9
		}
	}

	label := rowLabel(row)
	labelStyle := lipgloss.NewStyle().
		Width(2).
		Align(lipgloss.Right).
		Foreground(m.theme.Border)
	if row == focusedRow {
		labelStyle = labelStyle.Foreground(m.theme.Cursor).Bold(true)
	}

	var cells []string
	cells = append(cells, labelStyle.Render(label))
	for col := 1; col <= 18; col++ {
		if element, ok := elementAt(row, col); ok {
			cells = append(cells, m.renderCell(element, cellWidth))
			continue
		}
		cells = append(cells, strings.Repeat(" ", cellWidth))
	}
	return strings.Join(cells, " ")
}

func (m Model) renderCell(element Element, cellWidth int) string {
	content := truncateCell(m.mode.cellContent(element), cellWidth)

	style := lipgloss.NewStyle().
		Width(cellWidth).
		Align(lipgloss.Center).
		Foreground(m.categoryColor(element.Category))

	if _, ok := m.highlights[element.Symbol]; ok {
		style = style.Background(m.theme.Selected).Foreground(m.theme.Text).Bold(true)
	}

	if element.Number == m.focusedNumber {
		style = style.Background(m.theme.Cursor).Foreground(m.theme.Text).Bold(true)
	}

	return style.Render(content)
}

func (m Model) renderDetailPanel(width int) string {
	element := m.Selected()
	if element == nil {
		return ""
	}

	lineStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	firstLine := strings.Join([]string{
		fmt.Sprintf("[%s] %s", element.Symbol, element.Name),
		fmt.Sprintf("Atomic #%d", element.Number),
		fmt.Sprintf("Group %s, Period %d", groupLabel(*element), element.Period),
		humanizeCategory(element.Category),
	}, " | ")

	secondLine := strings.Join([]string{
		fmt.Sprintf("Mass: %s u", formatMass(element.AtomicMass)),
		fmt.Sprintf("Electronegativity: %s", formatElectronegativity(element.Electronegativity)),
		fmt.Sprintf("Config: %s", element.ElectronConfig),
	}, " | ")

	thirdParts := []string{
		fmt.Sprintf("vdW: %s", formatRadius(element.VdwRadius)),
		fmt.Sprintf("Covalent: %s", formatRadius(element.CovalentRadius)),
		fmt.Sprintf("View: %s", m.mode.label()),
	}
	if _, ok := m.highlights[element.Symbol]; ok {
		thirdParts = append(thirdParts, "Highlighted")
	}
	if len(m.highlights) > 0 {
		thirdParts = append(thirdParts, fmt.Sprintf("Highlights: %d", len(m.highlights)))
	}
	thirdLine := strings.Join(thirdParts, " | ")

	panelWidth := width
	for _, line := range []string{firstLine, secondLine, thirdLine} {
		if panelWidth < lipgloss.Width(line) {
			panelWidth = lipgloss.Width(line)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lineStyle.Width(panelWidth).Render(truncateText(firstLine, panelWidth)),
		mutedStyle.Width(panelWidth).Render(truncateText(secondLine, panelWidth)),
		mutedStyle.Width(panelWidth).Render(truncateText(thirdLine, panelWidth)),
	)
}

func (m Model) renderHelp(width int) string {
	helpWidth := width - 2
	if helpWidth < 48 {
		helpWidth = 48
	}

	body := strings.Join([]string{
		"Arrows move through the table while skipping layout gaps.",
		"Tab cycles lenses: symbol, mass, electronegativity, electron config.",
		"1-7 jumps directly to a period. Enter submits the focused element.",
		"Esc closes help first, then cancels. ? toggles this overlay.",
	}, "\n")

	asciiBorder := lipgloss.Border{
		Top:         "-",
		Bottom:      "-",
		Left:        "|",
		Right:       "|",
		TopLeft:     "+",
		TopRight:    "+",
		BottomLeft:  "+",
		BottomRight: "+",
	}

	return lipgloss.NewStyle().
		Border(asciiBorder).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Width(helpWidth).
		Foreground(m.theme.TextMuted).
		Render(body)
}

func (m Model) categoryColor(category string) color.Color {
	switch category {
	case categoryAlkaliMetal:
		return m.theme.AlkaliMetal
	case categoryAlkalineEarth:
		return m.theme.AlkalineEarth
	case categoryTransitionMetal:
		return m.theme.TransitionMetal
	case categoryPostTransition:
		return m.theme.PostTransition
	case categoryMetalloid:
		return m.theme.Metalloid
	case categoryNonmetal:
		return m.theme.Nonmetal
	case categoryHalogen:
		return m.theme.Halogen
	case categoryNobleGas:
		return m.theme.NobleGas
	case categoryLanthanide:
		return m.theme.Lanthanide
	case categoryActinide:
		return m.theme.Actinide
	default:
		return m.theme.Text
	}
}

func (m *Model) moveHorizontal(delta int) {
	row, col, ok := elementPosition(m.focusedNumber)
	if !ok {
		return
	}
	for nextCol := col + delta; nextCol >= 1 && nextCol <= 18; nextCol += delta {
		if element, ok := elementAt(row, nextCol); ok {
			m.focusedNumber = element.Number
			return
		}
	}
}

func (m *Model) moveVertical(delta int) {
	row, col, ok := elementPosition(m.focusedNumber)
	if !ok {
		return
	}
	for nextRow := row + delta; nextRow >= 1 && nextRow <= 9; nextRow += delta {
		if element, ok := elementAt(nextRow, col); ok {
			m.focusedNumber = element.Number
			return
		}
	}
}

func (m *Model) jumpToPeriod(period int) {
	_, preferredCol, ok := elementPosition(m.focusedNumber)
	if !ok {
		preferredCol = 1
	}
	if element, ok := closestElementInRow(period, preferredCol); ok {
		m.focusedNumber = element.Number
	}
}

func (mode viewMode) next(delta int) viewMode {
	size := int(viewModeElectronConfig) + 1
	next := (int(mode) + delta) % size
	if next < 0 {
		next += size
	}
	return viewMode(next)
}

func (mode viewMode) label() string {
	switch mode {
	case viewModeMass:
		return "atomic mass"
	case viewModeElectronegativity:
		return "electronegativity"
	case viewModeElectronConfig:
		return "electron config"
	default:
		return "symbol"
	}
}

func (mode viewMode) cellContent(element Element) string {
	switch mode {
	case viewModeMass:
		return compactMass(element.AtomicMass)
	case viewModeElectronegativity:
		return compactElectronegativity(element.Electronegativity)
	case viewModeElectronConfig:
		if token := lastConfigToken(element.ElectronConfig); token != "" {
			return token
		}
		return element.Symbol
	default:
		return element.Symbol
	}
}

func submitCmd(number int) tea.Cmd {
	element, ok := elementByNumber(number)
	if !ok {
		return nil
	}
	return func() tea.Msg {
		return crust.SubmitMsg{
			Component: componentKey,
			Data: map[string]any{
				"number":   element.Number,
				"symbol":   element.Symbol,
				"name":     element.Name,
				"group":    element.Group,
				"period":   element.Period,
				"category": element.Category,
			},
		}
	}
}

func cancelCmd() tea.Cmd {
	return func() tea.Msg {
		return crust.CancelMsg{
			Component: componentKey,
			Reason:    "user cancelled",
		}
	}
}

func closestElementInRow(row, preferredCol int) (Element, bool) {
	if row < 1 || row > 9 {
		return Element{}, false
	}
	for distance := 0; distance <= 17; distance++ {
		left := preferredCol - distance
		if left >= 1 {
			if element, ok := elementAt(row, left); ok {
				return element, true
			}
		}
		if distance == 0 {
			continue
		}
		right := preferredCol + distance
		if right <= 18 {
			if element, ok := elementAt(row, right); ok {
				return element, true
			}
		}
	}
	return Element{}, false
}

func rowLabel(row int) string {
	switch row {
	case 8:
		return "Ln"
	case 9:
		return "An"
	default:
		return strconv.Itoa(row)
	}
}

func groupLabel(element Element) string {
	if element.Group == 0 {
		return "f-block"
	}
	return strconv.Itoa(element.Group)
}

func humanizeCategory(category string) string {
	parts := strings.Split(category, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func compactMass(value float64) string {
	switch {
	case value >= 100:
		return trimFloat(value, 1)
	case value >= 10:
		return trimFloat(value, 2)
	default:
		return trimFloat(value, 3)
	}
}

func formatMass(value float64) string {
	return trimFloat(value, 5)
}

func compactElectronegativity(value float64) string {
	if value == 0 {
		return "?"
	}
	return trimFloat(value, 2)
}

func formatElectronegativity(value float64) string {
	if value == 0 {
		return "n/a"
	}
	return trimFloat(value, 2)
}

func formatRadius(value float64) string {
	if value == 0 {
		return "n/a"
	}
	return trimFloat(value, 2) + " A"
}

func trimFloat(value float64, precision int) string {
	text := fmt.Sprintf("%.*f", precision, value)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	return text
}

func lastConfigToken(config string) string {
	fields := strings.Fields(config)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}

func truncateCell(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	return string(runes[:width])
}

func truncateText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= width {
		return text
	}
	if width <= 3 {
		return string(runes[:width])
	}
	return string(runes[:width-3]) + "..."
}
