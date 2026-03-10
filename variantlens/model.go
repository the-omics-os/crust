// Package variantlens provides a multi-layer variant consequence inspector for
// Bubble Tea applications.
//
// VariantLens is an interactive overlay component for stepping through
// annotated variants while keeping local reference, altered sequence, codon,
// amino-acid, and feature context aligned in one terminal-native view.
package variantlens

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
)

const (
	defaultWidth       = 96
	minWidth           = 56
	defaultContextSize = 12
	minContextSize     = 3
	contextStep        = 3
)

// VariantChangedMsg is emitted when the focused variant changes.
type VariantChangedMsg struct {
	Index   int
	Variant Variant
}

// ContextSizeChangedMsg is emitted when the sequence window changes.
type ContextSizeChangedMsg struct {
	ContextSize int
}

// ViewModeChangedMsg is emitted when the body view changes.
type ViewModeChangedMsg struct {
	Mode ViewMode
}

// DetailToggledMsg is emitted when expanded detail mode opens or closes.
type DetailToggledMsg struct {
	Expanded bool
	Variant  Variant
}

// Model is the Bubble Tea model for VariantLens.
type Model struct {
	context  VariantContext
	width    int
	theme    Theme
	selected int
	viewMode ViewMode
	detail   bool
	showHelp bool
}

