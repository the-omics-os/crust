package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust"
	"github.com/the-omics-os/crust/variantlens"
)

type demoModel struct {
	lens   variantlens.Model
	done   bool
	result string
}

func (m demoModel) Init() tea.Cmd {
	return m.lens.Init()
}

func (m demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.lens.SetWidth(msg.Width - 4)
	case tea.KeyPressMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.result = "quit"
			m.done = true
			return m, tea.Quit
		}
	case crust.SubmitMsg:
		m.result = fmt.Sprintf("submitted %v", msg.Data["variant"])
		m.done = true
		return m, tea.Quit
	case crust.CancelMsg:
		m.result = "cancelled"
		m.done = true
		return m, tea.Quit
	}

	updated, cmd := m.lens.Update(msg)
	m.lens = updated.(variantlens.Model)
	return m, cmd
}

func (m demoModel) View() tea.View {
	if m.done {
		return tea.NewView(m.result + "\n")
	}
	width := m.lens.Width()
	return tea.NewView(
		m.lens.Render() +
			"\n\n" +
			renderDemoHint(width) + "\n",
	)
}

func renderDemoHint(width int) string {
	lines := wrapDemoText("Demo only: q quits.", maxInt(width, 16))
	return strings.Join(lines, "\n")
}

func wrapDemoText(text string, width int) []string {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	ctx := variantlens.VariantContext{
		RefSequence:    "GATTGCGATCCTGAAACCTTGACCGTTA",
		ReferenceStart: 178,
		ContextSize:    6,
		Variants: []variantlens.Variant{
			{
				Position:    181,
				Ref:         "T",
				Alt:         "G",
				Type:        "SNV",
				Consequence: "missense",
				HGVS:        "c.181T>G | p.Cys61Gly",
				Gene:        "BRCA1",
				Impact:      "HIGH",
				Evidence:    "ClinVar: Pathogenic | gnomAD: 0.00002 | SIFT: Deleterious",
			},
			{
				Position:    186,
				Ref:         "T",
				Alt:         "C",
				Type:        "SNV",
				Consequence: "missense",
				HGVS:        "c.186T>C | p.Asp62Asn",
				Gene:        "BRCA1",
				Impact:      "MODERATE",
				Evidence:    "ClinVar: Uncertain significance | gnomAD: 0.0011 | PolyPhen: Possibly damaging",
			},
			{
				Position:    191,
				Ref:         "A",
				Alt:         "ATG",
				Type:        "insertion",
				Consequence: "frameshift",
				HGVS:        "c.191dupTG | p.Leu64fs",
				Gene:        "BRCA1",
				Impact:      "HIGH",
				Evidence:    "ClinVar: Likely pathogenic | cohort: 3/812",
			},
		},
		Features: []variantlens.Feature{
			{Name: "Exon 5", Type: "exon", Start: 178, End: 205},
			{Name: "CDS 5", Type: "CDS", Start: 178, End: 205},
			{Name: "BRCT domain", Type: "domain", Start: 182, End: 196},
			{Name: "Primer A", Type: "primer", Start: 189, End: 201},
			{Name: "Splice motif", Type: "motif", Start: 196, End: 199},
		},
	}

	lens := variantlens.New(
		variantlens.WithContext(ctx),
		variantlens.WithWidth(100),
	)

	program := tea.NewProgram(demoModel{lens: lens})
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
