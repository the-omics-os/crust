// Package thresholdslider provides an interactive numeric threshold adjustment
// component for BubbleTea applications.
//
// Users adjust a value with arrow keys. On Enter, it returns a crust.SubmitMsg
// via tea.Cmd. On Esc, it returns a crust.CancelMsg. Value changes emit a
// ValueChangedMsg for live preview updates.
package thresholdslider

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

// ValueChangedMsg is emitted when the slider value changes during interaction.
// Host applications can listen for this to update live previews.
type ValueChangedMsg struct {
	Value float64
}

// Model is the BubbleTea model for the threshold slider.
type Model struct {
	label     string
	min       float64
	max       float64
	step      float64
	value     float64
	unit      string
	count     int
	total     int
	width     int
	precision int
	theme     Theme
}

// New creates a ThresholdSlider with the given options.
func New(opts ...Option) Model {
	m := Model{
		label: "Threshold",
		min:   0,
		max:   1,
		step:  0.01,
		value: 0.5,
		width: 60,
		theme: DefaultTheme(),
	}
	for _, opt := range opts {
		opt(&m)
	}
	// Clamp value to range.
	m.value = clamp(m.value, m.min, m.max)
	// Derive precision from step.
	m.precision = decimalPlaces(m.step)
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

	switch km.String() {
	case "right":
		old := m.value
		m.value = clamp(m.value+m.step, m.min, m.max)
		if m.value != old {
			return m, valueChangedCmd(m.value, m.precision)
		}
		return m, nil
	case "left":
		old := m.value
		m.value = clamp(m.value-m.step, m.min, m.max)
		if m.value != old {
			return m, valueChangedCmd(m.value, m.precision)
		}
		return m, nil
	case "shift+right":
		old := m.value
		m.value = clamp(m.value+10*m.step, m.min, m.max)
		if m.value != old {
			return m, valueChangedCmd(m.value, m.precision)
		}
		return m, nil
	case "shift+left":
		old := m.value
		m.value = clamp(m.value-10*m.step, m.min, m.max)
		if m.value != old {
			return m, valueChangedCmd(m.value, m.precision)
		}
		return m, nil
	case "enter":
		return m, func() tea.Msg {
			return crust.SubmitMsg{
				Component: "threshold_slider",
				Data:      map[string]any{"value": roundTo(m.value, m.precision)},
			}
		}
	case "esc":
		return m, func() tea.Msg {
			return crust.CancelMsg{
				Component: "threshold_slider",
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

// Render returns the slider as a plain string (useful for embedding).
func (m Model) Render() string {
	return m.render()
}

func (m Model) render() string {
	w := m.width
	if w < 20 {
		w = 20
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Title).Width(w)
	valueStyle := lipgloss.NewStyle().Bold(true).Foreground(m.theme.Value)
	countStyle := lipgloss.NewStyle().Foreground(m.theme.Count)
	filledStyle := lipgloss.NewStyle().Foreground(m.theme.Filled)
	emptyStyle := lipgloss.NewStyle().Foreground(m.theme.Empty)
	cursorStyle := lipgloss.NewStyle().Foreground(m.theme.Cursor).Bold(true)

	// Value string.
	valStr := fmt.Sprintf("%.*f", m.precision, m.value)
	if m.unit != "" {
		valStr += " " + m.unit
	}
	renderedVal := valueStyle.Render(valStr)

	// Slider bar.
	valDisplayWidth := lipgloss.Width(renderedVal)
	barOverhead := 6 + valDisplayWidth
	barWidth := w - barOverhead
	if barWidth < 10 {
		barWidth = 10
	}

	fillRatio := 0.0
	rangeVal := m.max - m.min
	if rangeVal > 0 {
		fillRatio = (m.value - m.min) / rangeVal
	}
	fillRatio = clamp(fillRatio, 0, 1)

	cursorPos := int(math.Round(fillRatio * float64(barWidth-1)))
	if cursorPos < 0 {
		cursorPos = 0
	}
	if cursorPos >= barWidth {
		cursorPos = barWidth - 1
	}

	var bar strings.Builder
	for i := 0; i < barWidth; i++ {
		if i == cursorPos {
			bar.WriteString(cursorStyle.Render("|"))
		} else if i < cursorPos {
			bar.WriteString(filledStyle.Render("="))
		} else {
			bar.WriteString(emptyStyle.Render("-"))
		}
	}

	sliderLine := fmt.Sprintf("  [%s]  %s", bar.String(), renderedVal)

	// Count line.
	var countLine string
	if m.total > 0 {
		pct := float64(m.count) / float64(m.total) * 100
		countLine = countStyle.Render(fmt.Sprintf(
			"%d / %d items passing (%.1f%%)",
			m.count, m.total, pct,
		))
	}

	title := titleStyle.Render(m.label)

	parts := []string{"", title, "", sliderLine, ""}
	if countLine != "" {
		parts = append(parts, countLine, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// Value returns the current slider value.
func (m Model) Value() float64 { return roundTo(m.value, m.precision) }

// SetCount updates the count/total display.
func (m *Model) SetCount(count, total int) {
	m.count = count
	m.total = total
}

// SetWidth updates the rendering width.
func (m *Model) SetWidth(w int) { m.width = w }

func valueChangedCmd(value float64, precision int) tea.Cmd {
	return func() tea.Msg {
		return ValueChangedMsg{Value: roundTo(value, precision)}
	}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func decimalPlaces(step float64) int {
	if step >= 1 {
		return 0
	}
	s := fmt.Sprintf("%g", step)
	// Handle scientific notation: 1e-06 means 6 decimal places.
	if eIdx := strings.Index(s, "e-"); eIdx >= 0 {
		exp, err := strconv.Atoi(s[eIdx+2:])
		if err == nil {
			mantissa := s[:eIdx]
			dotIdx := strings.IndexByte(mantissa, '.')
			mantissaDecimals := 0
			if dotIdx >= 0 {
				mantissaDecimals = len(mantissa) - dotIdx - 1
			}
			return exp + mantissaDecimals
		}
	}
	idx := strings.IndexByte(s, '.')
	if idx < 0 {
		return 0
	}
	return len(s) - idx - 1
}

func roundTo(v float64, places int) float64 {
	pow := math.Pow(10, float64(places))
	return math.Round(v*pow) / pow
}
