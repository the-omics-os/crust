// Package ontologybrowser provides an interactive ontology tree browser for
// Bubble Tea applications.
package ontologybrowser

import (
	"fmt"
	"image/color"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

const (
	componentName = "ontology_browser"
	defaultWidth  = 84
	defaultHeight = 22
	minWidth      = 48
	minHeight     = 16
	searchCharCap = 120
)

// ExpandMsg tells the host to fetch children for the given node ID.
type ExpandMsg struct {
	NodeID string
}

type pane int

const (
	paneTree pane = iota
	paneSearch
)

type visibleNode struct {
	NodeID string
	Depth  int
}

type searchResult struct {
	NodeID string
	Score  int
}

// Model is the Bubble Tea model for the ontology browser.
type Model struct {
	roots    []OntologyNode
	width    int
	height   int
	theme    Theme
	viewport viewport.Model

	searchQuery string
	activePane  pane
	helpVisible bool

	visible       []visibleNode
	selectedIndex int

	searchResults []searchResult
	searchIndex   int
	searchOffset  int

	expanded map[string]bool
	loading  map[string]bool
}

// New creates a new OntologyBrowser with the given options.
func New(opts ...Option) Model {
	vp := viewport.New(
		viewport.WithWidth(defaultWidth-4),
		viewport.WithHeight(8),
	)
	vp.SoftWrap = true
	vp.FillHeight = true
	vp.MouseWheelEnabled = true
	vp.MouseWheelDelta = 3

	m := Model{
		width:      defaultWidth,
		height:     defaultHeight,
		theme:      DefaultTheme(),
		viewport:   vp,
		activePane: paneTree,
		expanded:   map[string]bool{},
		loading:    map[string]bool{},
	}

	for _, opt := range opts {
		opt(&m)
	}

	m.roots = normalizeNodes(m.roots, 0)
	m.syncLayout()
	m.rebuild()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.SetWidth(msg.Width)
		m.SetHeight(msg.Height)
		return m, nil
	case tea.KeyPressMsg:
		return m.updateKey(msg)
	default:
		return m, nil
	}
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the browser as a plain string.
func (m Model) Render() string {
	return m.render()
}

// SetRoots replaces the root ontology nodes.
func (m *Model) SetRoots(nodes []OntologyNode) {
	m.roots = normalizeNodes(cloneNodes(nodes), 0)
	m.expanded = map[string]bool{}
	m.loading = map[string]bool{}
	m.selectedIndex = 0
	m.searchIndex = 0
	m.searchOffset = 0
	m.rebuild()
}

// SetChildren attaches loaded children to a node.
func (m *Model) SetChildren(nodeID string, children []OntologyNode) {
	node := findNodeByID(m.roots, nodeID)
	if node == nil {
		return
	}

	shouldExpand := m.expanded[nodeID] || m.loading[nodeID]
	normalized := normalizeNodes(cloneNodes(children), node.Depth+1)
	if normalized == nil {
		normalized = []OntologyNode{}
	}

	node.Children = normalized
	node.Loaded = true
	delete(m.loading, nodeID)
	if len(normalized) == 0 || !shouldExpand {
		delete(m.expanded, nodeID)
	} else {
		m.expanded[nodeID] = true
	}

	m.rebuild()
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(w int) {
	if w > 0 {
		m.width = w
	}
	m.syncLayout()
}

// SetHeight updates the rendering height.
func (m *Model) SetHeight(h int) {
	if h > 0 {
		m.height = h
	}
	m.syncLayout()
}

// Width returns the current rendering width.
func (m Model) Width() int { return m.width }

// Height returns the current rendering height.
func (m Model) Height() int { return m.height }

// Selected returns the currently highlighted node.
func (m Model) Selected() *OntologyNode {
	node := m.currentNode()
	if node == nil {
		return nil
	}
	cloned := cloneNode(*node)
	return &cloned
}

func (m Model) updateKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?":
		m.helpVisible = !m.helpVisible
		return m, nil
	case "ctrl+c":
		return m, cancelCmd()
	case "esc":
		if m.helpVisible {
			m.helpVisible = false
			return m, nil
		}
		if m.activePane == paneSearch {
			return m.focusTree()
		}
		return m, cancelCmd()
	}

	if m.helpVisible {
		return m, nil
	}

	switch msg.String() {
	case "tab":
		if m.activePane == paneTree {
			return m.focusSearch()
		}
		return m.focusTree()
	case "/":
		if m.activePane == paneTree {
			return m.focusSearch()
		}
	}

	if m.activePane == paneSearch {
		return m.updateSearchKey(msg)
	}
	return m.updateTreeKey(msg)
}

