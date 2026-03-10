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
	"sort"
	"strconv"
	"strings"
	"unicode"

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
	jumpQuery     string
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
	if m.showHelp {
		switch msg.String() {
		case "esc", "?":
			m.showHelp = false
		}
		return m, nil
	}

	switch key := msg.String(); key {
	case "left":
		m.clearJumpQuery()
		m.moveHorizontal(-1)
	case "right":
		m.clearJumpQuery()
		m.moveHorizontal(1)
	case "up":
		m.clearJumpQuery()
		m.moveVertical(-1)
	case "down":
		m.clearJumpQuery()
		m.moveVertical(1)
	case "home":
		m.clearJumpQuery()
		m.jumpToRowEdge(-1)
	case "end":
		m.clearJumpQuery()
		m.jumpToRowEdge(1)
	case "pgup":
		m.clearJumpQuery()
		m.stepPeriod(-1)
	case "pgdown":
		m.clearJumpQuery()
		m.stepPeriod(1)
	case "tab":
		m.clearJumpQuery()
		m.mode = m.mode.next(1)
	case "shift+tab":
		m.clearJumpQuery()
		m.mode = m.mode.next(-1)
	case "backspace":
		m.backspaceJumpQuery()
	case "enter":
		m.clearJumpQuery()
		return m, submitCmd(m.focusedNumber)
	case "esc":
		if m.jumpQuery != "" {
			m.clearJumpQuery()
			return m, nil
		}
		return m, cancelCmd()
	case "?":
		m.showHelp = true
	default:
		if len(key) == 1 && key[0] >= '1' && key[0] <= '7' {
			m.clearJumpQuery()
			m.jumpToPeriod(int(key[0] - '0'))
			return m, nil
		}
		if isJumpQueryKey(msg) {
			m.extendJumpQuery(msg.Text)
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
	if m.jumpQuery != "" {
		parts = append(parts, m.renderJumpPanel(width))
	}
	parts = append(parts, m.renderFooter(width))
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

	return lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("Periodic Table"),
		m.renderLensBar(width),
		m.renderFindBar(width),
	)
}

func (m Model) renderLensBar(width int) string {
	var tabs []string
	for _, mode := range []viewMode{viewModeSymbol, viewModeMass, viewModeElectronegativity, viewModeElectronConfig} {
		style := lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(m.theme.TextMuted)
		if mode == m.mode {
			style = style.Background(m.theme.Selected).Foreground(m.theme.Text).Bold(true)
		}
		tabs = append(tabs, style.Render(mode.tabLabel()))
	}

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(strings.Join(tabs, " "))
}

func (m Model) renderFindBar(width int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Foreground(m.theme.TextMuted)
	if m.jumpQuery != "" {
		style = style.Foreground(m.theme.Text)
	}
	return style.Render(truncateText(m.findBarText(), width))
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
		fmt.Sprintf("[%d] %s %s", element.Number, element.Symbol, element.Name),
		humanizeCategory(element.Category),
		fmt.Sprintf("Group %s", groupLabel(*element)),
		fmt.Sprintf("Period %d", element.Period),
	}, " | ")

	secondLine := strings.Join([]string{
		fmt.Sprintf("Grid: %s", m.mode.label()),
		fmt.Sprintf("Mass: %s u", formatMass(element.AtomicMass)),
		fmt.Sprintf("EN: %s", formatElectronegativity(element.Electronegativity)),
	}, " | ")

	thirdLine := fmt.Sprintf("Config: %s", element.ElectronConfig)

	fourthParts := []string{
		fmt.Sprintf("vdW: %s", formatRadius(element.VdwRadius)),
		fmt.Sprintf("Covalent: %s", formatRadius(element.CovalentRadius)),
	}
	if _, ok := m.highlights[element.Symbol]; ok {
		fourthParts = append(fourthParts, "In highlight set")
	} else if len(m.highlights) > 0 {
		fourthParts = append(fourthParts, fmt.Sprintf("Highlights: %d", len(m.highlights)))
	}
	fourthLine := strings.Join(fourthParts, " | ")

	panelWidth := width
	for _, line := range []string{firstLine, secondLine, thirdLine, fourthLine} {
		if panelWidth < lipgloss.Width(line) {
			panelWidth = lipgloss.Width(line)
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		lineStyle.Width(panelWidth).Render(truncateText(firstLine, panelWidth)),
		mutedStyle.Width(panelWidth).Render(truncateText(secondLine, panelWidth)),
		mutedStyle.Width(panelWidth).Render(truncateText(thirdLine, panelWidth)),
		mutedStyle.Width(panelWidth).Render(truncateText(fourthLine, panelWidth)),
	)
}

