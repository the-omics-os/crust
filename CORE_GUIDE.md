# Crust — Core Guide

**Crust** — Life sciences components for Bubble Tea.

A standalone Go component library for **structured biological inspection and construction** in the terminal. Sequences, molecules, plates, tracks, trees, maps, and workbenches — rendered natively on Charm v2.

CRITICAL RULES: 
1. don't create branches, you work in your own folder
2. Follow the key design principles to ensure full integration
3. Build only what is asked for (no crazy applications) and built it robust, professional, modular and dynamic
---

## Identity

| Field | Value |
|-------|-------|
| Name | **Crust** |
| Tagline | Life sciences components for Bubble Tea |
| Module | `github.com/the-omics-os/crust` |
| License | MIT |
| Go | 1.25+ |
| Charm | v2 only (`charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`) |
| First consumer | Lobster AI (`lobster-tui/internal/biocomp/` adapter layer) |

**What Crust is:** A library of components for structured biological inspection and construction.

**What Crust is NOT:** A Bubbles replacement, a protocol layer, a CLI framework, or a plotting library.

---

## Design Principles

1. **Standalone `tea.Model`.** Every component is a normal BubbleTea v2 model. Typed Go constructors, functional options. No JSON, no protocol coupling.
2. **Charm v2 native.** `tea.View` return type, `tea.KeyPressMsg`, `color.Color` interface. No v1 support.
3. **Build on Bubbles.** Use `bubbles/viewport` for scrolling, `bubbles/textinput` for text, `bubbles/key` for bindings. Never reimplement solved problems.
4. **Dynamic theming.** Per-component `Theme` struct with `DefaultTheme()` factory and `WithTheme()` option. No hardcoded colors.
5. **Composable.** Each component independently importable: `import "github.com/the-omics-os/crust/sequenceviewer"`. No god package.
6. **Domain-correct.** Scientific conventions over aesthetics. IUPAC nucleotide colors, Clustal amino acid schemes, accepted QC thresholds.
7. **Compute-capable.** Components carry rich data, not just characters. Residues have chemophysical properties. Analysis runs natively in Go.
8. **Pure MIT.** No GPL or LGPL dependencies. Reimplement algorithms from reference sources.

---

## Universal Interaction Semantics

Every Crust component follows these keybinding conventions:

| Key | Action | Scope |
|-----|--------|-------|
| Arrow keys | Navigate within the current plane (scroll, move cursor, adjust value) | Always |
| `Tab` | Switch planes: cycle view modes, switch between panes, change data layer | Always |
| `Esc` | Exit the current interaction level (close overlay, leave edit mode, dismiss) | Always |
| `Enter` | Confirm / submit / expand / activate the focused item | Interactive components |
| `?` | Toggle help overlay showing component-specific keybindings | All components |

**Planes:** A "plane" is a logical view layer. Examples: Identity/Hydrophobicity/Charge views in SequenceViewer, raw/normalized/hit modes in PlateMap, construct/protein/assembly lenses in VectorWorkbench. Tab always cycles planes. Arrows always navigate within the current plane.

---

## Protocol & Streaming Semantics

Crust components are standalone, but their design must be **protocol-ready** — compatible with the Lobster adapter pattern without requiring changes.

### Message Signaling

Components signal completion via `tea.Cmd` returning typed messages:

```go
// Root package: crust/result.go
type SubmitMsg struct {
    Component string         // registry key
    Data      map[string]any // response payload
}

type CancelMsg struct {
    Component string
    Reason    string
}
```

- Interactive components return `SubmitMsg` on Enter (final confirmation) and `CancelMsg` on Esc.
- Non-interactive components (QCDashboard, inline SequenceViewer) never return these.
- Component-specific intermediate messages (e.g., `ValueChangedMsg`) are defined per-component.

### Streaming Updates

Components that accept live data updates expose typed `Set*()` mutators:

```go
// QCDashboard
func (m *Model) SetMetrics(metrics []Metric)

// ThresholdSlider
func (m *Model) SetCount(count, total int)

// SequenceViewer
func (m *Model) SetSequence(seq string, t SequenceType)
```

The Lobster adapter maps `component_render` with same ID → `SetData(json)` → deserialize → call typed setter. This is the established streaming protocol from `charm-tui-protocol.md`.

