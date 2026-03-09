# CLAUDE.md — Crust

**Crust** — Life sciences TUI components for Bubble Tea.

A standalone Go component library providing terminal UI building blocks for biology, chemistry, and life sciences applications. Built natively on Charm v2 (BubbleTea v2, Bubbles v2, Lipgloss v2). Part of the Omics-OS ecosystem, with Lobster AI as the first consumer.

---

## Identity

| Field | Value |
|-------|-------|
| Name | **Crust** |
| Tagline | Life sciences components for Bubble Tea |
| Go module | `github.com/the-omics-os/crust` |
| Audience | Developers building scientific TUI applications |
| Relationship to Lobster | Lobster's `lobster-tui/internal/biocomp/` wraps Crust components into its protocol-driven adapter layer |
| License | TBD (Apache 2.0 or MIT — match Charm ecosystem) |

**Naming rationale:** Lobster crust (exoskeleton), short, Charm-idiomatic (Bubble Tea, Lip Gloss, Glamour, Wish...), terminal pun (shell/crust).

---

## Design Principles

1. **Pattern A — Standalone `tea.Model` implementations.** Every component is a normal BubbleTea v2 model with typed Go constructors. No JSON `Init()`, no protocol coupling. Any BubbleTea app can embed Crust components directly.

2. **Charm v2 native from day one.** Use `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`. Do not support v1. Follow v2 patterns: `tea.View` return type, `tea.KeyPressMsg`, `color.Color` interface.

3. **Build on Bubbles, don't replace them.** Crust sits on top of the Charm stack. Use `bubbles/viewport` for scrolling, `bubbles/textinput` for text fields, `bubbles/key` for bindings. Never reimplement what Bubbles already provides.

4. **Dynamic theming.** Components accept a style/theme configuration at construction time. No hardcoded colors. Ship sensible defaults but allow full customization through a `Theme` struct.

5. **Composable, not monolithic.** Each component is independently importable. Users pick what they need: `import "github.com/the-omics-os/crust/sequenceviewer"`. No god package.

6. **Domain-correct.** Color choices, labels, and defaults must be scientifically meaningful: codon coloring follows IUPAC conventions, amino acid colors follow standard schemes (e.g., Clustal, Taylor), QC thresholds use field-accepted ranges.

---

## What Crust is NOT

