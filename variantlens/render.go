package variantlens

import (
	"fmt"
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

type renderWindow struct {
	referenceStart int
	referenceEnd   int
	visibleStart   int
	visibleEnd     int
	markerIndex    int
	variantSpan    int
	refAligned     string
	altAligned     string
	clippedLeft    bool
	clippedRight   bool
	codonRef       string
	codonAlt       string
	aaRef          string
	aaAlt          string
}

func (m Model) render() string {
	width := clampMin(m.width, minWidth)
	if len(m.context.Variants) == 0 {
		return m.renderEmpty(width)
	}

	current := m.context.Variants[m.selected]
	window := m.buildRenderWindow(current)

	lines := make([]string, 0, 32)
	if m.showHelp {
		lines = append(lines, m.renderHelpBox(width))
	}
	lines = append(lines, m.renderHeader(current, width))
	lines = append(lines, m.renderSeparator(width))
	lines = append(lines, m.renderSequenceSection(window)...)
	lines = append(lines, m.renderSeparator(width))
	lines = append(lines, m.renderFeatureSection(window, current.Position, width)...)
	lines = append(lines, m.renderSeparator(width))
	lines = append(lines, m.renderBody(current, window, width)...)
	if m.detail {
		lines = append(lines, m.renderSeparator(width))
		lines = append(lines, m.renderExpandedDetail(current, width))
	}

	return lipgloss.JoinVertical(lipgloss.Left, compactStrings(lines)...)
}

func (m Model) renderEmpty(width int) string {
	body := []string{
		"No variants loaded.",
		"Use WithContext or SetContext to provide a reference sequence, variant list, and nearby features.",
	}
	return m.renderBox("VariantLens", body, width)
}

func (m Model) renderHeader(current Variant, width int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Header)
	textStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	mutedStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	impactStyle := lipgloss.NewStyle().Bold(true).Foreground(m.impactColor(current.Impact))

	headerParts := []string{
		fmt.Sprintf("Variant %d/%d", m.selected+1, len(m.context.Variants)),
		coalesce(current.Gene, "Unassigned gene"),
	}
	if hgvs := primaryHGVS(current.HGVS); hgvs != "" {
		headerParts = append(headerParts, hgvs)
	}
	if consequence := coalesce(current.Consequence, current.Type); consequence != "" {
		headerParts = append(headerParts, consequence)
	}
	if protein := proteinHGVS(current.HGVS); protein != "" {
		headerParts = append(headerParts, protein)
	}

	status := []string{
		fmt.Sprintf("view %s", m.viewMode.String()),
		fmt.Sprintf("+/-%dbp", m.context.ContextSize),
	}
	if m.detail {
		status = append(status, "detail armed")
	}

	line1Plain := strings.Join(headerParts, " | ")
	line2Plain := strings.Join(status, " | ")
	if len(line1Plain) > width {
		line1Plain = truncatePlain(line1Plain, width)
	}
	if len(line2Plain) > width {
		line2Plain = truncatePlain(line2Plain, width)
	}
	line1 := titleStyle.Render(line1Plain)
	line2 := textStyle.Render(line2Plain) + " | " + impactStyle.Render(coalesce(current.Impact, "UNSPECIFIED"))

	return lipgloss.JoinVertical(lipgloss.Left, line1, mutedStyle.Render(line2))
}

func (m Model) renderSeparator(width int) string {
	return lipgloss.NewStyle().
		Foreground(m.theme.Border).
		Render(strings.Repeat("-", clampMin(width, minWidth)))
}

