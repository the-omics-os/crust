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
	m.rebuild()
	m.syncLayout()
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
	m.searchQuery = ""
	m.activePane = paneTree
	m.expanded = map[string]bool{}
	m.loading = map[string]bool{}
	m.selectedIndex = 0
	m.searchIndex = 0
	m.searchResults = nil
	m.rebuild()
	m.syncLayout()
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
		switch {
		case m.helpVisible:
			m.helpVisible = false
			return m, nil
		case strings.TrimSpace(m.searchQuery) != "":
			m.clearSearch()
			return m, nil
		case m.activePane == paneSearch:
			return m.focusTree()
		default:
			return m, cancelCmd()
		}
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
		return m.focusSearch()
	}

	if m.activePane == paneTree && shouldStartSearch(msg) {
		m.activePane = paneSearch
		m.setSearchQuery(m.searchQuery + msg.Text)
		return m, nil
	}

	if m.activePane == paneSearch {
		return m.updateSearchKey(msg)
	}
	return m.updateTreeKey(msg)
}

func (m Model) focusSearch() (tea.Model, tea.Cmd) {
	m.activePane = paneSearch
	m.refreshSearchResults()
	m.syncSelectionToSearch()
	return m, nil
}

func (m Model) focusTree() (tea.Model, tea.Cmd) {
	m.activePane = paneTree
	return m, nil
}

func (m Model) updateSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "backspace", "ctrl+h":
		m.setSearchQuery(trimLastRune(m.searchQuery))
		return m, nil
	case "ctrl+w":
		m.setSearchQuery(trimLastWord(m.searchQuery))
		return m, nil
	case "ctrl+u":
		m.setSearchQuery("")
		return m, nil
	case "up":
		if m.hasSearchResults() {
			m.moveSearchIndex(-1)
			return m, nil
		}
	case "down":
		if m.hasSearchResults() {
			m.moveSearchIndex(1)
			return m, nil
		}
	case "home":
		if m.hasSearchResults() {
			m.searchIndex = 0
			m.syncSelectionToSearch()
			return m, nil
		}
	case "end":
		if m.hasSearchResults() {
			m.searchIndex = len(m.searchResults) - 1
			m.syncSelectionToSearch()
			return m, nil
		}
	case "pgup":
		if m.hasSearchResults() {
			m.moveSearchIndex(-5)
			return m, nil
		}
	case "pgdown":
		if m.hasSearchResults() {
			m.moveSearchIndex(5)
			return m, nil
		}
	case "enter":
		return m.submitCurrent()
	}

	if shouldStartSearch(msg) {
		m.setSearchQuery(m.searchQuery + msg.Text)
		return m, nil
	}

	return m.updateTreeKey(msg)
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
	case "right":
		return m.expandCurrent()
	case " ":
		return m.toggleCurrentBranch()
	case "enter":
		return m.submitCurrent()
	}
	return m, nil
}

func (m Model) expandCurrent() (tea.Model, tea.Cmd) {
	node := m.currentNode()
	if node == nil || m.loading[node.ID] {
		return m, nil
	}

	if node.Loaded && len(node.Children) == 0 {
		return m, nil
	}

	if node.Loaded && len(node.Children) > 0 {
		if !m.expanded[node.ID] {
			m.expanded[node.ID] = true
			m.refreshVisible(node.ID)
		}
		return m, nil
	}

	m.expanded[node.ID] = true
	m.loading[node.ID] = true
	m.refreshVisible(node.ID)
	return m, expandCmd(node.ID)
}

func (m Model) toggleCurrentBranch() (tea.Model, tea.Cmd) {
	node := m.currentNode()
	if node == nil || m.loading[node.ID] {
		return m, nil
	}

	if m.expanded[node.ID] {
		delete(m.expanded, node.ID)
		m.refreshVisible(node.ID)
		return m, nil
	}

	return m.expandCurrent()
}

