package sequenceviewer

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

type lineMeta struct {
	Start int
	End   int
}

func (m *Model) syncViewport() {
	m.ensureViewport()
	lines, meta := m.buildContentLines()
	m.lineMeta = meta
	m.viewport.SetWidth(m.width)
	m.viewport.SetHeight(m.contentHeight())
	m.viewport.SetContentLines(lines)

	maxOffset := 0
	if len(lines) > m.contentHeight() {
		maxOffset = len(lines) - m.contentHeight()
	}
	if m.viewport.YOffset() > maxOffset {
		m.viewport.SetYOffset(maxOffset)
	}
}

func (m Model) buildContentLines() ([]string, []lineMeta) {
	if len(m.residues) == 0 {
		msgStyle := lipgloss.NewStyle().Foreground(m.theme.Unknown)
		return []string{
			msgStyle.Render("No sequence loaded."),
			msgStyle.Render("Use WithSequence(...) or SetSequence(...) to populate the viewer."),
		}, []lineMeta{{}, {}}
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
			meta = append(meta, lineMeta{Start: startPos, End: endPos})
		}

		lines = append(lines, m.renderSequenceLine(chunk, labelWidth, coreWidth))
		meta = append(meta, lineMeta{Start: startPos, End: endPos})

		if m.showComplement && m.seqType == DNA {
			lines = append(lines, m.renderComplementLine(chunk, labelWidth, coreWidth))
			meta = append(meta, lineMeta{Start: startPos, End: endPos})
		}

		if m.view != IdentityView {
			lines = append(lines, m.renderPropertyLine(chunk, labelWidth, coreWidth))
			meta = append(meta, lineMeta{Start: startPos, End: endPos})
		}
	}

	return lines, meta
}

func (m Model) renderHeader() string {
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
		fmt.Sprintf("%s Sequence (%d %s)", m.seqType, len(m.residues), unit),
		fmt.Sprintf("View: %s", m.view),
	}

	switch m.seqType {
	case DNA, RNA:
		segments = append(segments,
			fmt.Sprintf("GC: %.1f%%", m.overallGC*100),
			fmt.Sprintf("Tm: %.1fC", m.meltingTemp),
		)
		if len(m.orfs) > 0 {
			segments = append(segments, fmt.Sprintf("ORFs: %d", len(m.orfs)))
		}
	case Protein:
		segments = append(segments,
			fmt.Sprintf("pI: %.2f", m.isoelectricPoint),
			fmt.Sprintf("MW: %.1f Da", m.totalWeight),
		)
	}

	if len(m.annotations) > 0 {
		segments = append(segments, fmt.Sprintf("Features: %d", len(m.annotations)))
	}

	line := fitSegments(segments, m.width)
	return lipgloss.NewStyle().Foreground(m.theme.Header).Bold(true).Render(line)
}

func (m Model) renderSeparator() string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Separator).
		Render(strings.Repeat("-", maxInt(m.width, minWidth)))
}

func (m Model) renderFooter() string {
	start, end := m.visibleSpan()
	segments := []string{
		"Tab: view",
		"Arrows: scroll",
		"PgUp/PgDn: page",
		"Home/End: jump",
	}
	if m.seqType == DNA {
		segments = append(segments, "c: complement")
	}
	segments = append(segments, "?: help")
	if start > 0 && end > 0 {
		segments = append(segments, fmt.Sprintf("Pos %d-%d of %d", start, end, len(m.residues)))
	}

	line := fitSegments(segments, m.width)
	return lipgloss.NewStyle().Foreground(m.theme.Separator).Render(line)
}

func (m Model) renderHelp() string {
	lines := []string{
		lipgloss.NewStyle().Foreground(m.theme.ViewLabel).Bold(true).Render("Help"),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain("Up/Down scroll one line. PgUp/PgDn page. Home/End jump to bounds.", m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain(m.helpModeLine(), m.width)),
		lipgloss.NewStyle().Foreground(m.theme.Header).Render(truncatePlain("Annotations are drawn as inline tracks and highlighted residues are underlined.", m.width)),
	}
	return strings.Join(lines, "\n")
}

func (m Model) helpModeLine() string {
	views := ApplicableViews(m.seqType)
	names := make([]string, 0, len(views))
	for _, view := range views {
		names = append(names, view.String())
	}
	line := "Tab cycles views: " + strings.Join(names, ", ") + "."
	if m.seqType == DNA {
		line += " c toggles the complement strand."
	}
	return line
}

func (m Model) renderSequenceLine(chunk []Residue, labelWidth, coreWidth int) string {
	plainCoreWidth := groupedDisplayWidth(len(chunk), groupSizeForType(m.seqType))
	pad := maxInt(2, coreWidth-plainCoreWidth+2)

	lineNumberStyle := lipgloss.NewStyle().Foreground(m.theme.LineNumber)
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
		return m.styledComplement(residue.Code)
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
		b.WriteString(lipgloss.NewStyle().Foreground(markerColor).Render(marker))
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

		color := annotation.Color
		if color == nil {
			color = m.theme.ViewLabel
		}
		return marker, color, true
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
		preferred = defaultResiduesPerLine(m.seqType)
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