func (m Model) renderSequenceSection(window renderWindow) []string {
	labelStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Width(8)
	textStyle := lipgloss.NewStyle().Foreground(m.theme.Text)
	caretStyle := lipgloss.NewStyle().Foreground(m.theme.Selection).Bold(true)

	leftTrim := ""
	rightTrim := ""
	leftTrimWidth := 0
	if window.clippedLeft {
		leftTrim = lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("...")
		leftTrimWidth = 3
	}
	if window.clippedRight {
		rightTrim = lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("...")
	}

	refSeq := leftTrim + m.renderGroupedSequence(window.refAligned, window.markerIndex, window.variantSpan, m.theme.RefBase) + rightTrim
	altSeq := leftTrim + m.renderGroupedSequence(window.altAligned, window.markerIndex, window.variantSpan, m.theme.AltBase) + rightTrim

	markerOffset := groupedIndex(window.markerIndex) + leftTrimWidth
	marker := strings.Repeat(" ", markerOffset) + caretStyle.Render("^")

	codonSummary := "No CDS frame context available for codon translation"
	if window.codonRef != "" {
		if window.aaRef != "" || window.aaAlt != "" {
			codonSummary = fmt.Sprintf("%s -> %s (%s -> %s)", coalesce(window.aaRef, "?"), coalesce(window.aaAlt, "?"), window.codonRef, window.codonAlt)
		} else {
			codonSummary = fmt.Sprintf("%s -> %s", window.codonRef, window.codonAlt)
		}
	}

	return []string{
		labeledLine(labelStyle, textStyle, "coords:", fmt.Sprintf("window %d-%d  focus %d", window.visibleStart, window.visibleEnd, window.visibleStart+window.markerIndex)),
		labeledLine(labelStyle, textStyle, "ref:", refSeq),
		labeledLine(labelStyle, textStyle, "alt:", altSeq),
		labeledLine(labelStyle, textStyle, "", marker),
		labeledLine(labelStyle, textStyle, "codon:", codonSummary),
	}
}

func (m Model) renderFeatureSection(window renderWindow, position, width int) []string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("Features")}
	features := overlappingFeatures(m.context.Features, window.visibleStart, window.visibleEnd)
	if len(features) == 0 {
		lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("  No annotated features in the visible window."))
		return lines
	}

	trackWidth := clampInt(width-30, 12, 56)
	for _, feature := range features {
		bar := m.featureBar(feature, window.visibleStart, window.visibleEnd, position, trackWidth)
		color := m.featureColor(feature.Type)
		label := clipAndPad(coalesce(feature.Name, feature.Type), 16)
		rangeText := fmt.Sprintf("%d-%d", feature.Start, feature.End)
		featureStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
		rangeStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)

		lines = append(lines, "  "+featureStyle.Render(label)+" "+bar+" "+rangeStyle.Render(rangeText))
	}
	return lines
}

func (m Model) renderBody(current Variant, window renderWindow, width int) []string {
	switch m.viewMode {
	case ViewDetail:
		return m.renderDetailBody(current, window, width)
	case ViewHGVS:
		return m.renderHGVSBody(current, width)
	case ViewEvidence:
		return m.renderEvidenceBody(current, width)
	default:
		return m.renderSummaryBody(current, width)
	}
}

func (m Model) renderSummaryBody(current Variant, width int) []string {
	lines := []string{
		labeledText("type", current.Type, width, m.theme),
		labeledText("consequence", current.Consequence, width, m.theme),
		labeledText("impact", current.Impact, width, m.theme),
		labeledText("hgvs", coalesce(strings.Join(splitAnnotatedText(current.HGVS), " | "), current.HGVS), width, m.theme),
	}

	evidence := current.Evidence
	if evidence == "" {
		evidence = "No supporting evidence provided."
	}
	lines = append(lines, labeledText("evidence", evidence, width, m.theme))
	lines = append(lines, lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("Enter opens focused detail. Enter again confirms the current variant."))
	return lines
}

