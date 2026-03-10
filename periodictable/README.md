# PeriodicTable

An interactive periodic table for Bubble Tea that turns the terminal into a chemistry lookup surface instead of a flat reference sheet. It renders the full 118-element layout, lets you move through the real table geometry, and keeps the currently focused element grounded with a readable property panel below the grid.

`PeriodicTable` is designed for overlay-style scientific workflows: quick element selection during molecule inspection, composition checks, isotope reasoning, and any flow where you need the table itself to stay spatially legible while the details update immediately.

## Hero

- Full 118-element periodic table with detached lanthanide and actinide rows.
- Gap-aware arrow navigation, period jumps, lens switching, help overlay, and submit/cancel semantics.
- Category-aware coloring plus an external highlight layer for workflow-specific emphasis.
- Detail panel with atomic mass, electronegativity, electron configuration, and radius data.

## GIF Placeholder

`[ GIF: moving across the table, switching lenses, selecting an element ]`

## Usage

```go
package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/the-omics-os/crust/periodictable"
)

func main() {
	model := periodictable.New(
		periodictable.WithWidth(118),
		periodictable.WithSelected("Fe"),
		periodictable.WithHighlights("C", "N", "O", "S"),
	)

	_, _ = tea.NewProgram(model).Run()
}
```

## Interaction

- Arrow keys move through the real table layout and skip over empty gaps.
- `Tab` cycles the in-cell lens: symbol, atomic mass, electronegativity, electron config.
- `1` through `7` jump directly to a period.
- `Enter` returns a `crust.SubmitMsg` with the focused element.
- `Esc` closes help first, then returns a `crust.CancelMsg`.
- `?` toggles the help overlay.

## Public API

```go
model := periodictable.New(opts...)
model.SetWidth(118)
current := model.Selected()
rendered := model.Render()
```

Available options:

- `WithWidth(w int)`
- `WithTheme(theme Theme)`
- `WithSelected(symbol string)`
- `WithHighlights(symbols ...string)`

## What The Component Renders

The grid stays intentionally compact so the periodic table remains perceivable as a single structure. Instead of trying to fit every property into every cell, the component uses lenses:

- `symbol`: the fastest scanning mode for lookup and selection.
- `atomic mass`: useful when checking rough composition or isotopic intuition.
- `electronegativity`: useful when thinking about polarity and reactivity.
- `electron config`: shows the last orbital token in-grid and the full configuration in the detail panel.

That split keeps the world-level structure visible while still making the focused element information-rich.

## GIF Placeholder

`[ GIF: period jump and help overlay ]`

## Data Model

Each element carries:

- atomic number, symbol, and name
- group and period
- category
- atomic mass
- electronegativity
- electron configuration
- van der Waals radius
- covalent radius

The package hardcodes all 118 elements so it stays dependency-free and protocol-ready.

## Theming

The default theme colors elements by category and uses separate accents for:

- the focused cursor
- externally highlighted elements
- borders and secondary text

You can override the entire palette with `WithTheme`.

## GIF Placeholder

`[ GIF: workflow-specific highlighted elements ]`