func (m Model) focusSearch() (tea.Model, tea.Cmd) {
	if m.activePane == paneSearch {
		return m, nil
	}

	m.activePane = paneSearch
	m.rebuildSearchResults()
	return m, nil
}

func (m Model) focusTree() (tea.Model, tea.Cmd) {
	m.activePane = paneTree
	return m, nil
}

func (m Model) updateSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		if len(m.searchResults) == 0 {
			return m, nil
		}
		if m.searchIndex > 0 {
			m.searchIndex--
			m.syncSearchOffset()
			m.selectNodeByID(m.searchResults[m.searchIndex].NodeID)
		}
		return m, nil
	case "down":
		if len(m.searchResults) == 0 {
			return m, nil
		}
		if m.searchIndex < len(m.searchResults)-1 {
			m.searchIndex++
			m.syncSearchOffset()
			m.selectNodeByID(m.searchResults[m.searchIndex].NodeID)
		}
		return m, nil
	case "enter":
		if len(m.searchResults) == 0 || m.searchIndex >= len(m.searchResults) {
			return m, nil
		}
		nodeID := m.searchResults[m.searchIndex].NodeID
		node := findNodeByID(m.roots, nodeID)
		if node == nil {
			return m, nil
		}
		return m, submitNodeCmd(*node, pathToNode(m.roots, nodeID))
	case "backspace", "ctrl+h":
		m.setSearchQuery(trimLastRune(m.searchQuery))
		return m, nil
	case "ctrl+w":
		m.setSearchQuery(trimLastWord(m.searchQuery))
		return m, nil
	case "ctrl+u":
		m.setSearchQuery("")
		return m, nil
	}

	if msg.Text != "" {
		m.setSearchQuery(m.searchQuery + msg.Text)
	}
	return m, nil
}

func (m Model) updateTreeKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.selectIndex(m.selectedIndex - 1)
	case "down":
		m.selectIndex(m.selectedIndex + 1)
	case "home":
		m.selectIndex(0)
	case "end":
		m.selectIndex(len(m.visible) - 1)
	case "pgup":
		m.selectIndex(m.selectedIndex - maxInt(1, m.viewport.Height()-1))
	case "pgdown":
		m.selectIndex(m.selectedIndex + maxInt(1, m.viewport.Height()-1))
	case "left":
		m.collapseOrSelectParent()
	case "right", "enter":
		return m.expandOrSelectCurrent()
	}
	return m, nil
}

func (m Model) expandOrSelectCurrent() (tea.Model, tea.Cmd) {
	node := m.currentNode()
	if node == nil {
		return m, nil
	}

	if m.loading[node.ID] {
		return m, nil
	}

	if node.Loaded && len(node.Children) == 0 {
		return m, submitNodeCmd(*node, pathToNode(m.roots, node.ID))
	}

	if node.Loaded && len(node.Children) > 0 {
		if !m.expanded[node.ID] {
			m.expanded[node.ID] = true
			m.rebuild()
		}
		return m, nil
	}

	m.expanded[node.ID] = true
	m.loading[node.ID] = true
	m.rebuild()
	return m, expandCmd(node.ID)
}

func (m *Model) collapseOrSelectParent() {
	node := m.currentNode()
	if node == nil {
		return
	}

	if m.expanded[node.ID] {
		delete(m.expanded, node.ID)
		delete(m.loading, node.ID)
		m.rebuild()
		return
	}

	parent := parentOfNode(m.roots, node.ID)
	if parent != nil {
		m.selectNodeByID(parent.ID)
	}
}

func (m *Model) selectIndex(index int) {
	if len(m.visible) == 0 {
		m.selectedIndex = 0
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.visible) {
		index = len(m.visible) - 1
	}
	m.selectedIndex = index
	m.viewport.EnsureVisible(m.selectedIndex, 0, 0)
}

