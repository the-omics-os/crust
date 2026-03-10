package platemap

import (
	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

const plateMapComponent = "plate_map"

// ViewMode is the active rendering lens for the plate.
type ViewMode int

const (
	ViewRawSignal ViewMode = iota
	ViewNormalized
	ViewZScore
	ViewHitClass
	ViewControlLayout
	ViewMissingness
)

var allViewModes = [...]ViewMode{
	ViewRawSignal,
	ViewNormalized,
	ViewZScore,
	ViewHitClass,
	ViewControlLayout,
	ViewMissingness,
}

// Model is the Bubble Tea model for PlateMap.
type Model struct {
	plate          PlateData
	mode           ViewMode
	theme          Theme
	width          int
	height         int
	cursorRow      int
	cursorCol      int
	rowOffset      int
	colOffset      int
	selectedRow    int
	selectedCol    int
	detailExpanded bool
	helpVisible    bool
}

// New creates a PlateMap with the given options.
func New(opts ...Option) Model {
	m := Model{
		plate: PlateData{
			Format: Plate96,
			Title:  "Plate Map",
		},
		mode:        ViewRawSignal,
		theme:       DefaultTheme(),
		width:       80,
		height:      18,
		selectedRow: -1,
		selectedCol: -1,
	}
	for _, opt := range opts {
		opt(&m)
	}
	m.normalize()
	return m
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		if typed.Width > 0 {
			m.width = typed.Width
		}
		if typed.Height > 0 {
			m.height = typed.Height
		}
		m.normalize()
		return m, nil
	case tea.KeyPressMsg:
		switch typed.String() {
		case "up":
			m.moveCursor(-1, 0)
		case "down":
			m.moveCursor(1, 0)
		case "left":
			m.moveCursor(0, -1)
		case "right":
			m.moveCursor(0, 1)
		case "shift+up":
			m.moveCursor(-1, 0)
			m.selectedRow = m.cursorRow
			m.selectedCol = -1
		case "shift+down":
			m.moveCursor(1, 0)
			m.selectedRow = m.cursorRow
			m.selectedCol = -1
		case "shift+left":
			m.moveCursor(0, -1)
			m.selectedCol = m.cursorCol
			m.selectedRow = -1
		case "shift+right":
			m.moveCursor(0, 1)
			m.selectedCol = m.cursorCol
			m.selectedRow = -1
		case "tab":
			m.mode = m.mode.next()
		case "shift+tab":
			m.mode = m.mode.prev()
		case "enter":
			m.detailExpanded = true
			m.normalize()
			return m, submitCmd(m.submitData())
		case "esc":
			switch {
			case m.helpVisible:
				m.helpVisible = false
			case m.detailExpanded:
				m.detailExpanded = false
			case m.selectedRow >= 0 || m.selectedCol >= 0:
				m.selectedRow = -1
				m.selectedCol = -1
			default:
				return m, cancelCmd()
			}
		case "?":
			m.helpVisible = !m.helpVisible
		default:
			return m, nil
		}
		m.normalize()
	}
	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the plate as a plain string.
func (m Model) Render() string {
	return m.render()
}

// SetPlate replaces the full plate payload.
func (m *Model) SetPlate(plate PlateData) {
	m.plate = plate.Copy()
	m.normalize()
}

// SetWells replaces all wells while preserving the rest of the payload.
func (m *Model) SetWells(wells []Well) {
	m.plate.Wells = append([]Well(nil), wells...)
	m.normalize()
}

// UpsertWell adds or replaces a single well by coordinate.
func (m *Model) UpsertWell(well Well) {
	requiredFormat, ok := requiredFormatForCoordinate(well.Row, well.Col)
	if !ok {
		return
	}
	if !m.plate.Format.Valid() || requiredFormat.WellCount() > m.plate.Format.WellCount() {
		m.plate.Format = requiredFormat
	}

	for idx := range m.plate.Wells {
		if m.plate.Wells[idx].Row == well.Row && m.plate.Wells[idx].Col == well.Col {
			m.plate.Wells[idx] = well
			m.normalize()
			return
		}
	}

	m.plate.Wells = append(m.plate.Wells, well)
	m.normalize()
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(width int) {
	m.width = width
	m.normalize()
}

// SetHeight updates the rendering height budget.
func (m *Model) SetHeight(height int) {
	m.height = height
	m.normalize()
}

// SetViewMode updates the active view mode.
func (m *Model) SetViewMode(mode ViewMode) {
	m.mode = mode
	m.normalize()
}

// SetCursor updates the focused coordinate.
func (m *Model) SetCursor(row, col int) {
	m.cursorRow = row
	m.cursorCol = col
	m.normalize()
}

// Plate returns a deep copy of the current plate payload.
func (m Model) Plate() PlateData {
	return m.plate.Copy()
}

// Width returns the configured rendering width.
func (m Model) Width() int { return m.width }

// Height returns the configured rendering height.
func (m Model) Height() int { return m.height }

// Mode returns the active rendering mode.
func (m Model) Mode() ViewMode { return m.mode }

// Cursor returns the focused row and column.
func (m Model) Cursor() (int, int) { return m.cursorRow, m.cursorCol }

// Coordinate returns the focused well coordinate.
func (m Model) Coordinate() string { return coordinate(m.cursorRow, m.cursorCol) }

func (m *Model) normalize() {
	m.plate = normalizePlateData(m.plate)
	m.plate.Format = requiredFormatForWells(m.plate.Wells, m.plate.Format)
	m.mode = normalizeViewMode(m.mode)

	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()

	m.cursorRow = clampInt(m.cursorRow, 0, rows-1)
	m.cursorCol = clampInt(m.cursorCol, 0, cols-1)

	if m.selectedRow >= rows || m.selectedRow < -1 {
		m.selectedRow = -1
	}
	if m.selectedCol >= cols || m.selectedCol < -1 {
		m.selectedCol = -1
	}

	m.clampOffsets()
	m.ensureVisible()
}

func (m *Model) moveCursor(deltaRow, deltaCol int) {
	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()

	m.cursorRow = clampInt(m.cursorRow+deltaRow, 0, rows-1)
	m.cursorCol = clampInt(m.cursorCol+deltaCol, 0, cols-1)
	m.ensureVisible()
}

func (m *Model) clampOffsets() {
	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()
	visibleRows, visibleCols := m.gridViewportSize()

	m.rowOffset = clampInt(m.rowOffset, 0, maxInt(0, rows-visibleRows))
	m.colOffset = clampInt(m.colOffset, 0, maxInt(0, cols-visibleCols))
}

func (m *Model) ensureVisible() {
	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()
	visibleRows, visibleCols := m.gridViewportSize()

	if m.cursorRow < m.rowOffset {
		m.rowOffset = m.cursorRow
	}
	if m.cursorRow >= m.rowOffset+visibleRows {
		m.rowOffset = m.cursorRow - visibleRows + 1
	}
	if m.cursorCol < m.colOffset {
		m.colOffset = m.cursorCol
	}
	if m.cursorCol >= m.colOffset+visibleCols {
		m.colOffset = m.cursorCol - visibleCols + 1
	}

	m.rowOffset = clampInt(m.rowOffset, 0, maxInt(0, rows-visibleRows))
	m.colOffset = clampInt(m.colOffset, 0, maxInt(0, cols-visibleCols))
}

func (m Model) submitData() map[string]any {
	row, col := m.cursorRow, m.cursorCol
	payload := map[string]any{
		"coordinate": coordinate(row, col),
		"row":        row,
		"col":        col,
		"view":       m.mode.String(),
		"format":     int(m.plate.Format),
		"title":      m.plate.Title,
	}

	if m.selectedRow >= 0 {
		payload["selected_row"] = rowLabel(m.selectedRow)
	}
	if m.selectedCol >= 0 {
		payload["selected_col"] = m.selectedCol + 1
	}

	if well, ok := m.wellAt(row, col); ok {
		payload["present"] = true
		payload["well"] = wellToMap(well)
	} else {
		payload["present"] = false
	}

	return payload
}

func (m Model) wellAt(row, col int) (Well, bool) {
	for _, well := range m.plate.Wells {
		if well.Row == row && well.Col == col {
			return well, true
		}
	}
	return Well{}, false
}

func (v ViewMode) String() string {
	switch v {
	case ViewRawSignal:
		return "Raw Signal"
	case ViewNormalized:
		return "Normalized"
	case ViewZScore:
		return "Z-Score"
	case ViewHitClass:
		return "Hit Class"
	case ViewControlLayout:
		return "Control Layout"
	case ViewMissingness:
		return "Missingness"
	default:
		return ViewRawSignal.String()
	}
}

func (v ViewMode) next() ViewMode {
	v = normalizeViewMode(v)
	return allViewModes[(int(v)+1)%len(allViewModes)]
}

func (v ViewMode) prev() ViewMode {
	v = normalizeViewMode(v)
	return allViewModes[(int(v)+len(allViewModes)-1)%len(allViewModes)]
}

func normalizeViewMode(mode ViewMode) ViewMode {
	if int(mode) < 0 || int(mode) >= len(allViewModes) {
		return ViewRawSignal
	}
	return mode
}

func requiredFormatForWells(wells []Well, current PlateFormat) PlateFormat {
	if !current.Valid() {
		current = inferPlateFormat(len(wells))
	}
	for _, well := range wells {
		required, ok := requiredFormatForCoordinate(well.Row, well.Col)
		if ok && required.WellCount() > current.WellCount() {
			current = required
		}
	}
	return current
}

func submitCmd(data map[string]any) tea.Cmd {
	return func() tea.Msg {
		return crust.SubmitMsg{
			Component: plateMapComponent,
			Data:      data,
		}
	}
}

func cancelCmd() tea.Cmd {
	return func() tea.Msg {
		return crust.CancelMsg{
			Component: plateMapComponent,
			Reason:    "user cancelled",
		}
	}
}

func wellToMap(well Well) map[string]any {
	return map[string]any{
		"row":        well.Row,
		"col":        well.Col,
		"signal":     well.Signal,
		"normalized": well.Normalized,
		"z_score":    well.ZScore,
		"control":    normalizeControl(well.Control),
		"sample_id":  well.SampleID,
		"reagent":    well.Reagent,
		"hit":        well.Hit,
		"missing":    well.Missing,
	}
}

func clampInt(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
