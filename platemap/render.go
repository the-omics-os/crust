package platemap

import (
	"fmt"
	"image/color"
	"math"
	"sort"
	"strings"

	bhelp "charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

const (
	minRenderWidth  = 32
	minRenderHeight = 11
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

type replicateInfo struct {
	label      string
	key        string
	count      int
	observed   int
	hitCount   int
	meanSignal float64
	meanZ      float64
}

type legendRow struct {
	title    string
	segments []string
}

func (m Model) render() string {
	m.normalize()

	parts := []string{
		m.renderHeader(),
		m.renderGrid(),
		m.renderSeparator(),
		m.renderContextBar(),
		m.renderLegend(),
	}
	if m.inspectorVisible {
		parts = append(parts, m.renderInspector())
	}
	parts = append(parts, m.renderKeyHelp())

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m Model) renderHeader() string {
	stats := m.computeStats()
	title := m.headerTitle()
	loaded := stats.present + stats.missing
	denominator := stats.samples
	if denominator == 0 {
		denominator = stats.present
	}

	lineOneParts := []string{
		title,
		"Focus: " + coordinate(m.cursorRow, m.cursorCol),
		fmt.Sprintf("Loaded: %d/%d", loaded, m.plate.Format.WellCount()),
		fmt.Sprintf("Hits: %d/%d", stats.hits, denominator),
	}
	if m.selectedRow >= 0 {
		lineOneParts = append(lineOneParts, "Sweep: row "+rowLabel(m.selectedRow))
	}
	if m.selectedCol >= 0 {
		lineOneParts = append(lineOneParts, fmt.Sprintf("Sweep: col %d", m.selectedCol+1))
	}

	visibleRows, visibleCols := m.gridViewportSize()
	rowStart := m.rowOffset
	rowEnd := minInt(m.plate.Format.Rows(), rowStart+visibleRows)
	colStart := m.colOffset
	colEnd := minInt(m.plate.Format.Cols(), colStart+visibleCols)

	lineThree := fmt.Sprintf(
		"Window: rows %s-%s/%d | cols %d-%d/%d | Missing: %d",
		rowLabel(rowStart),
		rowLabel(rowEnd-1),
		m.plate.Format.Rows(),
		colStart+1,
		colEnd,
		m.plate.Format.Cols(),
		stats.missing,
	)

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render(strings.Join(lineOneParts, " | ")),
		m.renderLensStrip(),
		mutedStyle.Render(lineThree),
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
		if col == m.cursorCol {
			style = style.Background(m.theme.CursorBg).Bold(true)
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
		if row == m.cursorRow {
			style = style.Background(m.theme.CursorBg).Bold(true)
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

func (m Model) renderContextBar() string {
	well, ok := m.wellAt(m.cursorRow, m.cursorCol)
	textStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	parts := make([]string, 0, 4)
	if summary, active := m.selectionSummary(); active {
		parts = append(parts, m.renderCapsule("Sweep", m.theme.SelectedCol))
		parts = append(parts, mutedStyle.Render(summary))
	}
	parts = append(parts, m.renderCapsule("Focus "+coordinate(m.cursorRow, m.cursorCol), m.theme.CursorBg))
	parts = append(parts, textStyle.Render(m.focusSummary(well, ok)))
	return lipgloss.NewStyle().MaxWidth(m.renderWidth()).Render(joinStyled(parts, "  "))
}

func (m Model) renderLegend() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header)
	subtitleStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	rows := []string{
		titleStyle.Render("Legend:") + " " + subtitleStyle.Render(m.mode.String()+" view"),
		m.renderLegendRow("Always", m.baseLegendSegments()),
	}
	for _, row := range m.modeLegendRows() {
		rows = append(rows, m.renderLegendRow(row.title, row.segments))
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) renderInspector() string {
	well, ok := m.wellAt(m.cursorRow, m.cursorCol)
	coord := coordinate(m.cursorRow, m.cursorCol)
	width := maxInt(24, m.renderWidth()-4)
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Width(width)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	if !ok {
		empty := lipgloss.JoinVertical(
			lipgloss.Left,
			titleStyle.Render("Inspect "+coord),
			mutedStyle.Render("No well data loaded for this coordinate."),
			mutedStyle.Render(m.metadataSummary()),
		)
		return panelStyle.Render(empty)
	}

	titleLine := lipgloss.JoinHorizontal(
		lipgloss.Center,
		titleStyle.Render("Inspect "+coord),
		"  ",
		m.renderStatusBadges(well),
	)

	identityBlock := lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderSectionTitle("Well"),
		m.renderFieldLine("Sample", displayOrDash(well.SampleID)),
		m.renderFieldLine("Reagent", displayOrDash(well.Reagent)),
		m.renderFieldLine("Control", controlDisplayName(normalizeControl(well.Control))),
	)

	metricsBlock := m.renderMetricsBlock(width, well)

	var topSection string
	if width >= 72 {
		leftWidth := width / 2
		rightWidth := width - leftWidth - 1
		topSection = lipgloss.JoinHorizontal(
			lipgloss.Top,
			lipgloss.NewStyle().Width(leftWidth).Render(identityBlock),
			" ",
			lipgloss.NewStyle().Width(rightWidth).Render(metricsBlock),
		)
	} else {
		topSection = lipgloss.JoinVertical(lipgloss.Left, identityBlock, metricsBlock)
	}

	body := lipgloss.JoinVertical(
		lipgloss.Left,
		titleLine,
		"",
		topSection,
		"",
		m.renderReplicateBlock(well),
		"",
		m.renderMetadataBlock(),
	)
	return panelStyle.Render(body)
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
	info, ok := m.replicateInfo(target)
	if !ok {
		return "", false
	}
	return fmt.Sprintf(
		"Replicates by %s (%s): n=%d | observed=%d | mean signal=%.2f | mean z=%.2f | hits=%d",
		info.label,
		info.key,
		info.count,
		info.observed,
		info.meanSignal,
		info.meanZ,
		info.hitCount,
	), true
}

func (m Model) replicateInfo(target Well) (replicateInfo, bool) {
	key := strings.TrimSpace(target.Reagent)
	label := "reagent"
	if key == "" {
		key = strings.TrimSpace(target.SampleID)
		label = "sample_id"
	}
	if key == "" {
		return replicateInfo{}, false
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
		return replicateInfo{}, false
	}

	meanSignal := 0.0
	meanZ := 0.0
	if observed > 0 {
		meanSignal = signalSum / float64(observed)
		meanZ = zSum / float64(observed)
	}

	return replicateInfo{
		label:      label,
		key:        key,
		count:      count,
		observed:   observed,
		hitCount:   hitCount,
		meanSignal: meanSignal,
		meanZ:      meanZ,
	}, true
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

func (m Model) renderLensStrip() string {
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	activeStyle := lipgloss.NewStyle().Foreground(m.theme.Text).Background(m.theme.Header).Bold(true)
	inactiveStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	parts := []string{labelStyle.Render("Lenses:")}
	for _, mode := range allViewModes {
		label := mode.shortLabel()
		if mode == m.mode {
			parts = append(parts, activeStyle.Render(" "+label+" "))
		} else {
			parts = append(parts, inactiveStyle.Render(label))
		}
	}

	return strings.Join(parts, " ")
}

func (m Model) renderKeyHelp() string {
	helpModel := bhelp.New()
	helpModel.ShowAll = m.helpVisible
	helpModel.SetWidth(m.renderWidth())
	styles := helpModel.Styles
	styles.ShortKey = lipgloss.NewStyle().Foreground(m.theme.Header).Bold(true)
	styles.ShortDesc = lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	styles.ShortSeparator = lipgloss.NewStyle().Foreground(m.theme.Border)
	styles.FullKey = lipgloss.NewStyle().Foreground(m.theme.Header).Bold(true)
	styles.FullDesc = lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	styles.FullSeparator = lipgloss.NewStyle().Foreground(m.theme.Border)
	styles.Ellipsis = lipgloss.NewStyle().Foreground(m.theme.Border)
	helpModel.Styles = styles
	return helpModel.View(defaultKeyMap)
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

	overhead := m.headerLineCount() + 1 + m.contextLineCount() + m.legendLineCount() + m.inspectorLineCount() + m.helpLineCount()
	visibleRows := m.renderHeight() - overhead
	if visibleRows < 1 {
		visibleRows = 1
	}
	if visibleRows > rows {
		visibleRows = rows
	}

	return visibleRows, visibleCols
}

func (m Model) headerLineCount() int {
	return 3
}

func (m Model) contextLineCount() int {
	return lipgloss.Height(m.renderContextBar())
}

func (m Model) legendLineCount() int {
	return lipgloss.Height(m.renderLegend())
}

func (m Model) inspectorLineCount() int {
	if m.inspectorVisible {
		return lipgloss.Height(m.renderInspector())
	}
	return 0
}

func (m Model) helpLineCount() int {
	return lipgloss.Height(m.renderKeyHelp())
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

func (m Model) selectionSummary() (string, bool) {
	switch {
	case m.selectedRow >= 0:
		return m.bandSummary("Row "+rowLabel(m.selectedRow), m.plate.Format.Cols(), func(well Well) bool {
			return well.Row == m.selectedRow
		}), true
	case m.selectedCol >= 0:
		return m.bandSummary(fmt.Sprintf("Col %d", m.selectedCol+1), m.plate.Format.Rows(), func(well Well) bool {
			return well.Col == m.selectedCol
		}), true
	default:
		return "", false
	}
}

func (m Model) bandSummary(label string, total int, match func(Well) bool) string {
	var loaded, missing, hits, observed int
	var zSum float64
	for _, well := range m.plate.Wells {
		if !match(well) {
			continue
		}
		loaded++
		if well.Missing {
			missing++
			continue
		}
		observed++
		zSum += well.ZScore
		if well.Hit {
			hits++
		}
	}

	meanZ := "n/a"
	if observed > 0 {
		meanZ = fmt.Sprintf("%.2f", zSum/float64(observed))
	}

	return fmt.Sprintf("%s sweep | loaded %d/%d | hits %d | missing %d | mean z %s", label, loaded, total, hits, missing, meanZ)
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

func joinStyled(parts []string, separator string) string {
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}

	var builder strings.Builder
	for index, part := range parts {
		if index > 0 {
			builder.WriteString(separator)
		}
		builder.WriteString(part)
	}
	return builder.String()
}

func displayOrDash(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}

func controlDisplayName(control string) string {
	switch control {
	case ControlPositive:
		return "Positive"
	case ControlNegative:
		return "Negative"
	case ControlEmpty:
		return "Empty"
	default:
		return "Sample"
	}
}

func (m Model) renderCapsule(text string, background color.Color) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.Text).
		Background(background).
		Padding(0, 1).
		Render(text)
}

func (m Model) renderStatusBadges(well Well) string {
	var badges []string
	badges = append(badges, m.renderCapsule(strings.ToUpper(controlDisplayName(normalizeControl(well.Control))), m.theme.Border))
	if well.Hit {
		badges = append(badges, m.renderCapsule("HIT", m.theme.Hit))
	}
	if well.Missing {
		badges = append(badges, m.renderCapsule("MISSING", m.theme.Missing))
	}
	return joinStyled(badges, " ")
}

func (m Model) renderSectionTitle(title string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header).Render(title)
}

func (m Model) renderFieldLine(label, value string) string {
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Width(10)
	valueStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	return labelStyle.Render(label) + valueStyle.Render(value)
}

func (m Model) renderMetricCard(label, value string, accent color.Color) string {
	title := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render(label)
	val := lipgloss.NewStyle().Bold(true).Foreground(accent).Render(value)
	return lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(m.theme.Border).
		Padding(0, 1).
		Render(lipgloss.JoinVertical(lipgloss.Left, title, val))
}

func (m Model) renderMetricsBlock(width int, well Well) string {
	cards := []string{
		m.renderMetricCard("Signal", fmt.Sprintf("%.2f", well.Signal), m.theme.Header),
		m.renderMetricCard("Norm", fmt.Sprintf("%.2f", well.Normalized), m.theme.Sample),
		m.renderMetricCard("Z", fmt.Sprintf("%.2f", well.ZScore), zScoreColor(well.ZScore, m.theme)),
	}
	if width >= 72 {
		parts := make([]string, 0, len(cards)*2-1)
		for index, card := range cards {
			if index > 0 {
				parts = append(parts, " ")
			}
			parts = append(parts, card)
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, cards...)
}

func (m Model) renderReplicateBlock(well Well) string {
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	if info, ok := m.replicateInfo(well); ok {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderSectionTitle("Replicates"),
			m.renderFieldLine("Basis", info.label+" "+info.key),
			m.renderFieldLine("Observed", fmt.Sprintf("%d/%d", info.observed, info.count)),
			m.renderFieldLine("Hits", fmt.Sprintf("%d", info.hitCount)),
			m.renderFieldLine("Mean Z", fmt.Sprintf("%.2f", info.meanZ)),
		)
	}
	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.renderSectionTitle("Replicates"),
		mutedStyle.Render("No shared reagent or sample ID found"),
	)
}

func (m Model) renderMetadataBlock() string {
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	if len(m.plate.Metadata) == 0 {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderSectionTitle("Metadata"),
			mutedStyle.Render("None"),
		)
	}

	keys := make([]string, 0, len(m.plate.Metadata))
	for key := range m.plate.Metadata {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := []string{m.renderSectionTitle("Metadata")}
	for _, key := range keys {
		lines = append(lines, m.renderFieldLine(key, m.plate.Metadata[key]))
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) baseLegendSegments() []string {
	return []string{
		m.renderLegendToken("+", "positive ctrl", m.theme.PositiveCtrl),
		m.renderLegendToken("-", "negative ctrl", m.theme.NegativeCtrl),
		m.renderLegendToken("o", "empty", m.theme.Empty),
		m.renderLegendToken("x", "missing", m.theme.Missing),
	}
}

func (m Model) modeLegendRows() []legendRow {
	switch m.mode {
	case ViewRawSignal:
		return []legendRow{
			{
				title: "Signal",
				segments: []string{
					m.renderLegendToken("#", "hit overlay", m.theme.Hit),
					m.renderLegendToken(".", "lowest", sampleBucketColor(0.00, m.theme)),
					m.renderLegendToken(":", "low", sampleBucketColor(0.25, m.theme)),
					m.renderLegendToken("*", "medium", sampleBucketColor(0.50, m.theme)),
					m.renderLegendToken("O", "high", sampleBucketColor(0.75, m.theme)),
					m.renderLegendToken("@", "highest", sampleBucketColor(1.00, m.theme)),
				},
			},
		}
	case ViewNormalized:
		return []legendRow{
			{
				title: "Normalized",
				segments: []string{
					m.renderLegendToken("#", "hit overlay", m.theme.Hit),
					m.renderLegendToken(".", "lowest", sampleBucketColor(0.00, m.theme)),
					m.renderLegendToken(":", "low", sampleBucketColor(0.25, m.theme)),
					m.renderLegendToken("*", "medium", sampleBucketColor(0.50, m.theme)),
					m.renderLegendToken("O", "high", sampleBucketColor(0.75, m.theme)),
					m.renderLegendToken("@", "highest", sampleBucketColor(1.00, m.theme)),
				},
			},
		}
	case ViewZScore:
		return []legendRow{
			{
				title: "Magnitude",
				segments: []string{
					m.renderLegendToken("#", "hit overlay", m.theme.Hit),
					m.renderLegendToken(".", "near 0", m.theme.TextMuted),
					m.renderLegendToken(":", "mild", m.theme.TextMuted),
					m.renderLegendToken("*", "moderate", m.theme.Text),
					m.renderLegendToken("O", "strong", m.theme.Header),
					m.renderLegendToken("@", "extreme", m.theme.Header),
				},
			},
			{
				title: "Color",
				segments: []string{
					m.renderLegendWord("negative", m.theme.NegativeCtrl),
					m.renderLegendWord("neutral", m.theme.TextMuted),
					m.renderLegendWord("positive", m.theme.PositiveCtrl),
				},
			},
		}
	case ViewHitClass:
		return []legendRow{
			{
				title: "Samples",
				segments: []string{
					m.renderLegendToken("#", "hit", m.theme.Hit),
					m.renderLegendToken(".", "non-hit sample", m.theme.Sample),
				},
			},
		}
	case ViewControlLayout:
		return []legendRow{
			{
				title: "Samples",
				segments: []string{
					m.renderLegendToken(".", "sample wells", m.theme.Sample),
				},
			},
		}
	case ViewMissingness:
		return []legendRow{
			{
				title: "Samples",
				segments: []string{
					m.renderLegendToken(".", "observed sample", m.theme.Sample),
				},
			},
		}
	default:
		return nil
	}
}

func (v ViewMode) shortLabel() string {
	switch v {
	case ViewRawSignal:
		return "Raw"
	case ViewNormalized:
		return "Norm"
	case ViewZScore:
		return "Z"
	case ViewHitClass:
		return "Hit"
	case ViewControlLayout:
		return "Ctrl"
	case ViewMissingness:
		return "Missing"
	default:
		return "Raw"
	}
}

func (m Model) renderLegendRow(title string, segments []string) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.TextMuted)
	prefix := titleStyle.Render(title + ":")
	if len(segments) == 0 {
		return prefix
	}

	const separator = "  "
	hangingIndent := strings.Repeat(" ", lipgloss.Width(prefix)+1)
	lines := make([]string, 0, 2)
	current := prefix

	for _, segment := range segments {
		candidate := segment
		if current == prefix {
			candidate = current + " " + segment
		} else {
			candidate = current + separator + segment
		}

		if lipgloss.Width(candidate) > m.renderWidth() {
			lines = append(lines, current)
			current = hangingIndent + segment
			continue
		}
		current = candidate
	}

	if current != "" {
		lines = append(lines, current)
	}

	return strings.Join(lines, "\n")
}

func (m Model) renderLegendToken(glyph, label string, foreground color.Color) string {
	glyphStyle := lipgloss.NewStyle().
		Width(3).
		Align(lipgloss.Center).
		Bold(true).
		Foreground(foreground).
		Background(m.theme.CursorBg)

	labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

	return lipgloss.JoinHorizontal(
		lipgloss.Center,
		glyphStyle.Render(glyph),
		" ",
		labelStyle.Render(label),
	)
}

func (m Model) renderLegendWord(label string, foreground color.Color) string {
	return lipgloss.NewStyle().Bold(true).Foreground(foreground).Render(label)
}