func (m *Model) collapseOrSelectParent() {
	node := m.currentNode()
	if node == nil {
		return
	}

	if m.expanded[node.ID] {
		delete(m.expanded, node.ID)
		delete(m.loading, node.ID)
		m.refreshVisible(node.ID)
		return
	}

	parent := parentOfNode(m.roots, node.ID)
	if parent != nil {
		m.selectNodeByID(parent.ID)
	}
}

func (m Model) submitCurrent() (tea.Model, tea.Cmd) {
	node := m.currentNode()
	if node == nil {
		return m, nil
	}
	return m, submitNodeCmd(*node, pathToNode(m.roots, node.ID))
}

func (m *Model) selectIndex(index int) {
	if len(m.visible) == 0 {
		m.selectedIndex = 0
		m.refreshTreeViewport()
		return
	}

	prev := m.selectedIndex
	if index < 0 {
		index = 0
	}
	if index >= len(m.visible) {
		index = len(m.visible) - 1
	}
	m.selectedIndex = index
	if prev != m.selectedIndex {
		m.refreshTreeViewport()
		return
	}
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

func (m Model) currentNodeID() string {
	node := m.currentNode()
	if node == nil {
		return ""
	}
	return node.ID
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
	m.width = width
	m.height = height

	treeHeight := m.layoutTreeHeight(width, height)
	innerWidth := maxInt(width-4, 24)

	m.viewport.SetWidth(innerWidth)
	m.viewport.SetHeight(treeHeight)
	m.refreshTreeViewport()
}

func (m Model) layoutTreeHeight(width, height int) int {
	fixed := lipgloss.Height(m.renderHeader(width)) +
		lipgloss.Height(m.renderInspectorPane(width)) +
		lipgloss.Height(m.renderFooter(width))

	treeHeight := height - fixed - 3
	if treeHeight < 4 {
		treeHeight = 4
	}
	return treeHeight
}

func (m *Model) rebuild() {
	selectedID := m.currentNodeID()
	m.refreshVisible(selectedID)
	m.refreshSearchResults()
	m.syncSelectionToSearch()
}

func (m *Model) refreshVisible(preferredID string) {
	m.visible = flattenVisibleNodes(m.roots, m.expanded)
	if preferredID != "" {
		if idx := m.visibleIndex(preferredID); idx >= 0 {
			m.selectedIndex = idx
		}
	}
	if len(m.visible) == 0 {
		m.selectedIndex = 0
	} else if m.selectedIndex >= len(m.visible) {
		m.selectedIndex = len(m.visible) - 1
	}
	m.refreshTreeViewport()
}

func (m *Model) refreshSearchResults() {
	currentResultID := ""
	if len(m.searchResults) > 0 && m.searchIndex >= 0 && m.searchIndex < len(m.searchResults) {
		currentResultID = m.searchResults[m.searchIndex].NodeID
	}

	m.searchResults = searchKnownNodes(m.searchQuery, m.roots)

	if len(m.searchResults) == 0 {
		m.searchIndex = 0
		return
	}

	if idx := m.searchResultIndex(currentResultID); idx >= 0 {
		m.searchIndex = idx
	} else if m.searchIndex >= len(m.searchResults) {
		m.searchIndex = len(m.searchResults) - 1
	} else if m.searchIndex < 0 {
		m.searchIndex = 0
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

func (m Model) hasSearchResults() bool {
	return strings.TrimSpace(m.searchQuery) != "" && len(m.searchResults) > 0
}

func (m *Model) moveSearchIndex(delta int) {
	if len(m.searchResults) == 0 {
		return
	}

	next := m.searchIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= len(m.searchResults) {
		next = len(m.searchResults) - 1
	}
	m.searchIndex = next
	m.syncSelectionToSearch()
}

func (m *Model) syncSelectionToSearch() {
	if strings.TrimSpace(m.searchQuery) == "" || len(m.searchResults) == 0 {
		return
	}

	nodeID := m.searchResults[m.searchIndex].NodeID
	changed := m.expandPathToNode(nodeID)
	if changed {
		m.refreshVisible(nodeID)
		return
	}
	m.selectNodeByID(nodeID)
}

func (m *Model) expandPathToNode(nodeID string) bool {
	path := pathToNode(m.roots, nodeID)
	if len(path) < 2 {
		return false
	}

	changed := false
	for _, node := range path[:len(path)-1] {
		if !m.expanded[node.ID] {
			m.expanded[node.ID] = true
			changed = true
		}
	}
	return changed
}

func (m *Model) clearSearch() {
	m.searchQuery = ""
	m.searchResults = nil
	m.searchIndex = 0
	m.activePane = paneTree
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
		m.renderInspectorPane(width),
		m.renderFooter(width),
	)
}

func (m Model) renderHeader(width int) string {
	titleStyle := lipgloss.NewStyle().Foreground(m.theme.Title).Bold(true)
	metaStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	filterHintStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	var meta string
	switch {
	case strings.TrimSpace(m.searchQuery) != "" && len(m.searchResults) > 0:
		meta = fmt.Sprintf(
			"Filter %q across %d loaded terms  •  showing %d/%d",
			m.searchQuery,
			countNodes(m.roots),
			m.searchIndex+1,
			len(m.searchResults),
		)
	case strings.TrimSpace(m.searchQuery) != "":
		meta = fmt.Sprintf("Filter %q across %d loaded terms  •  no matches yet", m.searchQuery, countNodes(m.roots))
	default:
		meta = fmt.Sprintf(
			"Browsing %d loaded terms  •  %d branches open",
			countNodes(m.roots),
			len(m.expanded),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		titleStyle.Width(width).Render("Ontology Browser"),
		metaStyle.Width(width).Render(meta),
		filterHintStyle.Width(width).Render(m.renderSearchInput()),
	)
}

func (m Model) renderSearchInput() string {
	prompt := lipgloss.NewStyle().Foreground(m.theme.SearchHighlight).Bold(true).Render("Filter> ")
	placeholder := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Render("Type to filter loaded terms")
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

func (m Model) renderTreePane(width int) string {
	title := "Tree"
	if node := m.currentNode(); node != nil {
		title += fmt.Sprintf("  current: %s", node.ID)
	}
	return m.renderPane(title, m.viewport.View(), true, width)
}

func (m Model) renderInspectorPane(width int) string {
	node := m.currentNode()
	if node == nil {
		body := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("No node selected.")
		return m.renderPane("Inspect", body, false, width)
	}

	title := lipgloss.NewStyle().Foreground(m.theme.Title).Bold(true)
	text := lipgloss.NewStyle().Foreground(m.theme.Text)
	muted := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	description := strings.TrimSpace(node.Description)
	if description == "" {
		description = "No description available for this term."
	}

	bodyLines := []string{
		title.Render(fmt.Sprintf("%s  %s", node.ID, node.Name)),
	}

	if m.height < 20 {
		bodyLines = append(bodyLines,
			text.Render(truncateText(description, width-4)),
			muted.Render(truncateText(nodeStatus(*node), width-4)),
		)
	} else {
		bodyLines = append(bodyLines,
			text.Render(truncateText(description, width-4)),
			muted.Render(truncateText(nodeStatus(*node), width-4)),
			muted.Render(truncateText("Path: "+strings.Join(pathLabels(pathToNode(m.roots, node.ID)), " / "), width-4)),
		)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, bodyLines...)
	return m.renderPane("Inspect", body, false, width)
}

func (m Model) renderFooter(width int) string {
	hint := "Arrows move • Right expands • Left collapses • Enter selects • Type filters • ? help"
	if m.activePane == paneSearch {
		hint = "Typing refines filter • Up/Down steps matches • Enter selects • Esc clears • Tab returns"
	}

	legend := m.renderLegend(width)
	hintLine := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted).
		Render(truncateText(hint, width))

	return lipgloss.JoinVertical(lipgloss.Left, hintLine, legend)
}

func (m Model) renderHelp(width int) string {
	text := lipgloss.NewStyle().Foreground(m.theme.Text)
	body := lipgloss.JoinVertical(
		lipgloss.Left,
		text.Render("Up/Down: move through the visible tree"),
		text.Render("Right: expand the current branch or ask the host to load its children"),
		text.Render("Left: collapse the current branch or move to its parent"),
		text.Render("Enter: confirm the currently highlighted term"),
		text.Render("Type or /: start filtering loaded terms immediately"),
		text.Render("Esc: clear the filter, return to browse, or cancel the browser"),
		text.Render("Tab: switch between browse and filter focus"),
		text.Render("Space: toggle the current branch open or closed"),
		"",
		text.Render("Legend"),
		m.renderLegend(width-4),
	)
	return m.renderPane("Help", body, false, width)
}

func (m Model) renderLegend(width int) string {
	selected := lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true).Render("› selected")
	closed := lipgloss.NewStyle().Foreground(m.theme.Collapsed).Render("▸ branch")
	open := lipgloss.NewStyle().Foreground(m.theme.Expanded).Render("▾ open")
	leaf := lipgloss.NewStyle().Foreground(m.theme.Leaf).Render("• leaf")
	loading := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("… loading")
	match := lipgloss.NewStyle().Foreground(m.theme.SearchHighlight).Bold(true).Render("match")
	context := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("dim = context")

	legend := strings.Join([]string{
		selected,
		closed,
		open,
		leaf,
		loading,
		match,
		context,
	}, "  ")

	return truncateANSI(legend, width)
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

func (m Model) renderTreeLine(node OntologyNode, selected bool) string {
	indent := strings.Repeat("  ", node.Depth)
	token := searchHighlightToken(m.searchQuery)
	matched := nodeMatchesQuery(node, m.searchQuery)

	marker := "  "
	if selected {
		marker = lipgloss.NewStyle().Foreground(m.theme.Selected).Bold(true).Render("› ")
	}

	idStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	nameColor := m.theme.Branch
	if node.Loaded && len(node.Children) == 0 {
		nameColor = m.theme.Leaf
	}
	if strings.TrimSpace(m.searchQuery) != "" && !matched && !selected {
		nameColor = m.theme.TextMuted
	}
	nameStyle := lipgloss.NewStyle().Foreground(nameColor)

	icon, iconColor := m.nodeGlyph(node)
	if strings.TrimSpace(m.searchQuery) != "" && matched {
		iconColor = m.theme.SearchHighlight
	}
	iconStyle := lipgloss.NewStyle().Foreground(iconColor)

	searchStyle := lipgloss.NewStyle().Foreground(m.theme.SearchHighlight).Bold(true)
	statusStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	idRendered := highlightMatch(node.ID, token, idStyle, searchStyle)
	nameRendered := highlightMatch(node.Name, token, nameStyle, searchStyle)

	line := marker +
		indent +
		iconStyle.Render(icon) +
		" " +
		idRendered
	if node.Name != "" {
		line += " " + nameRendered
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
	m.refreshSearchResults()
	m.syncSelectionToSearch()
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

func shouldStartSearch(msg tea.KeyPressMsg) bool {
	if strings.TrimSpace(msg.Text) == "" {
		return false
	}
	for _, r := range msg.Text {
		if unicode.IsGraphic(r) {
			return true
		}
	}
	return false
}

func nodeMatchesQuery(node OntologyNode, query string) bool {
	_, ok := scoreSearch(node, query)
	return ok
}

func nodeStatus(node OntologyNode) string {
	switch {
	case !node.Loaded:
		return "Children are not loaded yet. Press Right to reveal more structure."
	case len(node.Children) == 0:
		return "Leaf term. Press Enter to choose it."
	default:
		return fmt.Sprintf("Branch with %d loaded children. Press Right to open it or Left to collapse upward.", len(node.Children))
	}
}

func truncateText(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width <= 1 {
		return string(runes[:width])
	}
	return string(runes[:width-1]) + "…"
}

func truncateANSI(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}

	var out strings.Builder
	current := 0
	inANSI := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inANSI = true
			out.WriteRune(r)
		case inANSI:
			out.WriteRune(r)
			if r == 'm' {
				inANSI = false
			}
		default:
			if current >= width-1 {
				out.WriteRune('…')
				return out.String()
			}
			out.WriteRune(r)
			current++
		}
	}
	return out.String()
}
