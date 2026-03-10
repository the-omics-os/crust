package sequenceviewer

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

type lineKind int

const (
	lineKindUnknown lineKind = iota
	lineKindAnnotation
	lineKindSequence
	lineKindComplement
	lineKindProperty
)

type lineMeta struct {
	Start int
	End   int
	Kind  lineKind
}

func (m *Model) syncViewport() {
	m.ensureViewport()
	lines, meta := m.buildContentLines()
	m.lineMeta = meta
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(m.contentHeight())
	m.viewport.SetContentLines(lines)
	m.ensureFocusVisible()

	maxOffset := 0
	if len(lines) > m.contentHeight() {
		maxOffset = len(lines) - m.contentHeight()
	}
	if m.viewport.YOffset() > maxOffset {
		m.viewport.SetYOffset(maxOffset)
	}
}

func (m *Model) ensureFocusVisible() {
	target := m.focusLineIndex()
	if target < 0 {
		return
	}

	top := m.viewport.YOffset()
	height := m.contentHeight()
	bottom := top + height - 1
	padding := 1

	switch {
	case target < top+padding:
		offset := target - padding
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	case target > bottom-padding:
		offset := target - height + 1 + padding
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	}
}

func (m Model) focusLineIndex() int {
	position := m.FocusPosition()
	if position == 0 {
		return -1
	}
	for i, meta := range m.lineMeta {
		if meta.Kind != lineKindSequence {
			continue
		}
		if position >= meta.Start && position <= meta.End {
			return i
		}
	}
	return -1
}

func (m Model) buildContentLines() ([]string, []lineMeta) {
	if len(m.residues) == 0 {
		msgStyle := lipgloss.NewStyle().Foreground(m.theme.Unknown)
		return []string{
			msgStyle.Render("No sequence loaded."),
			msgStyle.Render("Use WithSequence(...) or SetSequence(...) to populate the viewer."),
		}, []lineMeta{{Kind: lineKindUnknown}, {Kind: lineKindUnknown}}
	}

	effectivePerLine := m.effectiveResiduesPerLine()
	labelWidth := maxInt(3, len(strconv.Itoa(m.maxResiduePosition())))
	coreWidth := groupedDisplayWidth(effectivePerLine, groupSizeForType(m.seqType))

	lines := make([]string, 0, len(m.residues))
	meta := make([]lineMeta, 0, len(m.residues))

	for i := 0; i < len(m.residues); i += effectivePerLine {
		end := i + effectivePerLine
		if end > len(m.residues) {
			end = len(m.residues)
		}
		chunk := m.residues[i:end]
		startPos := chunk[0].Position
		endPos := chunk[len(chunk)-1].Position

		if line := m.renderAnnotationLine(chunk, labelWidth, coreWidth); line != "" {
			lines = append(lines, line)
			meta = append(meta, lineMeta{Start: startPos, End: endPos, Kind: lineKindAnnotation})
		}

		lines = append(lines, m.renderSequenceLine(chunk, labelWidth, coreWidth))
		meta = append(meta, lineMeta{Start: startPos, End: endPos, Kind: lineKindSequence})

		if m.showComplement && m.seqType == DNA {
			lines = append(lines, m.renderComplementLine(chunk, labelWidth, coreWidth))
			meta = append(meta, lineMeta{Start: startPos, End: endPos, Kind: lineKindComplement})
		}

		if m.view != IdentityView {
			lines = append(lines, m.renderPropertyLine(chunk, labelWidth, coreWidth))
			meta = append(meta, lineMeta{Start: startPos, End: endPos, Kind: lineKindProperty})
		}
	}

	return lines, meta
}

func (m Model) renderHeader() string {
	lines := []string{
		lipgloss.NewStyle().Foreground(m.theme.Header).Bold(true).Render(m.renderHeaderSummary()),
		m.renderViewBar(),
		lipgloss.NewStyle().Foreground(m.theme.Hint).Render(truncatePlain(m.renderFocusSummary(), m.width)),
	}
	return strings.Join(lines, "\n")
}