### Adapter Contract

When Lobster wraps a Crust component, the adapter handles:

| Concern | Adapter (biocomp/) | Crust component |
|---------|-------------------|-----------------|
| JSON deserialization | Yes — `Init(json.RawMessage)` | No — typed constructors |
| Protocol lifecycle | Yes — component_render/response/close | No — pure tea.Model |
| Error boundaries | Yes — panic recovery | No — let it panic |
| Overlay framing | Yes — border, title, help bar | No — renders content only |
| Theme bridging | Yes — maps Lobster theme → Crust theme | Accepts theme via option |
| `tea.Cmd` propagation | Yes — returns cmd from `HandleMsg` | Returns cmd from `Update` |

---

## Standard Component Shape

```go
package mycomponent

import tea "charm.land/bubbletea/v2"

type Option func(*Model)
func New(opts ...Option) Model { /* defaults + apply opts */ }

func (m Model) Init() tea.Cmd                           { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { /* ... */ }
func (m Model) View() tea.View                          { return tea.NewView(m.render()) }

// Internal rendering
func (m Model) render() string { /* lipgloss styles from m.theme */ }
// Convenience for non-BubbleTea usage (print, tests)
func (m Model) Render() string { return m.render() }

// Public typed mutators for streaming updates
func (m *Model) SetWidth(w int)  { /* ... */ }
func (m *Model) SetData(d Data)  { /* ... */ }

// Public getters
func (m Model) Width() int  { return m.width }
```

**Conventions:**
- `New(opts...)` — functional options, matches Bubbles v2 (`viewport.New(viewport.WithWidth(80))`)
- `View()` returns `tea.View` via `tea.NewView(m.render())`
- `render()` is the internal string builder (lipgloss styles from theme)
- `Render()` is the public convenience for non-BubbleTea usage
- Value receiver for `View()` / `render()`, pointer receiver for `Set*()` mutations
- Per-component `Theme` struct + `DefaultTheme()` + `WithTheme(t Theme) Option`
- `width int` field, `WithWidth()` option, `SetWidth()` mutator, defensive minimum clamping

---

## Component Inventory

### Implemented

| Component | Package | Status | Mode | Description |
|-----------|---------|--------|------|-------------|
| **QCDashboard** | `qcdashboard/` | Done | inline | Multi-metric quality panel with pass/warn/fail status bars |
| **ThresholdSlider** | `thresholdslider/` | Done | overlay | Numeric threshold with live count feedback |

### In Progress

| Component | Package | Status | Mode | Description |
|-----------|---------|--------|------|-------------|
| **SequenceViewer** | `sequenceviewer/` | Building | inline | Property-aware DNA/RNA/protein viewer with view-switchable coloring, analysis engine, and 3D-ready residues |

### Planned (v0.1)

| Component | Package | Mode | Description |
|-----------|---------|------|-------------|
| **OntologyBrowser** | `ontologybrowser/` | overlay | GO/Reactome/ChEBI tree with lazy-load, search, select |
| **PeriodicTable** | `periodictable/` | overlay | Interactive element selection, property display |

### Planned (v0.2+)

| Component | Package | Mode | Description |
|-----------|---------|------|-------------|
| **SmallMoleculeViewer** | `moleculeviewer/` | overlay | 2D molecule from SMILES with atom navigation, functional-group highlighting |
| **PlateMap** | `platemap/` | inline | 96/384/1536-well assay plate with signal/z-score/control views |
| **VariantLens** | `variantlens/` | overlay | Multi-layer variant consequence inspector (nucleotide/codon/amino acid/feature) |
| **CoverageTrack** | `coveragetrack/` | inline | Linear coverage and interval track viewer with coordinate stepping |
| **ContactMap** | `contactmap/` | overlay | Residue-residue or locus-locus matrix with crosshair navigation |
| **ConstructMap** | `constructmap/` | overlay | Linearized plasmid/construct viewer with feature tracks |
| **PartsBrowser** | `partsbrowser/` | overlay | Biological parts palette grouped by role (promoter, CDS, tag, etc.) |
| **AssemblyPlan** | `assemblyplan/` | overlay | Golden Gate/Gibson/BioBrick assembly timeline and validation |
| **LineageTree** | `lineagetree/` | inline | Sample provenance tree with QC badges and expand/collapse |