// New creates a VariantLens with the given options.
func New(opts ...Option) Model {
	m := Model{
		context: VariantContext{
			ContextSize: defaultContextSize,
		},
		width:    defaultWidth,
		theme:    DefaultTheme(),
		viewMode: ViewSummary,
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
	km, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	key := km.String()
	if m.showHelp {
		switch key {
		case "?", "esc":
			m.showHelp = false
		}
		return m, nil
	}

	switch key {
	case "j", "down":
		if len(m.context.Variants) == 0 || m.selected >= len(m.context.Variants)-1 {
			return m, nil
		}
		m.selected++
		return m, variantChangedCmd(m.selected, m.context.Variants[m.selected])
	case "k", "up":
		if len(m.context.Variants) == 0 || m.selected <= 0 {
			return m, nil
		}
		m.selected--
		return m, variantChangedCmd(m.selected, m.context.Variants[m.selected])
	case "l", "right":
		old := m.context.ContextSize
		m.context.ContextSize = clampContextSize(m.context.ContextSize+contextStep, len(m.context.RefSequence))
		if m.context.ContextSize == old {
			return m, nil
		}
		return m, contextSizeChangedCmd(m.context.ContextSize)
	case "h", "left":
		old := m.context.ContextSize
		m.context.ContextSize = clampContextSize(m.context.ContextSize-contextStep, len(m.context.RefSequence))
		if m.context.ContextSize == old {
			return m, nil
		}
		return m, contextSizeChangedCmd(m.context.ContextSize)
	case "tab":
		m.viewMode = m.viewMode.next()
		return m, viewModeChangedCmd(m.viewMode)
	case "?":
		m.showHelp = !m.showHelp
		return m, nil
	case "enter":
		if len(m.context.Variants) == 0 {
			return m, nil
		}
		if !m.detail {
			m.detail = true
			return m, detailToggledCmd(true, m.context.Variants[m.selected])
		}
		return m, func() tea.Msg {
			return crust.SubmitMsg{
				Component: "variant_lens",
				Data: map[string]any{
					"context_size": m.context.ContextSize,
					"index":        m.selected,
					"variant":      m.context.Variants[m.selected],
					"view_mode":    m.viewMode.String(),
				},
			}
		}
	case "esc":
		if m.detail {
			m.detail = false
			return m, nil
		}
		return m, func() tea.Msg {
			return crust.CancelMsg{
				Component: "variant_lens",
				Reason:    "user cancelled",
			}
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.render())
}

// Render returns the lens as a plain string for embedding or tests.
func (m Model) Render() string {
	return m.render()
}

// Context returns a defensive copy of the current context.
func (m Model) Context() VariantContext {
	return cloneContext(m.context)
}

// SelectedIndex returns the focused variant index.
func (m Model) SelectedIndex() int {
	return m.selected
}

// SelectedVariant returns the focused variant, if present.
func (m Model) SelectedVariant() (Variant, bool) {
	if len(m.context.Variants) == 0 {
		return Variant{}, false
	}
	return m.context.Variants[m.selected], true
}

// ViewMode returns the active view mode.
func (m Model) ViewMode() ViewMode {
	return m.viewMode
}

// DetailMode reports whether expanded detail is visible.
func (m Model) DetailMode() bool {
	return m.detail
}

// HelpVisible reports whether help is visible.
func (m Model) HelpVisible() bool {
	return m.showHelp
}

// Width returns the current rendering width.
func (m Model) Width() int {
	return m.width
}

// ContextSize returns the current upstream/downstream window size.
func (m Model) ContextSize() int {
	return m.context.ContextSize
}

// SetContext replaces the full context using a defensive copy.
func (m *Model) SetContext(ctx VariantContext) {
	m.context = cloneContext(ctx)
	m.normalize()
}

// SetVariants replaces the variant list using a defensive copy.
func (m *Model) SetVariants(variants []Variant) {
	m.context.Variants = cloneVariants(variants)
	m.normalize()
}

// SetFeatures replaces the feature list using a defensive copy.
func (m *Model) SetFeatures(features []Feature) {
	m.context.Features = cloneFeatures(features)
	m.normalize()
}

// SetReferenceSequence replaces the reference sequence window.
func (m *Model) SetReferenceSequence(seq string) {
	m.context.RefSequence = normalizeSequence(seq)
	m.normalize()
}

// SetReferenceStart anchors the reference sequence to a stable coordinate.
func (m *Model) SetReferenceStart(start int) {
	m.context.ReferenceStart = start
	m.normalize()
}

// SetContextSize updates the visible window around the focused variant.
func (m *Model) SetContextSize(size int) {
	m.context.ContextSize = clampContextSize(size, len(m.context.RefSequence))
}

// SetSelectedVariant changes the focused variant index.
func (m *Model) SetSelectedVariant(index int) {
	m.selected = clampIndex(index, len(m.context.Variants))
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(w int) {
	m.width = clampMin(w, minWidth)
}

func (m *Model) normalize() {
	m.context = cloneContext(m.context)
	if m.context.ContextSize <= 0 {
		m.context.ContextSize = defaultContextSize
	}
	m.context.ContextSize = clampContextSize(m.context.ContextSize, len(m.context.RefSequence))
	m.width = clampMin(m.width, minWidth)
	m.selected = clampIndex(m.selected, len(m.context.Variants))
	if m.viewMode == "" {
		m.viewMode = ViewSummary
	}
}

func (m Model) referenceStart(current Variant) int {
	if m.context.ReferenceStart > 0 {
		return m.context.ReferenceStart
	}
	if current.Position > 0 {
		inferred := current.Position - m.context.ContextSize
		if inferred < 1 {
			return 1
		}
		return inferred
	}
	return 1
}

func variantChangedCmd(index int, variant Variant) tea.Cmd {
	return func() tea.Msg {
		return VariantChangedMsg{Index: index, Variant: variant}
	}
}

func contextSizeChangedCmd(size int) tea.Cmd {
	return func() tea.Msg {
		return ContextSizeChangedMsg{ContextSize: size}
	}
}

func viewModeChangedCmd(mode ViewMode) tea.Cmd {
	return func() tea.Msg {
		return ViewModeChangedMsg{Mode: mode}
	}
}

func detailToggledCmd(expanded bool, variant Variant) tea.Cmd {
	return func() tea.Msg {
		return DetailToggledMsg{Expanded: expanded, Variant: variant}
	}
}

func clampContextSize(size, sequenceLen int) int {
	size = clampMin(size, minContextSize)
	if sequenceLen == 0 {
		return size
	}
	return clampInt(size, minContextSize, clampMin(sequenceLen, minContextSize))
}

func clampIndex(index, length int) int {
	if length <= 0 {
		return 0
	}
	return clampInt(index, 0, length-1)
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

func clampMin(value, minValue int) int {
	if value < minValue {
		return minValue
	}
	return value
}

func maxInt(a, b int) int {
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

func formatPosition(position int) string {
	if position <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%d", position)
}
