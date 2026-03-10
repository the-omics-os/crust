package sequenceviewer

import (
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
	sequence string
	seqType  SequenceType
	residues []Residue
	view     ViewMode

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
		view:       IdentityView,
		width:      defaultWidth,
		height:     defaultHeight,
		theme:      DefaultTheme(),
		showHeader: true,
		gcWindow:   defaultGCWindow,
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
		case "up":
			m.viewport.ScrollUp(1)
			return m, nil
		case "down":
			m.viewport.ScrollDown(1)
			return m, nil
		case "pgup":
			m.viewport.PageUp()
			return m, nil
		case "pgdown":
			m.viewport.PageDown()
			return m, nil
		case "home":
			m.viewport.GotoTop()
			return m, nil
		case "end":
			m.viewport.GotoBottom()
			return m, nil
		case "tab":
			m.view = nextView(m.view, m.seqType)
			m.syncViewport()
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
			}
			return m, nil
		}
	}

	vp, cmd := m.viewport.Update(msg)
	m.viewport = vp
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
	m.applyState()
}

// SetResidues updates the viewer with a precomputed residue list.
func (m *Model) SetResidues(residues []Residue) {
	m.residues = copyResidues(residues)
	m.sequence = ""
	m.seqType = SequenceUnknown
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
		reserved += 2
	}
	if m.showHelp {
		reserved += 4
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