func (m Model) renderJumpPanel(width int) string {
	helpWidth := width - 2
	if helpWidth < 54 {
		helpWidth = 54
	}

	matches := rankedMatches(m.jumpQuery, 5)
	lines := []string{
		fmt.Sprintf("Find results for %q", m.jumpQuery),
	}

	if len(matches) == 0 {
		lines = append(lines, "No elements match yet")
	} else {
		lines = append(lines, fmt.Sprintf("Best match: %s [%s]", matches[0].Name, matches[0].Symbol))
		for i, element := range matches {
			prefix := "  "
			if i == 0 {
				prefix = "> "
			}
			lines = append(lines, fmt.Sprintf(
				"%s%-3s %-12s group %s period %d",
				prefix,
				element.Symbol,
				element.Name,
				groupLabel(element),
				element.Period,
			))
		}
	}

	lines = append(lines, "Backspace deletes | Esc clears | Enter selects the current element")

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
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderFooter(width int) string {
	hints := "Arrows browse | Type symbol or name to find | Tab changes grid | Enter selects | ? more keys"
	switch {
	case m.showHelp:
		hints = "Esc closes help | Enter selects the focused element"
	case m.jumpQuery != "":
		hints = "Type to refine | Backspace deletes | Enter selects | Esc stops finding"
	case width >= 110:
		hints += " | Home/End row | PgUp/PgDn period"
	}
	return lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Width(width).
		Align(lipgloss.Center).
		Render(truncateText(hints, width))
}

func (m Model) renderHelp(width int) string {
	helpWidth := width - 2
	if helpWidth < 72 {
		helpWidth = 72
	}

	lines := []string{
		"Browse: arrows follow the table geometry, including the detached Ln and An bridges",
		"Find: type a symbol or name at any time and the best match becomes the focus",
		"Step back: Esc closes help first, then clears find, then cancels the component",
		"Power keys: Home/End row edges | PgUp/PgDn periods | 1-7 exact periods | Shift+Tab reverse lens",
		"Select: Enter submits the focused element",
	}

	var rendered []string
	for _, line := range lines {
		rendered = append(rendered, truncateText(line, helpWidth-4))
	}

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
		Render(strings.Join(rendered, "\n"))
}

func (m Model) findBarText() string {
	if m.jumpQuery == "" {
		return "Find element: Type symbol or name"
	}

	best, total, ok := previewJumpQuery(m.jumpQuery)
	if !ok {
		return fmt.Sprintf("Find element: %s (no matches)", m.jumpQuery)
	}

	if element, matches, unique := resolveJumpQuery(m.jumpQuery); unique {
		return fmt.Sprintf("Find element: %s -> %s [%s]", m.jumpQuery, element.Name, element.Symbol)
	} else if matches > 1 {
		return fmt.Sprintf("Find element: %s -> %s [%s] (%d matches)", m.jumpQuery, best.Name, best.Symbol, total)
	}

	return fmt.Sprintf("Find element: %s -> %s [%s]", m.jumpQuery, best.Name, best.Symbol)
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
	if next, ok := bridgedHorizontalNumber(m.focusedNumber, delta); ok {
		m.focusedNumber = next
		return
	}

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
	if delta < 0 && row == 8 {
		if element, ok := closestElementInRow(6, col); ok {
			m.focusedNumber = element.Number
		}
		return
	}
	if delta > 0 && row == 9 {
		if element, ok := closestElementInRow(7, col); ok {
			m.focusedNumber = element.Number
		}
		return
	}
	for nextRow := row + delta; nextRow >= 1 && nextRow <= 9; nextRow += delta {
		if element, ok := elementAt(nextRow, col); ok {
			m.focusedNumber = element.Number
			return
		}
	}
}

func (m *Model) jumpToRowEdge(direction int) {
	row, _, ok := elementPosition(m.focusedNumber)
	if !ok {
		return
	}

	if direction < 0 {
		for col := 1; col <= 18; col++ {
			if element, ok := elementAt(row, col); ok {
				m.focusedNumber = element.Number
				return
			}
		}
		return
	}

	for col := 18; col >= 1; col-- {
		if element, ok := elementAt(row, col); ok {
			m.focusedNumber = element.Number
			return
		}
	}
}

func (m *Model) stepPeriod(delta int) {
	current := m.Selected()
	if current == nil {
		return
	}
	targetPeriod := current.Period + delta
	if targetPeriod < 1 || targetPeriod > 7 {
		return
	}
	_, preferredCol, ok := elementPosition(m.focusedNumber)
	if !ok {
		preferredCol = 1
	}
	if element, ok := closestElementInRow(targetPeriod, preferredCol); ok {
		m.focusedNumber = element.Number
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

func (m *Model) extendJumpQuery(text string) {
	candidate := strings.ToLower(m.jumpQuery + text)
	if m.applyJumpQuery(candidate) {
		return
	}
	if m.applyJumpQuery(strings.ToLower(text)) {
		return
	}
	m.jumpQuery = ""
}

func (m *Model) applyJumpQuery(query string) bool {
	best, _, ok := previewJumpQuery(query)
	if !ok {
		return false
	}
	m.jumpQuery = strings.ToLower(query)
	m.focusedNumber = best.Number
	return true
}

func (m *Model) backspaceJumpQuery() {
	if m.jumpQuery == "" {
		return
	}
	runes := []rune(m.jumpQuery)
	if len(runes) == 1 {
		m.jumpQuery = ""
		return
	}
	m.jumpQuery = string(runes[:len(runes)-1])
	if best, _, ok := previewJumpQuery(m.jumpQuery); ok {
		m.focusedNumber = best.Number
	}
}

func (m *Model) clearJumpQuery() {
	m.jumpQuery = ""
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

func (mode viewMode) tabLabel() string {
	switch mode {
	case viewModeMass:
		return "Mass"
	case viewModeElectronegativity:
		return "EN"
	case viewModeElectronConfig:
		return "Config"
	default:
		return "Symbol"
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

func resolveJumpQuery(query string) (Element, int, bool) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return Element{}, 0, false
	}

	if element, ok := elementBySymbol(query); ok {
		return element, 1, true
	}

	for _, element := range allElements {
		if strings.ToLower(element.Name) == query {
			return element, 1, true
		}
		if strconv.Itoa(element.Number) == query {
			return element, 1, true
		}
	}

	matches := rankedMatches(query, 32)
	if len(matches) == 1 {
		return matches[0], 1, true
	}
	return Element{}, len(matches), false
}

func previewJumpQuery(query string) (Element, int, bool) {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return Element{}, 0, false
	}

	matches := rankedMatches(query, 32)
	if len(matches) == 0 {
		return Element{}, 0, false
	}
	return matches[0], len(matches), true
}

func rankedMatches(query string, limit int) []Element {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return nil
	}

	type scored struct {
		element Element
		score   int
	}

	var matches []scored
	for _, element := range allElements {
		if score, ok := searchScore(query, element); ok {
			matches = append(matches, scored{element: element, score: score})
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score < matches[j].score
		}
		return matches[i].element.Number < matches[j].element.Number
	})

	if len(matches) > limit {
		matches = matches[:limit]
	}

	elements := make([]Element, 0, len(matches))
	for _, match := range matches {
		elements = append(elements, match.element)
	}
	return elements
}

func searchScore(query string, element Element) (int, bool) {
	symbol := strings.ToLower(element.Symbol)
	name := strings.ToLower(element.Name)
	number := strconv.Itoa(element.Number)

	switch {
	case query == symbol:
		return 0, true
	case query == name:
		return 1, true
	case query == number:
		return 2, true
	case strings.HasPrefix(symbol, query):
		return 3, true
	case strings.HasPrefix(name, query):
		return 4, true
	case strings.HasPrefix(number, query):
		return 5, true
	case strings.Contains(name, query):
		return 6, true
	default:
		return 0, false
	}
}

func bridgedHorizontalNumber(number, delta int) (int, bool) {
	switch {
	case delta > 0 && number == 56:
		return 57, true
	case delta > 0 && number >= 57 && number < 71:
		return number + 1, true
	case delta > 0 && number == 71:
		return 72, true
	case delta > 0 && number == 88:
		return 89, true
	case delta > 0 && number >= 89 && number < 103:
		return number + 1, true
	case delta > 0 && number == 103:
		return 104, true
	case delta < 0 && number == 72:
		return 71, true
	case delta < 0 && number > 57 && number <= 71:
		return number - 1, true
	case delta < 0 && number == 57:
		return 56, true
	case delta < 0 && number == 104:
		return 103, true
	case delta < 0 && number > 89 && number <= 103:
		return number - 1, true
	case delta < 0 && number == 89:
		return 88, true
	default:
		return 0, false
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

func isJumpQueryKey(msg tea.KeyPressMsg) bool {
	if msg.Mod != 0 || msg.Text == "" {
		return false
	}
	runes := []rune(msg.Text)
	return len(runes) == 1 && (unicode.IsLetter(runes[0]) || unicode.IsDigit(runes[0]))
}
