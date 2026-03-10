# PlateMap

Keyboard-first assay plate inspection for Bubble Tea.

`platemap` renders 96-, 384-, and 1536-well plates directly in the terminal with view-mode switching, fast cursor travel, explicit row/column sweeps, non-committal inspection, and deliberate well confirmation. It is built as a standalone `tea.Model`, so it drops into any Charm v2 app without protocol glue.

<!-- GIF PLACEHOLDER: full plate navigation and mode switching -->

## What It Does

- Navigates wells with arrows and adds `Home` / `End` / `PgUp` / `PgDn` so large plates still feel traversable.
- Cycles between raw signal, normalized, z-score, hit-class, control-layout, and missingness views.
- Makes row and column sweeps explicit with `r` and `c`, while keeping `shift+arrow` as a movement shortcut.
- Separates inspection from confirmation: `Space` opens detail, `Enter` confirms the focused well.
- Keeps a live legend in the lower band with plain-language rows for invariant states and the active lens encoding.
- Supports streaming updates through `SetPlate`, `SetWells`, and `UpsertWell`.

<!-- GIF PLACEHOLDER: row and column selection -->

## Quick Start

```go
package main

import (
	tea "charm.land/bubbletea/v2"

	"github.com/the-omics-os/crust/platemap"
)

func main() {
	plate := platemap.PlateData{
		Format: platemap.Plate96,
		Title:  "Primary Screen Plate",
		Wells: []platemap.Well{
			{Row: 0, Col: 0, Control: platemap.ControlPositive, SampleID: "POS-1", Signal: 1.8, Normalized: 1.1, ZScore: 2.3},
			{Row: 1, Col: 2, Control: platemap.ControlSample, SampleID: "BRCA1-01", Reagent: "BRCA1-siRNA", Signal: 0.42, Normalized: 0.81, ZScore: -2.3, Hit: true},
		},
	}

	model := platemap.New(
		platemap.WithPlate(plate),
		platemap.WithWidth(90),
		platemap.WithHeight(18),
	)

	_, _ = tea.NewProgram(model).Run()
}
```

## Interaction Model

- `Arrow keys`: move between wells.
- `Home` / `End`: jump to the start or end of the current row.
- `PgUp` / `PgDn`: move by one visible band of rows.
- `Tab`: cycle view modes.
- `Shift+Tab`: cycle backward.
- `r`: toggle the current row sweep.
- `c`: toggle the current column sweep.
- `Shift+Arrow`: sweep while moving.
- `Space` or `i`: inspect the focused well without committing.
- `Enter`: confirm the focused well and emit a `SubmitMsg`.
- `Esc`: close help, close inspection, clear sweep, then emit `CancelMsg` when nothing else is active.
- `?`: toggle inline help.

<!-- GIF PLACEHOLDER: expanded well detail footer -->

## View Modes

- `Raw Signal`: sample wells use intensity glyphs derived from raw signal values.
- `Normalized`: same layout, but intensity comes from normalized values.
- `Z-Score`: glyph intensity comes from absolute z-score and color reflects sign.
- `Hit Class`: emphasizes hits while preserving controls and missing wells.
- `Control Layout`: shows control geometry across the plate.
- `Missingness`: isolates missing wells while keeping the rest legible.

The lower legend is mode-aware: one `Always` row explains invariant states such as controls and missing wells, while the lens-specific rows explain how to read the current view in plain language, for example signal scale, z-score magnitude, or sign color.

## Streaming and Embedding

Use the model as a normal Bubble Tea component:

- `SetPlate(PlateData)` replaces the full payload.
- `SetWells([]Well)` replaces only the wells while preserving title and metadata.
- `UpsertWell(Well)` adds or replaces one coordinate, which is useful for live assay updates.
- `SetWidth`, `SetHeight`, `SetViewMode`, and `SetCursor` let a host application drive the view explicitly.

On `Enter`, the model emits a `crust.SubmitMsg` with the focused coordinate, active view, and well payload. `Space` / `i` open inspection without submission. On `Esc`, once local UI state is cleared, it emits a `crust.CancelMsg`.

## Data Model

The component works with three public types:

- `PlateFormat`: `Plate96`, `Plate384`, `Plate1536`
- `Well`: row/column plus raw, normalized, z-score, control, sample, reagent, hit, and missingness state
- `PlateData`: format, wells, title, and arbitrary metadata

Rows are zero-based internally and render as spreadsheet labels (`A`, `B`, ..., `AA`, `AB`, ...). Columns are zero-based internally and render as one-based human coordinates.

## Implementation Notes

- Large plates are rendered through a cropped window rather than trying to draw all rows and columns at once.
- The interaction model is split into focus, inspect, sweep, and confirm so exploration does not feel like commitment.
- Wells are deep-copied on input and output so host mutations do not leak into component state.
- Replicate detail is inferred from shared reagent first, then shared sample ID when reagent is empty.
- The package stays self-contained inside `platemap/` and does not depend on Lobster-specific code.

## Validation

The package includes tests for:

- constructor defaults and option handling
- defensive-copy semantics
- cursor movement and row/column selection
- submit/cancel behavior
- streaming updates
- render stability across 96/384/1536 formats

<!-- GIF PLACEHOLDER: streaming well updates -->