func (m Model) renderDetailBody(current Variant, window renderWindow, width int) []string {
	features := overlappingFeatures(m.context.Features, window.visibleStart, window.visibleEnd)
	featureNames := "None"
	if len(features) > 0 {
		names := make([]string, 0, len(features))
		for _, feature := range features {
			names = append(names, fmt.Sprintf("%s (%d-%d)", coalesce(feature.Name, feature.Type), feature.Start, feature.End))
		}
		featureNames = strings.Join(names, "; ")
	}

	lines := []string{
		labeledText("gene", current.Gene, width, m.theme),
		labeledText("position", formatPosition(current.Position), width, m.theme),
		labeledText("ref/alt", fmt.Sprintf("%s -> %s", coalesce(current.Ref, "-"), coalesce(current.Alt, "-")), width, m.theme),
		labeledText("window", fmt.Sprintf("%d-%d (reference anchored at %d)", window.visibleStart, window.visibleEnd, window.referenceStart), width, m.theme),
		labeledText("features", featureNames, width, m.theme),
	}
	return lines
}

func (m Model) renderHGVSBody(current Variant, width int) []string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("HGVS")}
	parts := splitAnnotatedText(current.HGVS)
	if len(parts) == 0 {
		return append(lines, lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("  No HGVS annotation provided."))
	}
	for _, part := range parts {
		lines = append(lines, labeledText("notation", part, width, m.theme))
	}
	return lines
}

func (m Model) renderEvidenceBody(current Variant, width int) []string {
	lines := []string{lipgloss.NewStyle().Bold(true).Foreground(m.theme.Text).Render("Evidence")}
	parts := splitAnnotatedText(current.Evidence)
	if len(parts) == 0 {
		return append(lines, lipgloss.NewStyle().Foreground(m.theme.TextMuted).Render("  No evidence annotations provided."))
	}
	for _, part := range parts {
		lines = append(lines, labeledText("item", part, width, m.theme))
	}
	return lines
}

func (m Model) renderExpandedDetail(current Variant, width int) string {
	body := []string{
		fmt.Sprintf("Focused variant: %s %s", coalesce(current.Gene, "gene?"), coalesce(primaryHGVS(current.HGVS), fmt.Sprintf("%s>%s", current.Ref, current.Alt))),
		fmt.Sprintf("Ref/Alt: %s -> %s", coalesce(current.Ref, "-"), coalesce(current.Alt, "-")),
		fmt.Sprintf("Consequence: %s | Impact: %s", coalesce(current.Consequence, "n/a"), coalesce(current.Impact, "n/a")),
		"Press Enter again to emit crust.SubmitMsg for this variant, or Esc to return to browsing.",
	}
	return m.renderBox("Focused Detail", body, width)
}

func (m Model) renderHelpBox(width int) string {
	body := []string{
		"j/k or up/down : step between variants",
		"h/l or left/right : narrow or widen sequence context",
		"tab : cycle summary, detail, HGVS, and evidence lenses",
		"enter : open focused detail, then confirm the focused variant",
		"esc : close help, leave focused detail, or cancel the overlay",
		"? : toggle this help",
	}
	return m.renderBox("VariantLens Help", body, width)
}