func (m *Model) selectNodeByID(nodeID string) {
	if idx := m.visibleIndex(nodeID); idx >= 0 {
		m.selectIndex(idx)
	}
}

func (m Model) currentNode() *OntologyNode {
	if len(m.visible) == 0 || m.selectedIndex < 0 || m.selectedIndex >= len(m.visible) {
		return nil
	}
	return findNodeByID(m.roots, m.visible[m.selectedIndex].NodeID)
}

func (m Model) visibleIndex(nodeID string) int {
	for i, entry := range m.visible {
		if entry.NodeID == nodeID {
			return i
		}
	}
	return -1
}

func (m *Model) syncLayout() {
	width := maxInt(m.width, minWidth)
	height := maxInt(m.height, minHeight)
	treeHeight, _ := m.layoutHeights()
	innerWidth := maxInt(width-4, 24)

	m.viewport.SetWidth(innerWidth)
	m.viewport.SetHeight(treeHeight)
	m.refreshTreeViewport()

	m.width = width
	m.height = height
}

func (m Model) layoutHeights() (treeHeight int, searchLines int) {
	height := maxInt(m.height, minHeight)

	switch {
	case height >= 28:
		searchLines = 6
	case height >= 22:
		searchLines = 5
	default:
		searchLines = 3
	}

	treeHeight = height - searchLines - 11
	if treeHeight < 5 {
		treeHeight = 5
	}
	return treeHeight, searchLines
}

func (m *Model) rebuild() {
	selectedID := ""
	if node := m.currentNode(); node != nil {
		selectedID = node.ID
	}

	m.visible = flattenVisibleNodes(m.roots, m.expanded)
	if selectedID != "" {
		if idx := m.visibleIndex(selectedID); idx >= 0 {
			m.selectedIndex = idx
		}
	}
	if len(m.visible) == 0 {
		m.selectedIndex = 0
	} else if m.selectedIndex >= len(m.visible) {
		m.selectedIndex = len(m.visible) - 1
	}

	m.refreshTreeViewport()
	m.rebuildSearchResults()
}

func (m *Model) rebuildSearchResults() {
	currentResultID := ""
	if len(m.searchResults) > 0 && m.searchIndex >= 0 && m.searchIndex < len(m.searchResults) {
		currentResultID = m.searchResults[m.searchIndex].NodeID
	}

	m.searchResults = searchVisibleNodes(
		m.searchQuery,
		m.visible,
		func(id string) *OntologyNode { return findNodeByID(m.roots, id) },
	)

	if len(m.searchResults) == 0 {
		m.searchIndex = 0
		m.searchOffset = 0
		return
	}

	if idx := m.searchResultIndex(currentResultID); idx >= 0 {
		m.searchIndex = idx
	} else if m.searchIndex >= len(m.searchResults) {
		m.searchIndex = len(m.searchResults) - 1
	} else if m.searchIndex < 0 {
		m.searchIndex = 0
	}

	m.syncSearchOffset()
	if m.activePane == paneSearch {
		m.selectNodeByID(m.searchResults[m.searchIndex].NodeID)
	}
}

func (m *Model) searchResultIndex(nodeID string) int {
	for i, result := range m.searchResults {
		if result.NodeID == nodeID {
			return i
		}
	}
	return -1
}

func (m *Model) syncSearchOffset() {
	_, visibleLines := m.layoutHeights()
	if len(m.searchResults) <= visibleLines {
		m.searchOffset = 0
		return
	}

	if m.searchIndex < m.searchOffset {
		m.searchOffset = m.searchIndex
	}
	if m.searchIndex >= m.searchOffset+visibleLines {
		m.searchOffset = m.searchIndex - visibleLines + 1
	}
	maxOffset := len(m.searchResults) - visibleLines
	if m.searchOffset > maxOffset {
		m.searchOffset = maxOffset
	}
	if m.searchOffset < 0 {
		m.searchOffset = 0
	}
}

func (m *Model) refreshTreeViewport() {
	lines := m.renderTreeLines()
	m.viewport.SetContentLines(lines)
	if len(m.visible) > 0 {
		m.viewport.EnsureVisible(m.selectedIndex, 0, 0)
	}
}

