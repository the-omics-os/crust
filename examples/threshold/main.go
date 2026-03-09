// Example: interactive threshold slider.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/the-omics-os/crust"
	"github.com/the-omics-os/crust/thresholdslider"
)

type model struct {
	slider thresholdslider.Model
	done   bool
	result string
}

func (m model) Init() tea.Cmd { return m.slider.Init() }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case crust.SubmitMsg:
		m.done = true
		m.result = fmt.Sprintf("Submitted: %v", msg.Data["value"])
		return m, tea.Quit
	case crust.CancelMsg:
		m.done = true
		m.result = "Cancelled"
		return m, tea.Quit
	case thresholdslider.ValueChangedMsg:
		// Live preview: update count based on value.
		// In a real app, you'd recalculate based on data.
		pct := msg.Value
		m.slider.SetCount(int(pct*20000), 20000)
	}

	var cmd tea.Cmd
	var updated tea.Model
	updated, cmd = m.slider.Update(msg)
	m.slider = updated.(thresholdslider.Model)
	return m, cmd
}

func (m model) View() tea.View {
	if m.done {
		return tea.NewView(m.result + "\n")
	}
	return tea.NewView(m.slider.Render() + "\n\n  Press q to quit\n")
}

func main() {
	slider := thresholdslider.New(
		thresholdslider.WithLabel("p-value cutoff"),
		thresholdslider.WithRange(0, 1),
		thresholdslider.WithStep(0.01),
		thresholdslider.WithDefault(0.05),
		thresholdslider.WithUnit(""),
		thresholdslider.WithCount(1542, 20000),
		thresholdslider.WithWidth(60),
	)

	p := tea.NewProgram(model{slider: slider})
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