func (m Model) renderHeaderSummary() string {
	unit := "residues"
	switch m.seqType {
	case DNA:
		unit = "bp"
	case RNA:
		unit = "nt"
	case Protein:
		unit = "aa"
	}

	segments := []string{
		fmt.Sprintf("%s Sequence", m.seqType),
		fmt.Sprintf("%d %s", len(m.residues), unit),
	}

	switch m.seqType {
	case DNA, RNA:
		segments = append(segments,
			fmt.Sprintf("GC %.1f%%", m.overallGC*100),
			fmt.Sprintf("Tm %.1fC", m.meltingTemp),
		)
		if len(m.orfs) > 0 {
			segments = append(segments, fmt.Sprintf("ORFs %d", len(m.orfs)))
		}
	case Protein:
		segments = append(segments,
			fmt.Sprintf("pI %.2f", m.isoelectricPoint),
			fmt.Sprintf("MW %.1f Da", m.totalWeight),
		)
	}

	if len(m.annotations) > 0 {
		segments = append(segments, fmt.Sprintf("Features %d", len(m.annotations)))
	}

	return fitSegments(segments, m.width)
}

func (m Model) renderViewBar() string {
	prefix := lipgloss.NewStyle().Foreground(m.theme.Hint).Render("Views")
	views := ApplicableViews(m.seqType)
	parts := make([]string, 0, len(views)+1)
	parts = append(parts, prefix)
	for _, view := range views {
		style := lipgloss.NewStyle().Foreground(m.theme.ViewLabel).Padding(0, 1)
		if view == m.view {
			style = lipgloss.NewStyle().
				Foreground(m.theme.FocusForeground).
				Background(m.theme.FocusBackground).
				Bold(true).
				Padding(0, 1)
		}
		parts = append(parts, style.Render(view.String()))
	}

	line := strings.Join(parts, " ")
	if lipgloss.Width(line) <= m.width {
		return line
	}

	return lipgloss.NewStyle().
		Foreground(m.theme.ViewLabel).
		Render(truncatePlain("View "+m.view.String()+" (Tab cycles)", m.width))
}

func (m Model) renderFocusSummary() string {
	residue, ok := m.focusedResidue()
	if !ok {
		return "Focus: none"
	}

	segments := []string{
		fmt.Sprintf("Focus %d %c", residue.Position, residue.Code),
	}
	if summary := m.selectionSummary(); summary != "" {
		segments = append(segments, summary)
	}

	switch m.seqType {
	case DNA, RNA:
		segments = append(segments,
			fmt.Sprintf("Complement %c", Complement(residue.Code, m.seqType)),
			fmt.Sprintf("GC window %.0f%%", residue.Properties.GCWindow*100),
		)
	case Protein:
		switch m.view {
		case HydrophobicityView:
			segments = append(segments, fmt.Sprintf("Hydrophobicity %.1f", residue.Properties.Hydrophobicity))
		case ChargeView:
			segments = append(segments, fmt.Sprintf("Charge %+0.1f", residue.Properties.Charge))
		case MolWeightView:
			segments = append(segments, fmt.Sprintf("Mol wt %.1f Da", residue.Properties.MolWeight))
		default:
			segments = append(segments,
				fmt.Sprintf("Charge %+0.1f", residue.Properties.Charge),
				fmt.Sprintf("Mol wt %.1f Da", residue.Properties.MolWeight),
			)
		}
	}

	if summary := m.focusRegionSummary(); summary != "" {
		segments = append(segments, summary)
	}

	return fitSegments(segments, m.width)
}

func (m Model) selectionSummary() string {
	start, end, ok := m.SelectionRange()
	if !ok {
		return ""
	}
	return fmt.Sprintf("Selection %d-%d (%d %s)", start, end, m.selectionCount(), m.residueUnit())
}

func (m Model) focusRegionSummary() string {
	position := m.FocusPosition()
	if position == 0 {
		return ""
	}

	var parts []string
	if annotations := m.annotationNamesAt(position); len(annotations) > 0 {
		parts = append(parts, "Features "+strings.Join(annotations, ", "))
	}
	if siteNames := m.restrictionNamesAt(position); len(siteNames) > 0 {
		parts = append(parts, "Sites "+strings.Join(siteNames, ", "))
	}
	if orf, ok := m.orfAt(position); ok {
		parts = append(parts, fmt.Sprintf("ORF frame %d", orf.Frame))
	}

	return strings.Join(parts, " | ")
}

func (m Model) annotationNamesAt(position int) []string {
	var names []string
	for _, annotation := range m.annotations {
		if position >= annotation.Start && position <= annotation.End {
			names = append(names, annotation.Name)
		}
	}
	return names
}

func (m Model) restrictionNamesAt(position int) []string {
	var names []string
	for _, site := range m.restrictionSites {
		if position >= site.Start && position <= site.End {
			names = append(names, site.Enzyme.Name)
		}
	}
	return names
}

