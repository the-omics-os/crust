package platemap

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	minRenderWidth  = 32
	minRenderHeight = 9
	cellWidth       = 3
)

type plateStats struct {
	hits      int
	samples   int
	present   int
	missing   int
	rawMin    float64
	rawMax    float64
	normMin   float64
	normMax   float64
	maxAbsZ   float64
	hasRaw    bool
	hasNorm   bool
	hasZScore bool
}

func (m Model) render() string {
	m.normalize()

	parts := []string{
		m.renderHeader(),
		m.renderGrid(),
		m.renderSeparator(),
		m.renderFooter(),
	}
	if m.helpVisible {
		parts = append(parts, m.renderHelp())
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderHeader() string {
	stats := m.computeStats()
	title := m.headerTitle()
	denominator := stats.samples
	if denominator == 0 {
		denominator = stats.present
	}

	lineOneParts := []string{
		title,
		"View: " + m.mode.String(),
		fmt.Sprintf("Hits: %d/%d", stats.hits, denominator),
		"Cursor: " + coordinate(m.cursorRow, m.cursorCol),
	}
	if m.selectedRow >= 0 {
		lineOneParts = append(lineOneParts, "Row: "+rowLabel(m.selectedRow))
	}
	if m.selectedCol >= 0 {
		lineOneParts = append(lineOneParts, fmt.Sprintf("Col: %d", m.selectedCol+1))
	}

	visibleRows, visibleCols := m.gridViewportSize()
	rowStart := m.rowOffset
	rowEnd := minInt(m.plate.Format.Rows(), rowStart+visibleRows)
	colStart := m.colOffset
	colEnd := minInt(m.plate.Format.Cols(), colStart+visibleCols)

	lineTwo := fmt.Sprintf(
		"Window rows %s-%s/%d | cols %d-%d/%d | Present: %d | Missing: %d",
		rowLabel(rowStart),
		rowLabel(rowEnd-1),
		m.plate.Format.Rows(),
		colStart+1,
		colEnd,
		m.plate.Format.Cols(),
		stats.present,
		stats.missing,
	)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(strings.Join(lineOneParts, " | ")),
		mutedStyle.Render(lineTwo),
	)
}

func (m Model) headerTitle() string {
	formatLabel := m.plate.Format.String()
	title := strings.TrimSpace(m.plate.Title)
	switch {
	case title == "":
		return formatLabel
	case title == formatLabel:
		return title
	default:
		return title + " | " + formatLabel
	}
}

func (m Model) renderGrid() string {
	stats := m.computeStats()
	wells := m.indexedWells()
	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()
	visibleRows, visibleCols := m.gridViewportSize()
	rowStart := m.rowOffset
	rowEnd := minInt(rows, rowStart+visibleRows)
	colStart := m.colOffset
	colEnd := minInt(cols, colStart+visibleCols)
	rowLabelWidth := maxInt(1, len(rowLabel(rows-1)))

	headerCellStyle := lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center).Foreground(m.theme.Header)
	var headerCells []string
	for col := colStart; col < colEnd; col++ {
		style := headerCellStyle
		if col == m.selectedCol {
			style = style.Background(m.theme.SelectedCol)
		}
		headerCells = append(headerCells, style.Render(fmt.Sprintf("%d", col+1)))
	}

	lines := []string{
		strings.Repeat(" ", rowLabelWidth) + " " + strings.Join(headerCells, " "),
	}

	rowLabelStyle := lipgloss.NewStyle().Width(rowLabelWidth).Align(lipgloss.Right).Foreground(m.theme.Header)
	for row := rowStart; row < rowEnd; row++ {
		style := rowLabelStyle
		if row == m.selectedRow {
			style = style.Background(m.theme.SelectedRow)
		}

		var cells []string
		for col := colStart; col < colEnd; col++ {
			well, ok := wells[gridKey(row, col)]
			cells = append(cells, m.renderCell(row, col, well, ok, stats))
		}

		lines = append(lines, style.Render(rowLabel(row))+" "+strings.Join(cells, " "))
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderCell(row, col int, well Well, ok bool, stats plateStats) string {
	glyph, fg := m.cellGlyphAndColor(well, ok, stats)
	style := lipgloss.NewStyle().Width(cellWidth).Align(lipgloss.Center).Foreground(fg)

	if row == m.selectedRow {
		style = style.Background(m.theme.SelectedRow)
	}
	if col == m.selectedCol {
		style = style.Background(m.theme.SelectedCol)
	}
	if row == m.cursorRow && col == m.cursorCol {
		style = style.Background(m.theme.CursorBg).Bold(true)
	}

	return style.Render(glyph)
}

func (m Model) cellGlyphAndColor(well Well, ok bool, stats plateStats) (string, color.Color) {
	if !ok {
		return " ", m.theme.Empty
	}
	if well.Missing {
		return "x", m.theme.Missing
	}

	control := normalizeControl(well.Control)
	switch m.mode {
	case ViewControlLayout:
		return controlGlyphAndColor(control, m.theme)
	case ViewHitClass:
		if glyph, fg, handled := overlayGlyphAndColor(well, control, m.theme); handled {
			return glyph, fg
		}
		return ".", m.theme.Sample
	case ViewMissingness:
		if glyph, fg, handled := controlOnlyGlyphAndColor(control, m.theme); handled {
			return glyph, fg
		}
		return ".", m.theme.Sample
	case ViewRawSignal:
		if glyph, fg, handled := overlayGlyphAndColor(well, control, m.theme); handled {
			return glyph, fg
		}
		ratio := normalizedRange(well.Signal, stats.rawMin, stats.rawMax)
		return intensityGlyph(ratio), sampleBucketColor(ratio, m.theme)
	case ViewNormalized:
		if glyph, fg, handled := overlayGlyphAndColor(well, control, m.theme); handled {
			return glyph, fg
		}
		ratio := normalizedRange(well.Normalized, stats.normMin, stats.normMax)
		return intensityGlyph(ratio), sampleBucketColor(ratio, m.theme)
	case ViewZScore:
		if glyph, fg, handled := overlayGlyphAndColor(well, control, m.theme); handled {
			return glyph, fg
		}
		ratio := normalizedAbs(well.ZScore, stats.maxAbsZ)
		return intensityGlyph(ratio), zScoreColor(well.ZScore, m.theme)
	default:
		return ".", m.theme.Sample
	}
}

func (m Model) renderSeparator() string {
	style := lipgloss.NewStyle().Foreground(m.theme.Border)
	return style.Render(strings.Repeat("-", m.renderWidth()))
}

func (m Model) renderFooter() string {
	well, ok := m.wellAt(m.cursorRow, m.cursorCol)
	coord := coordinate(m.cursorRow, m.cursorCol)
	textStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	lines := []string{
		textStyle.Render(fmt.Sprintf("[%s] %s", coord, m.focusSummary(well, ok))),
	}
	if !m.detailExpanded {
		return strings.Join(lines, "\n")
	}

	if !ok {
		lines = append(lines,
			mutedStyle.Render("Signal: n/a | Normalized: n/a | Z: n/a | Control: n/a"),
			mutedStyle.Render("Replicates: none available for an unloaded well"),
			mutedStyle.Render(m.metadataSummary()),
		)
		return strings.Join(lines, "\n")
	}

	lines = append(lines,
		textStyle.Render(fmt.Sprintf(
			"Signal: %.2f | Normalized: %.2f | Z: %.2f | Control: %s | Missing: %t",
			well.Signal,
			well.Normalized,
			well.ZScore,
			normalizeControl(well.Control),
			well.Missing,
		)),
	)

	if summary, ok := m.replicateSummary(well); ok {
		lines = append(lines, textStyle.Render(summary))
	} else {
		lines = append(lines, mutedStyle.Render("Replicates: no shared reagent or sample ID found"))
	}

	lines = append(lines, mutedStyle.Render(m.metadataSummary()))
	return strings.Join(lines, "\n")
}

func (m Model) focusSummary(well Well, ok bool) string {
	if !ok {
		return "No well data loaded"
	}

	parts := []string{wellKindLabel(well)}
	if sampleID := strings.TrimSpace(well.SampleID); sampleID != "" {
		parts = append(parts, "Sample: "+sampleID)
	}
	if reagent := strings.TrimSpace(well.Reagent); reagent != "" {
		parts = append(parts, "Reagent: "+reagent)
	}
	if well.Hit && wellKindLabel(well) != "Hit" {
		parts = append(parts, "HIT")
	}
	return strings.Join(parts, " | ")
}

func (m Model) replicateSummary(target Well) (string, bool) {
	key := strings.TrimSpace(target.Reagent)
	label := "reagent"
	if key == "" {
		key = strings.TrimSpace(target.SampleID)
		label = "sample_id"
	}
	if key == "" {
		return "", false
	}

	var count, observed, hitCount int
	var signalSum, zSum float64
	for _, well := range m.plate.Wells {
		match := false
		switch label {
		case "reagent":
			match = strings.TrimSpace(well.Reagent) == key
		default:
			match = strings.TrimSpace(well.SampleID) == key
		}
		if !match {
			continue
		}

		count++
		if well.Hit {
			hitCount++
		}
		if !well.Missing {
			observed++
			signalSum += well.Signal
			zSum += well.ZScore
		}
	}

	if count == 0 {
		return "", false
	}

	meanSignal := 0.0
	meanZ := 0.0
	if observed > 0 {
		meanSignal = signalSum / float64(observed)
		meanZ = zSum / float64(observed)
	}

	return fmt.Sprintf(
		"Replicates by %s (%s): n=%d | observed=%d | mean signal=%.2f | mean z=%.2f | hits=%d",
		label,
		key,
		count,
		observed,
		meanSignal,
		meanZ,
		hitCount,
	), true
}

func (m Model) metadataSummary() string {
	if len(m.plate.Metadata) == 0 {
		return "Metadata: none"
	}

	keys := make([]string, 0, len(m.plate.Metadata))
	for key := range m.plate.Metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", key, m.plate.Metadata[key]))
	}
	return "Metadata: " + strings.Join(pairs, ", ")
}

func (m Model) renderHelp() string {
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	return lipgloss.JoinVertical(
		lipgloss.Left,
		mutedStyle.Render("Keys: arrows move | shift+arrows select row or column | tab/shift+tab change view"),
		mutedStyle.Render("Enter inspects the focused well and emits SubmitMsg | esc clears active context or exits"),
		mutedStyle.Render("Glyphs: + positive ctrl | - negative ctrl | # hit | . sample | o empty | x missing"),
	)
}

func (m Model) gridViewportSize() (int, int) {
	rows := m.plate.Format.Rows()
	cols := m.plate.Format.Cols()
	rowLabelWidth := maxInt(1, len(rowLabel(rows-1)))

	visibleCols := (m.renderWidth() - rowLabelWidth) / (cellWidth + 1)
	if visibleCols < 1 {
		visibleCols = 1
	}
	if visibleCols > cols {
		visibleCols = cols
	}

	overhead := 2 + 1 + 1 + m.footerLineCount() + m.helpLineCount()
	visibleRows := m.renderHeight() - overhead
	if visibleRows < 1 {
		visibleRows = 1
	}
	if visibleRows > rows {
		visibleRows = rows
	}

	return visibleRows, visibleCols
}

func (m Model) footerLineCount() int {
	if m.detailExpanded {
		return 4
	}
	return 1
}

func (m Model) helpLineCount() int {
	if m.helpVisible {
		return 3
	}
	return 0
}

func (m Model) renderWidth() int {
	if m.width < minRenderWidth {
		return minRenderWidth
	}
	return m.width
}

func (m Model) renderHeight() int {
	if m.height < minRenderHeight {
		return minRenderHeight
	}
	return m.height
}

func (m Model) indexedWells() map[int]Well {
	index := make(map[int]Well, len(m.plate.Wells))
	for _, well := range m.plate.Wells {
		index[gridKey(well.Row, well.Col)] = well
	}
	return index
}

func (m Model) computeStats() plateStats {
	stats := plateStats{
		rawMin:  math.Inf(1),
		rawMax:  math.Inf(-1),
		normMin: math.Inf(1),
		normMax: math.Inf(-1),
	}

	for _, well := range m.plate.Wells {
		if well.Missing {
			stats.missing++
			continue
		}

		stats.present++
		if normalizeControl(well.Control) == ControlSample {
			stats.samples++
		}
		if well.Hit {
			stats.hits++
		}

		stats.rawMin = minFloat(stats.rawMin, well.Signal)
		stats.rawMax = maxFloat(stats.rawMax, well.Signal)
		stats.normMin = minFloat(stats.normMin, well.Normalized)
		stats.normMax = maxFloat(stats.normMax, well.Normalized)
		stats.maxAbsZ = maxFloat(stats.maxAbsZ, math.Abs(well.ZScore))
		stats.hasRaw = true
		stats.hasNorm = true
		stats.hasZScore = true
	}

	if !stats.hasRaw {
		stats.rawMin, stats.rawMax = 0, 1
	}
	if !stats.hasNorm {
		stats.normMin, stats.normMax = 0, 1
	}
	if !stats.hasZScore {
		stats.maxAbsZ = 1
	}

	return stats
}

func overlayGlyphAndColor(well Well, control string, theme Theme) (string, color.Color, bool) {
	if glyph, fg, handled := controlOnlyGlyphAndColor(control, theme); handled {
		return glyph, fg, true
	}
	if well.Hit {
		return "#", theme.Hit, true
	}
	return "", nil, false
}

func controlOnlyGlyphAndColor(control string, theme Theme) (string, color.Color, bool) {
	switch control {
	case ControlPositive:
		return "+", theme.PositiveCtrl, true
	case ControlNegative:
		return "-", theme.NegativeCtrl, true
	case ControlEmpty:
		return "o", theme.Empty, true
	default:
		return "", nil, false
	}
}

func controlGlyphAndColor(control string, theme Theme) (string, color.Color) {
	if glyph, fg, handled := controlOnlyGlyphAndColor(control, theme); handled {
		return glyph, fg
	}
	return ".", theme.Sample
}

func sampleBucketColor(ratio float64, theme Theme) color.Color {
	switch {
	case ratio < 0.33:
		return theme.TextMuted
	case ratio < 0.66:
		return theme.Sample
	default:
		return theme.Header
	}
}

func zScoreColor(value float64, theme Theme) color.Color {
	switch {
	case math.Abs(value) < 0.5:
		return theme.TextMuted
	case value < 0:
		return theme.NegativeCtrl
	default:
		return theme.PositiveCtrl
	}
}

func intensityGlyph(ratio float64) string {
	levels := []string{".", ":", "*", "O", "@"}
	idx := int(math.Round(clampFloat(ratio, 0, 1) * float64(len(levels)-1)))
	return levels[idx]
}

func normalizedRange(value, minValue, maxValue float64) float64 {
	if maxValue <= minValue {
		return 0
	}
	return clampFloat((value-minValue)/(maxValue-minValue), 0, 1)
}

func normalizedAbs(value, maxAbs float64) float64 {
	if maxAbs <= 0 {
		return 0
	}
	return clampFloat(math.Abs(value)/maxAbs, 0, 1)
}

func wellKindLabel(well Well) string {
	if well.Missing {
		return "Missing well"
	}

	switch normalizeControl(well.Control) {
	case ControlPositive:
		return "Positive control"
	case ControlNegative:
		return "Negative control"
	case ControlEmpty:
		return "Empty well"
	default:
		if well.Hit {
			return "Hit"
		}
		return "Sample"
	}
}

func gridKey(row, col int) int {
	return (row << 8) | col
}

func clampFloat(value, minValue, maxValue float64) float64 {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