**Note:** VectorWorkbench is a **composed application** built from ConstructMap + PartsBrowser + AssemblyPlan + SequenceViewer. It lives in `examples/vectorworkbench/` as a demo, not as a single component.

---

## Directory Structure

```
crust/
  CORE_GUIDE.md               # This file — general rules and mission
  go.mod / go.sum
  result.go                    # SubmitMsg, CancelMsg
  .gitignore                   # CLAUDE.md, .planning/

  sequenceviewer/              # Each component is a self-contained package
    CLAUDE.md                  #   Component-specific implementation guide (gitignored)
    .planning/                 #   Lobster integration mapping (gitignored)
      lobster_map.md
    model.go
    options.go
    types.go
    ...
    sequenceviewer_test.go

  qcdashboard/                 # Same pattern per component
    CLAUDE.md
    .planning/lobster_map.md
    model.go, options.go, ...

  examples/                    # Runnable demos
    sequence/main.go
    threshold/main.go
    qc/main.go
    vectorworkbench/main.go    # Composed application demo
```

**Each component folder** has:
- `CLAUDE.md` — implementation-specific guide for agents working on that component (gitignored)
- `.planning/lobster_map.md` — Lobster integration mapping: which agents, what protocol, what tools (gitignored)
- Source files following the standard shape
- Tests

---

## Development Rules

1. **Pure MIT** — no GPL, no LGPL dependencies. Reimplement algorithms from reference sources.
2. **Charm v2 only** — `charm.land/bubbletea/v2`, `charm.land/bubbles/v2`, `charm.land/lipgloss/v2`
3. **No external deps beyond Charm** — all bio/chem computation is pure Go
4. **No Lobster imports** — Crust must not import `lobster-tui` or `lobster`
5. **Every component has tests** — construction, rendering at various widths, interaction, property correctness
6. **Every component has an example** — runnable `main.go` in `examples/`
7. **Domain correctness** — scientific conventions over aesthetics
8. **CLAUDE.md per component** — agent-isolated implementation context (never committed)
9. **lobster_map.md per component** — Lobster integration design (never committed)

---

## Commands

```bash
go mod tidy           # Sync dependencies
go build ./...        # Compile all
go test ./...         # Run all tests
go vet ./...          # Static analysis
go run ./examples/sequence/    # Run example
```

---

## Reference Libraries (READ-ONLY, do not import)

| Library | Local Path | What to reference | License |
|---------|-----------|-------------------|---------|
| **gochem** | `~/GITHUB/GOLANG_REPOS/gochem/` | Atomic data, vdW radii, amino acid mappings, molecular geometry | LGPL 2.1 |
| **gonetics** | `~/GITHUB/GOLANG_REPOS/gonetics/` | IUPAC alphabets, complement, k-mers, FASTA parsing patterns | GPL v3 |
| **ribbon** | `~/GITHUB/GOLANG_REPOS/ribbon/` | PDB parsing, 118 elements with properties, mesh generation | MIT |
| **tg-oss** | `~/GITHUB/tg-oss/` | Vector editing, OVE, parsers, sequence utilities | MIT |
| **seqviz** | `~/GITHUB/seqviz/` | Sequence viewing, annotations, primers, enzyme sites | MIT |
| **DnaCauldron** | `~/GITHUB/DnaCauldron/` | Assembly workflow: Golden Gate, Gibson, BioBrick | MIT |

**Rule:** Reference algorithms and data tables. Reimplement in Go under MIT. Never import GPL/LGPL code.

---

## Relationship to Lobster

Crust is a **library**. Lobster is a **consumer**.

```
crust/                              # Standalone components (this repo)
  sequenceviewer/                   # tea.Model, typed constructors
  qcdashboard/
  thresholdslider/
  ...

lobster-tui/internal/biocomp/       # Adapter layer (Lobster repo)
  component.go                      # BioComponent interface
  registry.go                       # Factory registry
  overlay.go                        # Overlay frame renderer
  sequenceviewer_adapter.go         # JSON → crust.New(opts...) → BioComponent
  ...
```

Integration mapping lives in each component's `.planning/lobster_map.md` — which Lobster agents trigger it, what protocol messages carry the data, what Python tools are needed.