func (m Model) orfAt(position int) (ORF, bool) {
	for _, orf := range m.orfs {
		if position >= orf.Start && position <= orf.End {
			return orf, true
		}
	}
	return ORF{}, false
}

func (m Model) renderSeparator() string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Separator).
		Render(strings.Repeat("-", maxInt(m.width, minWidth)))
}

func (m Model) renderFooter() string {
	segments := []string{
		"Left/Right residue",
		"Up/Down row",
		"Shift+Arrows select",
		"PgUp/PgDn page",
		"[/] feature",
		"Tab lens",
	}
	if m.seqType == DNA {
		segments = append(segments, "c complement")
	}
	segments = append(segments, "?: help + legend")
	if start, end, ok := m.SelectionRange(); ok {
		segments = append(segments, fmt.Sprintf("Sel %d-%d", start, end))
	}
	if pos := m.FocusPosition(); pos > 0 {
		segments = append(segments, fmt.Sprintf("Pos %d/%d", pos, len(m.residues)))
	}

	return lipgloss.NewStyle().
		Foreground(m.theme.Separator).
		Render(fitSegments(segments, m.width))
}

func (m Model) renderHelp() string {
	return strings.Join(m.helpLines(), "\n")
}

func (m Model) helpLines() []string {
	moveLine := "Move: Left/Right residue, Up/Down row, PgUp/PgDn page, Home/End sequence bounds."
	selectLine := "Select: Shift with arrows extends a contiguous range from the focused residue. Shift+PgUp/PgDn/Home/End also works. Esc clears the range."
	jumpLine := "Jump: [ and ] move across annotated regions. DNA falls back to ORFs when no annotations exist."
	if len(m.annotations) == 0 && m.seqType != DNA {
		jumpLine = "Jump: [ and ] are reserved for feature jumps when annotations are available."
	}

	lensLine := "Lens: Tab cycles the available views for the active sequence type."
	if m.seqType == DNA {
		lensLine += " c toggles the complement strand."
	}

	lines := []string{
		lipgloss.NewStyle().Foreground(m.theme.ViewLabel).Bold(true).Render("Help"),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain(moveLine, m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain(selectLine, m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain(jumpLine, m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain(lensLine, m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Hint).Render(truncatePlain("The highlighted residue is the focus. The header explains that residue in the current lens.", m.width)),
	}
	lines = append(lines, m.legendLines()...)
	return lines
}

func (m Model) legendLines() []string {
	lines := []string{
		lipgloss.NewStyle().Foreground(m.theme.ViewLabel).Bold(true).Render("Legend"),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain("Highlight: bright cell = focus, muted band = selection.", m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(m.renderFeatureLegendLine()),
	}

	identityLine := m.renderIdentityLegendLine()
	if identityLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.Header).Render(identityLine))
	}

	if propertyLine := m.renderPropertyLegendLine(); propertyLine != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.Header).Render(propertyLine))
	}

	return lines
}

func (m Model) renderFeatureLegendLine() string {
	segments := []string{
		m.legendToken(">", m.theme.ViewLabel) + " forward start",
		m.legendToken("<", m.theme.ViewLabel) + " reverse end",
		m.legendToken("|", m.theme.ViewLabel) + " boundary",
		m.legendToken("=", m.theme.ViewLabel) + " feature body",
	}
	full := "Features: " + strings.Join(segments, "  ")
	if lipgloss.Width(full) <= m.width {
		return full
	}
	return truncatePlain("Features: > forward start, < reverse end, | boundary, = feature body.", m.width)
}

func (m Model) renderIdentityLegendLine() string {
	switch m.seqType {
	case DNA, RNA:
		tOrU := m.legendToken("T", m.theme.Thymine)
		label := "thymine"
		if m.seqType == RNA {
			tOrU = m.legendToken("U", m.theme.Uracil)
			label = "uracil"
		}
		segments := []string{
			m.legendToken("A", m.theme.Adenine) + " adenine",
			tOrU + " " + label,
			m.legendToken("G", m.theme.Guanine) + " guanine",
			m.legendToken("C", m.theme.Cytosine) + " cytosine",
		}
		full := "Identity colors: " + strings.Join(segments, "  ")
		if lipgloss.Width(full) <= m.width {
			return full
		}
		return truncatePlain("Identity colors: A adenine, T/U thymine or uracil, G guanine, C cytosine.", m.width)
	case Protein:
		segments := []string{
			m.legendToken("I", m.theme.Hydrophobic) + " hydrophobic",
			m.legendToken("K", m.theme.Positive) + " positive",
			m.legendToken("D", m.theme.Negative) + " negative",
			m.legendToken("S", m.theme.Polar) + " polar",
			m.legendToken("F", m.theme.Aromatic) + " aromatic",
			m.legendToken("G", m.theme.Special) + " special",
		}
		full := "Identity colors: " + strings.Join(segments, "  ")
		if lipgloss.Width(full) <= m.width {
			return full
		}
		return truncatePlain("Identity colors: hydrophobic, positive, negative, polar, aromatic, and special residues each have their own color.", m.width)
	default:
		return ""
	}
}