- Not a replacement for Bubbles (text input, list, table, spinner, progress bar — these are solved)
- Not a protocol layer (that's Lobster's `biocomp/` adapter)
- Not a CLI framework (that's BubbleTea itself)
- Not a plotting library (terminal scatter/volcano plots always look bad — generate images instead)

---

## v0.1 Component Inventory

Five components proving the concept across different interaction modes:

| Component | Package | Mode | Interactive | Streaming | Description |
|-----------|---------|------|-------------|-----------|-------------|
| **SequenceViewer** | `sequenceviewer` | inline | read-only (scrollable) | no | DNA/RNA/protein sequence display with codon coloring, reading frames, complement strand. The flagship. |
| **OntologyBrowser** | `ontologybrowser` | overlay | yes (navigate, expand, select) | yes (lazy-load children) | Tree navigation for GO/Reactome/Disease Ontology/ChEBI. Expandable nodes, search/filter. |
| **QCDashboard** | `qcdashboard` | inline | no | yes (metrics update) | Multi-metric quality panel with colored status bars and pass/warn/fail thresholds. |
| **ThresholdSlider** | `thresholdslider` | overlay | yes (adjust, submit) | yes (live count feedback) | Numeric threshold adjustment with visual bar and item count preview. |
| **PeriodicTable** | `periodictable` | overlay | yes (navigate, select) | no | Interactive periodic table. Element selection, group/period highlighting, property display. |

### Future candidates (post v0.1, not yet scoped)

- **AlignmentViewer** — Pairwise/MSA with conservation coloring, consensus line
- **MoleculeRenderer** — 2D molecular structure from SMILES (Unicode/ASCII)
- **DomainDiagram** — Protein domain layout (Pfam-style linear)
- **PhyloTree** — Phylogenetic tree from Newick format
- **Heatmap** — Colored grid with row/column labels (Unicode block characters)
- **SequenceInput** — Validated DNA/RNA/protein entry with complement preview
- **GenomeBrowser** — Simplified chromosome ideogram + annotation tracks

---

## Component Architecture

### Standard Component Shape

Every Crust component follows this pattern:

```go
package sequenceviewer

import tea "charm.land/bubbletea/v2"

// Model is the BubbleTea model for the sequence viewer.
type Model struct {
    // Typed fields — no json.RawMessage
    sequence string
    seqType  SequenceType // DNA, RNA, Protein
    theme    Theme
    width    int
    height   int
    // ... internal state
}

// Option is a functional option for constructing a Model.
type Option func(*Model)

// New creates a SequenceViewer with the given options.
func New(opts ...Option) Model {
    m := Model{
        theme: DefaultTheme(),
    }
    for _, opt := range opts {
        opt(&m)
    }
    return m
}

// Functional options
func WithSequence(seq string, t SequenceType) Option { ... }
func WithTheme(t Theme) Option { ... }
func WithWidth(w int) Option { ... }

// BubbleTea interface
func (m Model) Init() tea.Cmd          { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m Model) View() string           { ... }

// Public API for programmatic updates
func (m *Model) SetSequence(seq string, t SequenceType) { ... }
func (m Model) Sequence() string       { ... }
```

**Key conventions:**
- Functional options pattern (`New(opts...)`) — matches Bubbles v2 style (`viewport.New(viewport.WithWidth(80))`)
- Value receiver for `View()`, pointer receiver for mutations
- Public setters/getters for programmatic updates
- `Theme` struct accepted via `WithTheme()` option
- No `Init(json.RawMessage)` — that's Lobster's adapter concern

### Theme System

```go
// theme.go in each component package, or a shared crust/theme package

type Theme struct {
    // Base colors
    Primary    color.Color
    Secondary  color.Color
    Background color.Color
    Text       color.Color
    TextMuted  color.Color

    // Semantic colors
    Pass    color.Color // green — QC pass, valid sequence
    Warn    color.Color // yellow — QC warning, unusual value
    Fail    color.Color // red — QC fail, invalid input
    Info    color.Color // blue — informational

    // Domain-specific (components pick what they need)
    Adenine  color.Color // A — typically green
    Thymine  color.Color // T — typically red
    Guanine  color.Color // G — typically yellow/orange
    Cytosine color.Color // C — typically blue
    Uracil   color.Color // U — typically red (RNA)
}
```

Whether this is one shared `crust/theme` package or per-component themes is a v0.1 design decision. Start with per-component, extract shared theme if patterns converge.

---

## Relationship to Lobster

Crust is a **library**. Lobster is a **consumer**.

```
crust/                          # This repo — standalone components
  sequenceviewer/
  ontologybrowser/
  qcdashboard/
  thresholdslider/
  periodictable/

lobster-tui/internal/biocomp/   # Lobster's adapter layer (separate repo)
  component.go                  # BioComponent interface (protocol-driven)
  registry.go                   # Factory registry
  overlay.go                    # Overlay frame renderer
  celltype/                     # Lobster-specific: wraps crust components
  threshold/                    #   into BioComponent interface with
  qcdash/                       #   JSON Init(), protocol lifecycle
```

The adapter layer in `lobster-tui/internal/biocomp/` handles:
- JSON deserialization (`Init(json.RawMessage)` -> typed constructor)
- Protocol lifecycle (component_render, component_response, component_close)
- Error boundaries (recover from panics, send error responses)
- Overlay framing (border, title bar, help bar)

Crust components know nothing about JSON protocols or IPC.

---

## Directory Structure (Target)

```
crust/
  go.mod                        # module github.com/the-omics-os/crust
  go.sum
  CLAUDE.md                     # This file
  README.md                     # Public-facing docs
  LICENSE
  theme/                        # Shared theme types (if extracted)
    theme.go
    defaults.go
  sequenceviewer/
    model.go                    # tea.Model implementation
    options.go                  # Functional options
    theme.go                    # Component-specific theme/colors
    coloring.go                 # IUPAC codon coloring, amino acid schemes
    sequenceviewer_test.go
  ontologybrowser/
    model.go
    options.go
    node.go                     # OntologyNode type, tree operations
    ontologybrowser_test.go
  qcdashboard/
    model.go
    options.go
    metric.go                   # Metric type, status evaluation
    qcdashboard_test.go
  thresholdslider/
    model.go
    options.go
    thresholdslider_test.go
  periodictable/
    model.go
    options.go
    elements.go                 # Element data (118 elements, groups, periods)
    periodictable_test.go
  examples/                     # Runnable BubbleTea demo apps
    sequence/main.go
    ontology/main.go
    qc/main.go
    periodic/main.go
```

---

## Development Rules

1. **Go 1.24+** — match Lobster TUI
2. **Charm v2 only** — `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`
3. **No external dependencies beyond Charm** — keep the dependency tree minimal
4. **Every component has tests** — unit tests for Init/Update/View, visual snapshot tests if feasible
5. **Every component has an example** — runnable `main.go` in `examples/` that demonstrates the component standalone
6. **No Lobster imports** — Crust must not import anything from `lobster-tui` or `lobster`
7. **Domain correctness over aesthetics** — if IUPAC says adenine is green, it's green. Scientific conventions take precedence over "what looks cool"

---

## Commands

```bash
# Setup
go mod tidy

# Test
go test ./...

# Run examples
go run ./examples/sequence/
go run ./examples/periodic/

# Build check
go vet ./...
```

---

## Current Status

**Phase: Design & Scaffolding**

- [x] Project identity and naming (Crust)
- [x] v0.1 component inventory (5 components)
- [x] Architecture decisions (Pattern A, Charm v2, functional options)
- [ ] Initialize Go module
- [ ] Scaffold directory structure
- [ ] Port QCDashboard from lobster-tui biocomp (closest to done)
- [ ] Port ThresholdSlider from lobster-tui biocomp
- [ ] Build SequenceViewer (flagship)
- [ ] Build PeriodicTable
- [ ] Build OntologyBrowser
- [ ] Create example apps
- [ ] README with screenshots/GIFs