func (m Model) renderBox(title string, body []string, width int) string {
	innerWidth := clampMin(width-4, 20)
	borderStyle := lipgloss.NewStyle().Foreground(m.theme.Border)
	textStyle := lipgloss.NewStyle().Foreground(m.theme.Text)

	titleText := " " + title + " "
	dashes := clampMin(innerWidth+2-len(titleText), 2)
	left := 2
	if left > dashes {
		left = dashes
	}
	right := dashes - left
	top := "+" + strings.Repeat("-", left) + titleText + strings.Repeat("-", right) + "+"

	lines := []string{borderStyle.Render(top)}
	for _, entry := range body {
		for _, wrapped := range wrapWords(entry, innerWidth) {
			line := fitPlain(wrapped, innerWidth)
			lines = append(lines, borderStyle.Render("| ")+textStyle.Render(line)+borderStyle.Render(" |"))
		}
	}
	lines = append(lines, borderStyle.Render("+"+strings.Repeat("-", innerWidth+2)+"+"))
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m Model) buildRenderWindow(current Variant) renderWindow {
	refSeq := m.context.RefSequence
	refStart := m.referenceStart(current)
	if refSeq == "" {
		return renderWindow{
			referenceStart: refStart,
			referenceEnd:   refStart,
			visibleStart:   current.Position,
			visibleEnd:     current.Position,
			markerIndex:    0,
			variantSpan:    maxInt(1, maxInt(len(current.Ref), len(current.Alt))),
		}
	}

	refEnd := refStart + len(refSeq) - 1
	visibleStart := maxInt(refStart, current.Position-m.context.ContextSize)
	visibleEnd := minInt(refEnd, current.Position+m.context.ContextSize+maxInt(len(normalizeAllele(current.Ref))-1, 0))
	if visibleStart > visibleEnd {
		visibleStart = refStart
		visibleEnd = refEnd
	}

	localStart := clampInt(visibleStart-refStart, 0, len(refSeq))
	localEnd := clampInt(visibleEnd-refStart+1, 0, len(refSeq))
	if localEnd < localStart {
		localEnd = localStart
	}

	refSegment := refSeq[localStart:localEnd]
	markerIndex := clampInt(current.Position-visibleStart, 0, len(refSegment))
	refAligned, altAligned, variantSpan := alignAlleles(refSegment, markerIndex, current.Ref, current.Alt)
	codonRef, codonAlt, aaRef, aaAlt := m.codonSummary(current, refStart, refSeq)

	return renderWindow{
		referenceStart: refStart,
		referenceEnd:   refEnd,
		visibleStart:   visibleStart,
		visibleEnd:     visibleEnd,
		markerIndex:    markerIndex,
		variantSpan:    variantSpan,
		refAligned:     refAligned,
		altAligned:     altAligned,
		clippedLeft:    visibleStart > refStart,
		clippedRight:   visibleEnd < refEnd,
		codonRef:       codonRef,
		codonAlt:       codonAlt,
		aaRef:          aaRef,
		aaAlt:          aaAlt,
	}
}

func (m Model) renderGroupedSequence(sequence string, markerIndex, variantSpan int, baseColor color.Color) string {
	style := lipgloss.NewStyle().Foreground(baseColor)
	gapStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted)
	activeStyle := lipgloss.NewStyle().Foreground(baseColor).Background(m.theme.MismatchBg).Bold(true)
	activeGapStyle := lipgloss.NewStyle().Foreground(m.theme.TextMuted).Background(m.theme.MismatchBg).Bold(true)

	if variantSpan <= 0 {
		variantSpan = 1
	}

	var builder strings.Builder
	for i, base := range sequence {
		if i > 0 && i%3 == 0 {
			builder.WriteByte(' ')
		}

		char := string(base)
		currentStyle := style
		if base == '·' {
			currentStyle = gapStyle
		}
		if i >= markerIndex && i < markerIndex+variantSpan {
			currentStyle = activeStyle
			if base == '·' {
				currentStyle = activeGapStyle
			}
		}
		builder.WriteString(currentStyle.Render(char))
	}
	return builder.String()
}

func (m Model) featureBar(feature Feature, start, end, position, width int) string {
	if width <= 0 {
		width = 12
	}
	bar := make([]rune, width)
	for i := range bar {
		bar[i] = '.'
	}

	fill := featureRune(feature.Type)
	if feature.End >= start && feature.Start <= end {
		from := coordToColumn(maxInt(feature.Start, start), start, end, width)
		to := coordToColumn(minInt(feature.End, end)+1, start, end+1, width)
		if to <= from {
			to = from + 1
		}
		if to > width {
			to = width
		}
		for i := from; i < to; i++ {
			bar[i] = fill
		}
	}

	focusColumn := coordToColumn(position, start, end, width)
	if focusColumn >= 0 && focusColumn < width {
		bar[focusColumn] = '|'
	}

	return lipgloss.NewStyle().Foreground(m.featureColor(feature.Type)).Render("[" + string(bar) + "]")
}