func (m Model) renderPropertyLegendLine() string {
	if m.view == IdentityView {
		return ""
	}
	ramp := m.legendPropertyRamp()
	full := "Property bar: " + ramp + " low -> high"
	if lipgloss.Width(full) <= m.width {
		return full
	}
	return truncatePlain("Property bar: .:-=+*#%@ reads from low to high.", m.width)
}

func (m Model) legendPropertyRamp() string {
	glyphs := []string{".", ":", "-", "=", "+", "*", "#", "%", "@"}
	parts := make([]string, 0, len(glyphs))
	for i, glyph := range glyphs {
		color := m.scaledGradient(float64(i) / float64(len(glyphs)-1))
		parts = append(parts, m.legendToken(glyph, color))
	}
	return strings.Join(parts, " ")
}

func (m Model) legendToken(text string, fg color.Color) string {
	return lipgloss.NewStyle().Foreground(fg).Bold(true).Render(text)
}

func (m Model) renderSequenceLine(chunk []Residue, labelWidth, coreWidth int) string {
	plainCoreWidth := groupedDisplayWidth(len(chunk), groupSizeForType(m.seqType))
	pad := maxInt(2, coreWidth-plainCoreWidth+2)
	focused := m.chunkContainsFocus(chunk)

	lineNumberStyle := lipgloss.NewStyle().Foreground(m.theme.LineNumber)
	if focused {
		lineNumberStyle = lineNumberStyle.Foreground(m.theme.ViewLabel).Bold(true)
	}

	left := lineNumberStyle.Render(fmt.Sprintf("%*d  ", labelWidth, chunk[0].Position))
	right := lineNumberStyle.Render(fmt.Sprintf("%*d", labelWidth, chunk[len(chunk)-1].Position))
	core := m.renderGrouped(chunk, func(residue Residue) string {
		return m.styledResidue(residue)
	})

	return left + core + strings.Repeat(" ", pad) + right
}

func (m Model) renderComplementLine(chunk []Residue, labelWidth, coreWidth int) string {
	plainCoreWidth := groupedDisplayWidth(len(chunk), groupSizeForType(m.seqType))
	pad := coreWidth - plainCoreWidth
	if pad < 0 {
		pad = 0
	}
	prefix := strings.Repeat(" ", labelWidth+2)
	core := m.renderGrouped(chunk, func(residue Residue) string {
		return m.styledComplementResidue(residue)
	})
	return prefix + core + strings.Repeat(" ", pad)
}

func (m Model) renderPropertyLine(chunk []Residue, labelWidth, coreWidth int) string {
	plainCoreWidth := groupedDisplayWidth(len(chunk), groupSizeForType(m.seqType))
	pad := coreWidth - plainCoreWidth
	if pad < 0 {
		pad = 0
	}
	prefix := strings.Repeat(" ", labelWidth+2)
	core := m.renderGrouped(chunk, func(residue Residue) string {
		return m.styledPropertyGlyph(residue)
	})
	return prefix + core + strings.Repeat(" ", pad)
}

func (m Model) renderAnnotationLine(chunk []Residue, labelWidth, coreWidth int) string {
	plainCoreWidth := groupedDisplayWidth(len(chunk), groupSizeForType(m.seqType))
	pad := coreWidth - plainCoreWidth
	if pad < 0 {
		pad = 0
	}

	prefix := strings.Repeat(" ", labelWidth+2)
	groupSize := groupSizeForType(m.seqType)
	var b strings.Builder
	b.WriteString(prefix)
	hasAnnotation := false
	for i, residue := range chunk {
		if i > 0 && i%groupSize == 0 {
			b.WriteByte(' ')
		}
		marker, markerColor, ok := m.annotationMarkerAt(residue.Position)
		if !ok {
			b.WriteByte(' ')
			continue
		}
		hasAnnotation = true

		style := lipgloss.NewStyle().Foreground(markerColor)
		style = m.decoratePositionStyle(style, residue.Position)
		b.WriteString(style.Render(marker))
	}
	if !hasAnnotation {
		return ""
	}
	b.WriteString(strings.Repeat(" ", pad))
	return b.String()
}

