# SmallMoleculeViewer

Terminal-native small-molecule inspection for Crust. It accepts SMILES, MOL, or SDF input, generates or preserves 2D coordinates, renders Unicode bonds, and gives you atom-by-atom navigation, search, coloring planes, and a focused inspector pane without leaving Bubble Tea.

## Hero

SmallMoleculeViewer is a focused overlay component for chemistry-heavy TUIs. It is built for the moment where you need to inspect a compound structure, move through its bonded graph, understand what the current atom is doing, and return a typed result back to the host application.

`[GIF PLACEHOLDER: loading a molecule from SMILES and navigating atom-to-atom]`

## What It Does

- Parses practical small-molecule SMILES directly in pure Go.
- Accepts V2000 MOL blocks and SDF payloads when 2D coordinates already exist.
- Repairs missing MOL/SDF connectivity from source coordinates using a local gochem-style covalent-radius pass.
- Generates deterministic 2D coordinates when connectivity is available but coordinates are not.
- Reorients preserved coordinates onto the terminal grid by scoring candidate rotations instead of trusting the source frame.
- Opens single-bond linkers between separate ring systems so multi-ring alkaloids remain legible on the terminal.
- Renders a split view: structure on top, atom/bond inspector below.
- Navigates by graph adjacency with directional arrow-key movement.
- Cycles visualization planes for identity, heteroatoms, aromaticity, partial charge, and scaffold vs R-group.
- Supports inline search for atom labels, bond patterns, and common functional groups.
- Falls back to a compact adjacency view when the terminal is too narrow or the graph is too dense.

`[GIF PLACEHOLDER: cycling color planes and running a functional-group search]`

## Using It

```go
package main

import "github.com/the-omics-os/crust/moleculeviewer"

func makeViewer() moleculeviewer.Model {
	return moleculeviewer.New(
		moleculeviewer.WithName("Caffeine"),
		moleculeviewer.WithSMILES("CN1C=NC2=C1C(=O)N(C(=O)N2C)C"),
		moleculeviewer.WithWidth(96),
		moleculeviewer.WithHeight(24),
	)
}
```

The model is a normal `tea.Model`. `Enter` emits a `crust.SubmitMsg` containing the focused atom and active bond. `Esc` emits `crust.CancelMsg`.

## Demo

Run the local interactive demo from the repo root:

```bash
go run ./moleculeviewer/cmd/demo
```

## Interaction Model

- `Arrow keys`: move to the most directionally appropriate bonded neighbor.
- `Tab`: switch coloring plane.
- `/`: open the search prompt.
- `Enter`: submit the current atom/bond focus.
- `Esc`: close help/search first, then cancel the viewer.
- `?`: show help.

## Search Semantics

The search prompt is intentionally lightweight and terminal-friendly. It supports:

- Element symbols like `O`, `Cl`, `N`
- Atom labels like `O3`
- Bond patterns like `C=O`
- Structural terms like `aromatic`, `hetero`, `scaffold`, `r-group`
- Functional groups like `hydroxyl`, `carbonyl`, `amide`, `amine`, `halide`

## Rendering Strategy

The structure pane uses a deliberately small bond glyph alphabet (`─`, `│`, `╱`, `╲`) and lets color communicate bond order. Before rasterizing, the viewer scores multiple candidate rotations against terminal-grid artifacts so imported coordinates land in a more readable orientation. When the structure still will not read cleanly in the available space, the component switches to an adjacency-list view rather than pretending the terminal depiction is still useful.

`[GIF PLACEHOLDER: graceful fallback from diagram view to adjacency view on a narrow terminal]`

## Implementation Notes

- Pure Go parser and layout pipeline, with no external chemistry toolkit dependency.
- Typed `Molecule`, `Atom`, and `Bond` models for host-side integration.
- Functional-group detection and scaffold approximation are built into the domain layer so search and rendering share the same chemistry metadata.
- Existing coordinates from MOL/SDF input are preserved and normalized rather than discarded.
- Missing bond tables in coordinate-bearing MOL/SDF records are supplemented rather than left disconnected.
- Terminal projection is aspect-aware and rotation-aware rather than a fixed-scale dump of the 2D coordinates.
- Ring-to-ring single-bond linkers are stretched when needed so adjacent ring systems do not collapse into one unreadable cluster.

## Files

- `model.go`: Bubble Tea model, interaction state, submit/cancel behavior.
- `render.go`: split-pane rendering, diagram rasterization, and fallback view.
- `smiles.go`: SMILES/MOL/SDF ingestion.
- `layout.go`: coordinate generation and normalization.
- `molecule.go`: typed chemistry model, search, formula, and functional-group helpers.
- `options.go`: theme and constructor options.