func (m Model) renderTreeLines() []string {
	if len(m.visible) == 0 {
		return []string{
			lipgloss.NewStyle().
				Foreground(m.theme.TextMuted).
				Render("No ontology nodes loaded."),
		}
	}

	lines := make([]string, 0, len(m.visible))
	for i, entry := range m.visible {
		node := findNodeByID(m.roots, entry.NodeID)
		if node == nil {
			continue
		}
		lines = append(lines, m.renderTreeLine(*node, i == m.selectedIndex))
	}
	return lines
}

func (m Model) render() string {
	width := maxInt(m.width, minWidth)
	if m.helpVisible {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderHeader(width),
			m.renderHelp(width),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderHeader(width),
		m.renderTreePane(width),
		m.renderSearchPane(width),
		m.renderFooter(width),
	)
}

func (m Model) renderHeader(width int) string {
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Title).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	active := "tree"
	if m.activePane == paneSearch {
		active = "search"
	}

	meta := fmt.Sprintf(
		"Active: %s  Visible: %d  Known: %d",
		active,
		len(m.visible),
		countNodes(m.roots),
	)
	if query := strings.TrimSpace(m.searchQuery); query != "" {
		meta += fmt.Sprintf("  Query: %q", query)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Width(width).Render("Ontology Browser"),
		metaStyle.Width(width).Render(meta),
	)
}

func (m Model) renderTreePane(width int) string {
	title := fmt.Sprintf("Tree  %d visible nodes", len(m.visible))
	if len(m.loading) > 0 {
		title += fmt.Sprintf("  %d loading", len(m.loading))
	}
	return m.renderPane(title, m.viewport.View(), m.activePane == paneTree, width)
}

func (m Model) renderSearchPane(width int) string {
	_, resultLines := m.layoutHeights()

	title := "Search"
	if query := strings.TrimSpace(m.searchQuery); query != "" {
		title += fmt.Sprintf("  %d matches", len(m.searchResults))
	} else {
		title += "  visible nodes only"
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderSearchInput(),
		lipgloss.JoinVertical(lipgloss.Left, m.renderSearchLines(resultLines)...),
	)
	return m.renderPane(title, body, m.activePane == paneSearch, width)
}

func (m Model) renderSearchInput() string {
	prompt := lipgloss.NewStyle().Foreground(m.theme.SearchHighlight).Bold(true).Render("Search> ")
	placeholder := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Render("Filter visible nodes by ID, name, or description")
	text := lipgloss.NewStyle().Foreground(m.theme.Text)

	if m.searchQuery == "" {
		if m.activePane == paneSearch {
			cursor := lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true).Render("│")
			return prompt + placeholder + cursor
		}
		return prompt + placeholder
	}

	rendered := prompt + text.Render(m.searchQuery)
	if m.activePane == paneSearch {
		rendered += lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true).Render("│")
	}
	return rendered
}

func (m Model) renderSearchLines(limit int) []string {
	muted := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	text := lipgloss.NewStyle().Foreground(m.theme.Text)
	idStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	highlight := lipgloss.NewStyle().Foreground(m.theme.SearchHighlight).Bold(true)
	selectedMarker := lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true)

	if limit <= 0 {
		return nil
	}

	var lines []string
	token := searchHighlightToken(m.searchQuery)

	switch {
	case strings.TrimSpace(m.searchQuery) == "":
		lines = append(lines,
			muted.Render("Type to filter currently visible nodes."),
			muted.Render("Search does not fetch hidden or unloaded children."),
		)
	case len(m.searchResults) == 0:
		lines = append(lines, muted.Render("No visible nodes match the current query."))
	default:
		end := minInt(len(m.searchResults), m.searchOffset+limit)
		for i := m.searchOffset; i < end; i++ {
			result := m.searchResults[i]
			node := findNodeByID(m.roots, result.NodeID)
			if node == nil {
				continue
			}

			marker := "  "
			if i == m.searchIndex {
				marker = selectedMarker.Render("› ")
			}

			line := marker +
				highlightMatch(node.ID, token, idStyle, highlight) +
				" " +
				highlightMatch(node.Name, token, text, highlight)
			lines = append(lines, line)
		}
	}

	for len(lines) < limit {
		lines = append(lines, "")
	}
	return lines
}

func (m Model) renderPane(title, body string, active bool, width int) string {
	borderColor := m.theme.Border
	if active {
		borderColor = m.theme.Selected
	}

	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Title).Bold(true)
	frame := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width)

	return frame.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Render(title),
		body,
	))
}