func (m Model) codonSummary(current Variant, refStart int, refSeq string) (string, string, string, string) {
	cds, ok := findCDSFeature(m.context.Features, current.Position)
	if !ok || refSeq == "" {
		return "", "", "", ""
	}

	frameOffset := (current.Position - cds.Start) % 3
	codonStart := current.Position - frameOffset
	codonEnd := codonStart + 2
	refEnd := refStart + len(refSeq) - 1
	if codonStart < refStart || codonEnd > refEnd {
		return "", "", "", ""
	}

	localStart := codonStart - refStart
	refCodon := refSeq[localStart : localStart+3]
	altCodon := applyVariantToSequence(refCodon, current, codonStart)
	return refCodon, altCodon, translateCodon(refCodon), translateCodon(altCodon)
}

func (m Model) impactColor(impact string) color.Color {
	switch strings.ToUpper(strings.TrimSpace(impact)) {
	case "HIGH":
		return m.theme.HighImpact
	case "MODERATE":
		return m.theme.ModImpact
	case "LOW":
		return m.theme.LowImpact
	default:
		return m.theme.ModifierImpact
	}
}

func (m Model) featureColor(kind string) color.Color {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "exon":
		return m.theme.FeatureExon
	case "cds":
		return m.theme.FeatureCDS
	case "domain":
		return m.theme.FeatureDomain
	case "motif":
		return m.theme.FeatureMotif
	case "primer":
		return m.theme.FeaturePrimer
	default:
		return m.theme.TextMuted
	}
}

func labeledLine(labelStyle, textStyle lipgloss.Style, label, value string) string {
	return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), textStyle.Render(value))
}

func labeledText(label, text string, width int, theme Theme) string {
	if strings.TrimSpace(text) == "" {
		text = "n/a"
	}
	labelWidth := 12
	contentWidth := clampMin(width-labelWidth, 20)
	lines := wrapWords(text, contentWidth)
	if len(lines) == 0 {
		lines = []string{"n/a"}
	}

	labelStyle := lipgloss.NewStyle().Foreground(theme.TextMuted).Width(labelWidth)
	textStyle := lipgloss.NewStyle().Foreground(theme.Text)
	parts := make([]string, 0, len(lines))
	for i, line := range lines {
		currentLabel := ""
		if i == 0 {
			currentLabel = label + ":"
		}
		parts = append(parts, lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(currentLabel), textStyle.Render(line)))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func alignAlleles(refSegment string, offset int, refAllele, altAllele string) (string, string, int) {
	refAllele = normalizeAllele(refAllele)
	altAllele = normalizeAllele(altAllele)
	offset = clampInt(offset, 0, len(refSegment))

	pre := refSegment[:offset]
	replaceLen := len(refAllele)
	localEnd := minInt(len(refSegment), offset+replaceLen)
	observedRef := refSegment[offset:localEnd]
	post := refSegment[localEnd:]

	span := maxInt(len(observedRef), len(altAllele))
	span = maxInt(span, len(refAllele))
	if span == 0 {
		span = 1
	}

	return pre + padAllele(observedRef, span) + post, pre + padAllele(altAllele, span) + post, span
}

func applyVariantToSequence(sequence string, variant Variant, sequenceStart int) string {
	if sequence == "" {
		return ""
	}

	ref := normalizeAllele(variant.Ref)
	alt := normalizeAllele(variant.Alt)
	localStart := variant.Position - sequenceStart
	if localStart < 0 || localStart > len(sequence) {
		return sequence
	}

	replaceLen := len(ref)
	localEnd := minInt(len(sequence), localStart+replaceLen)
	if replaceLen == 0 {
		return sequence[:localStart] + alt + sequence[localStart:]
	}
	return sequence[:localStart] + alt + sequence[localEnd:]
}

func translateCodon(codon string) string {
	codon = strings.ToUpper(strings.ReplaceAll(codon, "U", "T"))
	if len(codon) != 3 || strings.ContainsRune(codon, '·') {
		return ""
	}
	return codonTable[codon]
}

func primaryHGVS(raw string) string {
	for _, token := range splitAnnotatedText(raw) {
		if strings.HasPrefix(token, "c.") || strings.HasPrefix(token, "g.") || strings.HasPrefix(token, "n.") {
			return token
		}
	}
	if tokens := splitAnnotatedText(raw); len(tokens) > 0 {
		return tokens[0]
	}
	return strings.TrimSpace(raw)
}

func proteinHGVS(raw string) string {
	for _, token := range splitAnnotatedText(raw) {
		if strings.HasPrefix(token, "p.") {
			return token
		}
	}
	return ""
}

func wrapWords(text string, width int) []string {
	width = clampMin(width, 8)
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := lines[len(lines)-1]
		if len(last)+1+len(word) <= width {
			lines[len(lines)-1] = last + " " + word
			continue
		}
		lines = append(lines, word)
	}
	return lines
}