func (m Model) renderGrouped(chunk []Residue, renderResidue func(Residue) string) string {
	groupSize := groupSizeForType(m.seqType)
	var b strings.Builder
	for i, residue := range chunk {
		if i > 0 && i%groupSize == 0 {
			b.WriteByte(' ')
		}
		b.WriteString(renderResidue(residue))
	}
	return b.String()
}

func (m Model) annotationMarkerAt(position int) (string, color.Color, bool) {
	for i := range m.annotations {
		annotation := m.annotations[i]
		if position < annotation.Start || position > annotation.End {
			continue
		}

		marker := "="
		switch {
		case position == annotation.Start && annotation.Direction > 0:
			marker = ">"
		case position == annotation.End && annotation.Direction < 0:
			marker = "<"
		case position == annotation.Start || position == annotation.End:
			marker = "|"
		}

		annotationColor := annotation.Color
		if annotationColor == nil {
			annotationColor = m.theme.ViewLabel
		}
		return marker, annotationColor, true
	}
	return "", nil, false
}

func (m Model) annotationColorAt(position int) (color.Color, bool) {
	for i := range m.annotations {
		annotation := m.annotations[i]
		if position < annotation.Start || position > annotation.End {
			continue
		}
		if annotation.Color != nil {
			return annotation.Color, true
		}
		return m.theme.ViewLabel, true
	}
	return nil, false
}

func (m Model) visibleSpan() (int, int) {
	if len(m.lineMeta) == 0 {
		return 0, 0
	}
	startLine := m.viewport.YOffset()
	if startLine < 0 {
		startLine = 0
	}
	endLine := startLine + maxInt(m.viewport.VisibleLineCount(), m.viewport.Height())
	if endLine > len(m.lineMeta) {
		endLine = len(m.lineMeta)
	}

	start, end := 0, 0
	for _, line := range m.lineMeta[startLine:endLine] {
		if line.Start == 0 {
			continue
		}
		if start == 0 || line.Start < start {
			start = line.Start
		}
		if line.End > end {
			end = line.End
		}
	}
	return start, end
}

func (m Model) effectiveResiduesPerLine() int {
	preferred := m.residuesPerLine
	if preferred <= 0 {
		preferred = len(m.residues)
	}
	if preferred <= 1 {
		return 1
	}

	groupSize := groupSizeForType(m.seqType)
	labelWidth := maxInt(3, len(strconv.Itoa(m.maxResiduePosition())))
	available := m.width - ((labelWidth + 2) + 2 + labelWidth)
	if available < 1 {
		return 1
	}
	if preferred > len(m.residues) {
		preferred = len(m.residues)
	}
	if preferred > available {
		preferred = available
	}

	best := 1
	for candidate := 1; candidate <= preferred; candidate++ {
		if groupedDisplayWidth(candidate, groupSize) <= available {
			best = candidate
		}
	}
	return best
}

func (m Model) maxResiduePosition() int {
	maxPos := 0
	for _, residue := range m.residues {
		if residue.Position > maxPos {
			maxPos = residue.Position
		}
	}
	if maxPos == 0 {
		return len(m.residues)
	}
	return maxPos
}

func (m Model) chunkContainsFocus(chunk []Residue) bool {
	position := m.FocusPosition()
	if position == 0 || len(chunk) == 0 {
		return false
	}
	return position >= chunk[0].Position && position <= chunk[len(chunk)-1].Position
}

func groupedDisplayWidth(count, groupSize int) int {
	if count <= 0 {
		return 0
	}
	if groupSize <= 0 {
		groupSize = count
	}
	return count + (count-1)/groupSize
}

func fitSegments(segments []string, width int) string {
	if width <= 0 {
		return strings.Join(segments, " | ")
	}
	var chosen []string
	for _, segment := range segments {
		candidate := append(chosen, segment)
		line := strings.Join(candidate, " | ")
		if len(line) <= width {
			chosen = candidate
			continue
		}
		if len(chosen) == 0 {
			return truncatePlain(segment, width)
		}
		break
	}
	return strings.Join(chosen, " | ")
}

func truncatePlain(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (m Model) residueUnit() string {
	switch m.seqType {
	case DNA:
		return "bp"
	case RNA:
		return "nt"
	case Protein:
		return "aa"
	default:
		return "residues"
	}
}