func (m Model) renderFooter(width int) string {
	node := m.currentNode()
	if node == nil {
		return lipgloss.NewStyle().
			Foreground(m.theme.TextMuted).
			Width(width).
			Render("No node selected.")
	}

	title := lipgloss.NewStyle().Foreground(m.theme.Title).Bold(true)
	muted := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Width(width)
	text := lipgloss.NewStyle().Foreground(m.theme.Text).Width(width)

	description := strings.TrimSpace(node.Description)
	if description == "" {
		description = "No description available for the current node."
	}

	path := pathLabels(pathToNode(m.roots, node.ID))
	footerLines := []string{
		title.Render(fmt.Sprintf("Selected: %s %s", node.ID, node.Name)),
		text.Render(description),
	}
	if len(path) > 0 {
		footerLines = append(footerLines, muted.Render("Path: "+strings.Join(path, " / ")))
	}

	return lipgloss.JoinVertical(lipgloss.Left, footerLines...)
}

func (m Model) renderHelp(width int) string {
	text := lipgloss.NewStyle().Foreground(m.theme.Text)
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		text.Render("Up/Down: move through visible nodes"),
		text.Render("Right or Enter: expand node, or fetch children if not loaded"),
		text.Render("Left: collapse the current branch or move to the parent"),
		text.Render("Tab or /: move focus into the search pane"),
		text.Render("Enter in search: submit the highlighted match"),
		text.Render("Esc: leave help, leave search, or cancel the browser"),
		text.Render("?: toggle this help screen"),
	)
	return m.renderPane("Help", body, false, width)
}

func (m Model) renderTreeLine(node OntologyNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)
	icon, iconColor := m.nodeGlyph(node)

	marker := "  "
	if selected {
		marker = lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true).Render("› ")
	}

	idStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	nameColor := m.theme.Branch
	if node.Loaded && len(node.Children) == 0 {
		nameColor = m.theme.Leaf
	}
	nameStyle := lipgloss.NewStyle().Foreground(nameColor)
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)
	statusStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	line := marker +
		indent +
		iconStyle.Render(icon) +
		" " +
		idStyle.Render(node.ID)
	if node.Name != "" {
		line += " " + nameStyle.Render(node.Name)
	}
	if m.loading[node.ID] {
		line += " " + statusStyle.Render("(loading)")
	}
	if selected {
		line = lipgloss.NewStyle().Bold(true).Render(line)
	}
	return line
}

func (m Model) nodeGlyph(node OntologyNode) (string, color.Color) {
	switch {
	case m.loading[node.ID]:
		return "…", lipgloss.Color("245")
	case !node.Loaded:
		return "▸", m.theme.Collapsed
	case len(node.Children) == 0:
		return "•", m.theme.Leaf
	case m.expanded[node.ID]:
		return "▾", m.theme.Expanded
	default:
		return "▸", m.theme.Collapsed
	}
}

func expandCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		return ExpandMsg{NodeID: nodeID}
	}
}

func submitNodeCmd(node OntologyNode, path []OntologyNode) tea.Cmd {
	pathIDs := pathIDs(path)
	pathNames := pathLabels(path)
	data := map[string]any{
		"id":          node.ID,
		"name":        node.Name,
		"description": node.Description,
		"depth":       node.Depth,
		"path_ids":    pathIDs,
		"path_names":  pathNames,
	}

	return func() tea.Msg {
		return crust.SubmitMsg{
			Component: componentName,
			Data:      data,
		}
	}
}

func cancelCmd() tea.Cmd {
	return func() tea.Msg {
		return crust.CancelMsg{
			Component: componentName,
			Reason:    "user cancelled",
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m *Model) setSearchQuery(query string) {
	m.searchQuery = truncateRunes(query, searchCharCap)
	m.rebuildSearchResults()
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}

func trimLastRune(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return ""
	}
	return string(runes[:len(runes)-1])
}

func trimLastWord(s string) string {
	runes := []rune(s)
	i := len(runes)
	for i > 0 && unicode.IsSpace(runes[i-1]) {
		i--
	}
	for i > 0 && !unicode.IsSpace(runes[i-1]) {
		i--
	}
	return string(runes[:i])
}
