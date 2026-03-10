package sequenceviewer

import (
	"sort"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
)

const (
	defaultWidth    = 80
	defaultHeight   = 14
	defaultGCWindow = 20
	minWidth        = 24
	minHeight       = 8
)

// Model is the Bubble Tea model for the sequence viewer.
type Model struct {
	sequence        string
	seqType         SequenceType
	residues        []Residue
	view            ViewMode
	focusIndex      int
	selectionAnchor int

	showComplement  bool
	annotations     []Annotation
	residuesPerLine int
	width           int
	height          int
	theme           Theme
	showHeader      bool
	showHelp        bool
	gcWindow        int

	overallGC        float64
	meltingTemp      float64
	isoelectricPoint float64
	totalWeight      float64
	orfs             []ORF
	restrictionSites []RestrictionSite

	viewport viewport.Model
	lineMeta []lineMeta
}

// New creates a SequenceViewer with the given options.
func New(opts ...Option) Model {
	m := Model{
		view:            IdentityView,
		width:           defaultWidth,
		height:          defaultHeight,
		theme:           DefaultTheme(),
		showHeader:      true,
		gcWindow:        defaultGCWindow,
		selectionAnchor: -1,
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.applyState()
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
		m.syncViewport()
		return m, nil
	case tea.KeyPressMsg:
		switch msg.String() {
		case "left", "h":
			m.moveFocus(-1)
			return m, nil
		case "shift+left":
			m.moveFocusExtended(-1)
			return m, nil
		case "right", "l":
			m.moveFocus(1)
			return m, nil
		case "shift+right":
			m.moveFocusExtended(1)
			return m, nil
		case "up", "k":
			m.moveFocus(-m.effectiveResiduesPerLine())
			return m, nil
		case "shift+up":
			m.moveFocusExtended(-m.effectiveResiduesPerLine())
			return m, nil
		case "down", "j":
			m.moveFocus(m.effectiveResiduesPerLine())
			return m, nil
		case "shift+down":
			m.moveFocusExtended(m.effectiveResiduesPerLine())
			return m, nil
		case "pgup":
			m.moveFocus(-m.pageStep())
			return m, nil
		case "shift+pgup":
			m.moveFocusExtended(-m.pageStep())
			return m, nil
		case "pgdown":
			m.moveFocus(m.pageStep())
			return m, nil
		case "shift+pgdown":
			m.moveFocusExtended(m.pageStep())
			return m, nil
		case "home":
			m.moveFocusTo(0, false)
			return m, nil
		case "shift+home":
			m.moveFocusTo(0, true)
			return m, nil
		case "end":
			m.moveFocusTo(len(m.residues)-1, false)
			return m, nil
		case "shift+end":
			m.moveFocusTo(len(m.residues)-1, true)
			return m, nil
		case "tab":
			m.view = nextView(m.view, m.seqType)
			m.syncViewport()
			return m, nil
		case "]", "n":
			m.jumpFocusRegion(1)
			return m, nil
		case "[", "N":
			m.jumpFocusRegion(-1)
			return m, nil
		case "c", "C":
			if m.seqType == DNA {
				m.showComplement = !m.showComplement
				m.syncViewport()
			}
			return m, nil
		case "?":
			m.showHelp = !m.showHelp
			m.syncViewport()
			return m, nil
		case "esc":
			if m.showHelp {
				m.showHelp = false
				m.syncViewport()
			} else if m.hasSelection() {
				m.clearSelection()
				m.syncViewport()
			}
			return m, nil
		}
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp
	m.syncFocusFromViewport()
	return m, cmd
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the rendered viewer string.
func (m Model) Render() string {
	return m.render()
}

// SetSequence updates the sequence and type.
func (m *Model) SetSequence(seq string, t SequenceType) {
	m.sequence = NormalizeSequence(seq)
	m.seqType = t
	m.residues = nil
	m.selectionAnchor = -1
	m.applyState()
}

// SetResidues updates the viewer with a precomputed residue list.
func (m *Model) SetResidues(residues []Residue) {
	m.residues = copyResidues(residues)
	m.sequence = ""
	m.seqType = SequenceUnknown
	m.selectionAnchor = -1
	m.applyState()
}

// SetView updates the current view mode.
func (m *Model) SetView(v ViewMode) {
	m.view = ensureApplicableView(v, m.seqType)
	m.syncViewport()
}

// SetWidth updates the render width.
func (m *Model) SetWidth(w int) {
	m.width = w
	m.syncViewport()
}

// Sequence returns the normalized sequence string.
func (m Model) Sequence() string { return m.sequence }

// Type returns the biological sequence type.
func (m Model) Type() SequenceType { return m.seqType }

// Length returns the number of residues.
func (m Model) Length() int { return len(m.residues) }

// ViewMode returns the active view.
func (m Model) ViewMode() ViewMode { return m.view }

// Residues returns a defensive copy of the residues.
func (m Model) Residues() []Residue { return copyResidues(m.residues) }

// ORFs returns the cached ORF analysis results.
func (m Model) ORFs() []ORF { return copyORFs(m.orfs) }

// RestrictionSites returns the cached restriction site matches.
func (m Model) RestrictionSites() []RestrictionSite {
	return copyRestrictionSites(m.restrictionSites)
}

// GCContent returns the overall GC fraction in the current sequence.
func (m Model) GCContent() float64 { return m.overallGC }

// MeltingTemp returns the estimated melting temperature in Celsius.
func (m Model) MeltingTemp() float64 { return m.meltingTemp }

// IsoelectricPoint returns the estimated pI for protein sequences.
func (m Model) IsoelectricPoint() float64 { return m.isoelectricPoint }

// FocusPosition returns the 1-based focused residue position.
func (m Model) FocusPosition() int {
	if residue, ok := m.focusedResidue(); ok {
		return residue.Position
	}
	return 0
}

// SelectionRange returns the inclusive 1-based selected residue range.
func (m Model) SelectionRange() (int, int, bool) {
	return m.selectionPositionRange()
}

// SetFocus moves focus to the given 1-based residue position.
func (m *Model) SetFocus(position int) {
	m.clearSelection()
	m.setFocusIndex(position - 1)
	m.syncViewport()
}

func (m *Model) applyState() {
	if m.width < minWidth {
		m.width = minWidth
	}
	if m.height < minHeight {
		m.height = minHeight
	}
	if m.gcWindow <= 0 {
		m.gcWindow = defaultGCWindow
	}

	switch {
	case len(m.residues) > 0:
		if m.seqType == SequenceUnknown {
			m.seqType = inferSequenceTypeFromResidues(m.residues)
		}
		m.residues = enrichResidues(m.residues, m.seqType, m.gcWindow)
		m.sequence = sequenceFromResidues(m.residues)
	case m.sequence != "":
		m.sequence = NormalizeSequence(m.sequence)
		if m.seqType == SequenceUnknown {
			m.seqType = inferSequenceTypeFromSequence(m.sequence)
		}
		m.residues = buildResiduesFromSequence(m.sequence, m.seqType, m.gcWindow)
	default:
		m.seqType = SequenceUnknown
		m.residues = nil
	}

	m.view = ensureApplicableView(m.view, m.seqType)
	if m.seqType != DNA {
		m.showComplement = false
	}
	m.selectionAnchor = -1
	m.setFocusIndex(m.focusIndex)

	m.recomputeAnalyses()
	m.syncViewport()
}

func (m *Model) recomputeAnalyses() {
	m.overallGC = overallGCContent(m.residues)
	m.meltingTemp = EstimateTm(m.residues)
	m.isoelectricPoint = EstimatePI(m.residues)
	m.totalWeight = totalMolecularWeight(m.residues)

	if m.seqType == DNA {
		m.orfs = FindORFs(m.residues, 1)
		m.restrictionSites = FindRestrictionSites(m.residues, nil)
	} else {
		m.orfs = nil
		m.restrictionSites = nil
	}
}

func (m *Model) ensureViewport() {
	contentHeight := m.contentHeight()
	if m.viewport.Width() == 0 && m.viewport.Height() == 0 {
		m.viewport = viewport.New(
			viewport.WithWidth(m.width),
			viewport.WithHeight(contentHeight),
		)
		m.viewport.SoftWrap = false
		m.viewport.FillHeight = false
		m.viewport.MouseWheelEnabled = true
		m.viewport.MouseWheelDelta = 1
		return
	}
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(contentHeight)
}

func (m Model) contentHeight() int {
	height := m.height - m.reservedHeight()
	if height < 3 {
		return 3
	}
	return height
}

func (m Model) reservedHeight() int {
	reserved := 2 // separator + footer
	if m.showHeader {
		reserved += m.headerLineCount() + 1
	}
	if m.showHelp {
		reserved += m.helpLineCount()
	}
	return reserved
}

func (m Model) render() string {
	var parts []string
	if m.showHeader {
		parts = append(parts, m.renderHeader(), m.renderSeparator())
	}
	parts = append(parts, m.viewport.View(), m.renderSeparator(), m.renderFooter())
	if m.showHelp {
		parts = append(parts, m.renderHelp())
	}
	return strings.Join(parts, "\n")
}

func (m *Model) moveFocus(delta int) {
	if len(m.residues) == 0 || delta == 0 {
		return
	}
	m.moveFocusTo(m.focusIndex+delta, false)
}

func (m *Model) moveFocusExtended(delta int) {
	if len(m.residues) == 0 || delta == 0 {
		return
	}
	m.moveFocusTo(m.focusIndex+delta, true)
}

func (m *Model) moveFocusTo(index int, extend bool) {
	if len(m.residues) == 0 {
		return
	}
	if extend {
		m.ensureSelectionAnchor()
	} else {
		m.clearSelection()
	}
	m.setFocusIndex(index)
	if m.selectionAnchor == m.focusIndex {
		m.clearSelection()
	}
	m.syncViewport()
}

func (m *Model) setFocusIndex(index int) {
	if len(m.residues) == 0 {
		m.focusIndex = 0
		m.selectionAnchor = -1
		return
	}
	if index < 0 {
		index = 0
	}
	if index >= len(m.residues) {
		index = len(m.residues) - 1
	}
	m.focusIndex = index
}

func (m *Model) ensureSelectionAnchor() {
	if m.selectionAnchor >= 0 {
		return
	}
	m.selectionAnchor = m.focusIndex
}

func (m *Model) clearSelection() {
	m.selectionAnchor = -1
}

func (m Model) hasSelection() bool {
	_, _, ok := m.selectionIndexRange()
	return ok
}

func (m Model) selectionIndexRange() (int, int, bool) {
	if len(m.residues) == 0 || m.selectionAnchor < 0 || m.selectionAnchor == m.focusIndex {
		return 0, 0, false
	}
	start, end := m.selectionAnchor, m.focusIndex
	if start > end {
		start, end = end, start
	}
	return start, end, true
}

func (m Model) selectionPositionRange() (int, int, bool) {
	start, end, ok := m.selectionIndexRange()
	if !ok {
		return 0, 0, false
	}
	return m.residues[start].Position, m.residues[end].Position, true
}

func (m Model) selectionCount() int {
	start, end, ok := m.selectionIndexRange()
	if !ok {
		return 0
	}
	return end - start + 1
}

func (m Model) focusedResidue() (Residue, bool) {
	if len(m.residues) == 0 {
		return Residue{}, false
	}
	index := m.focusIndex
	if index < 0 {
		index = 0
	}
	if index >= len(m.residues) {
		index = len(m.residues) - 1
	}
	return m.residues[index], true
}

func (m Model) pageStep() int {
	rowsPerPage := m.contentHeight() / m.rowHeightEstimate()
	if rowsPerPage < 1 {
		rowsPerPage = 1
	}
	return rowsPerPage * m.effectiveResiduesPerLine()
}

func (m Model) rowHeightEstimate() int {
	height := 1 // sequence
	if m.showComplement && m.seqType == DNA {
		height++
	}
	if m.view != IdentityView {
		height++
	}
	height++ // annotation track slot
	return height
}

func (m *Model) jumpFocusRegion(direction int) {
	targets := m.focusJumpTargets()
	if len(targets) == 0 {
		return
	}
	current := m.FocusPosition()
	if current == 0 {
		current = 1
	}

	if direction >= 0 {
		for _, target := range targets {
			if target > current {
				m.SetFocus(target)
				return
			}
		}
		m.SetFocus(targets[0])
		return
	}

	for i := len(targets) - 1; i >= 0; i-- {
		if targets[i] < current {
			m.SetFocus(targets[i])
			return
		}
	}
	m.SetFocus(targets[len(targets)-1])
}

func (m Model) focusJumpTargets() []int {
	targets := make([]int, 0, len(m.annotations)+len(m.orfs))
	seen := map[int]struct{}{}
	if len(m.annotations) > 0 {
		for _, annotation := range m.annotations {
			if annotation.Start <= 0 {
				continue
			}
			if _, ok := seen[annotation.Start]; ok {
				continue
			}
			targets = append(targets, annotation.Start)
			seen[annotation.Start] = struct{}{}
		}
	} else if m.seqType == DNA {
		for _, orf := range m.orfs {
			if orf.Start <= 0 {
				continue
			}
			if _, ok := seen[orf.Start]; ok {
				continue
			}
			targets = append(targets, orf.Start)
			seen[orf.Start] = struct{}{}
		}
	}
	sort.Ints(targets)
	return targets
}

func (m Model) headerLineCount() int {
	if !m.showHeader {
		return 0
	}
	return 3
}

func (m Model) helpLineCount() int {
	if !m.showHelp {
		return 0
	}
	return len(m.helpLines())
}

func (m *Model) syncFocusFromViewport() {
	if len(m.lineMeta) == 0 {
		return
	}
	startLine := m.viewport.YOffset()
	endLine := startLine + 1
	if endLine > len(m.lineMeta) {
		endLine = len(m.lineMeta)
	}
	for _, meta := range m.lineMeta[startLine:endLine] {
		if meta.Kind != lineKindSequence || meta.Start == 0 {
			continue
		}
		m.SetFocus(meta.Start)
		return
	}
}