func compactStrings(lines []string) []string {
	compacted := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" {
			continue
		}
		compacted = append(compacted, line)
	}
	return compacted
}

func fitPlain(text string, width int) string {
	text = truncatePlain(text, width)
	if len(text) < width {
		text += strings.Repeat(" ", width-len(text))
	}
	return text
}

func truncatePlain(text string, width int) string {
	if width <= 0 || len(text) <= width {
		return text
	}
	if width <= 3 {
		return text[:width]
	}
	return text[:width-3] + "..."
}

func padAllele(sequence string, width int) string {
	if len(sequence) >= width {
		return sequence
	}
	return sequence + strings.Repeat("·", width-len(sequence))
}

func groupedIndex(raw int) int {
	if raw <= 0 {
		return 0
	}
	return raw + raw/3
}

func clipAndPad(text string, width int) string {
	text = truncatePlain(text, width)
	if len(text) < width {
		text += strings.Repeat(" ", width-len(text))
	}
	return text
}

func coordToColumn(position, start, end, width int) int {
	if width <= 1 || end <= start {
		return 0
	}
	if position <= start {
		return 0
	}
	if position >= end {
		return width - 1
	}
	return (position - start) * width / (end - start)
}

func featureRune(kind string) rune {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "cds":
		return '#'
	case "domain":
		return '~'
	case "motif":
		return ':'
	case "primer":
		return '+'
	default:
		return '='
	}
}

func coalesce(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

var codonTable = map[string]string{
	"TTT": "Phe", "TTC": "Phe", "TTA": "Leu", "TTG": "Leu",
	"TCT": "Ser", "TCC": "Ser", "TCA": "Ser", "TCG": "Ser",
	"TAT": "Tyr", "TAC": "Tyr", "TAA": "Stop", "TAG": "Stop",
	"TGT": "Cys", "TGC": "Cys", "TGA": "Stop", "TGG": "Trp",
	"CTT": "Leu", "CTC": "Leu", "CTA": "Leu", "CTG": "Leu",
	"CCT": "Pro", "CCC": "Pro", "CCA": "Pro", "CCG": "Pro",
	"CAT": "His", "CAC": "His", "CAA": "Gln", "CAG": "Gln",
	"CGT": "Arg", "CGC": "Arg", "CGA": "Arg", "CGG": "Arg",
	"ATT": "Ile", "ATC": "Ile", "ATA": "Ile", "ATG": "Met",
	"ACT": "Thr", "ACC": "Thr", "ACA": "Thr", "ACG": "Thr",
	"AAT": "Asn", "AAC": "Asn", "AAA": "Lys", "AAG": "Lys",
	"AGT": "Ser", "AGC": "Ser", "AGA": "Arg", "AGG": "Arg",
	"GTT": "Val", "GTC": "Val", "GTA": "Val", "GTG": "Val",
	"GCT": "Ala", "GCC": "Ala", "GCA": "Ala", "GCG": "Ala",
	"GAT": "Asp", "GAC": "Asp", "GAA": "Glu", "GAG": "Glu",
	"GGT": "Gly", "GGC": "Gly", "GGA": "Gly", "GGG": "Gly",
}
