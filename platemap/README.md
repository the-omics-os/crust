# PlateMap

Keyboard-first assay plate inspection for Bubble Tea.

`platemap` renders 96-, 384-, and 1536-well plates directly in the terminal with view-mode switching, cursor navigation, row/column focus, streaming updates, and an expandable well detail footer. It is built as a standalone `tea.Model`, so it drops into any Charm v2 app without protocol glue.

<!-- GIF PLACEHOLDER: full plate navigation and mode switching -->

## What It Does

- Navigates wells with arrow keys and keeps large plates usable through automatic row/column panning.
- Cycles between raw signal, normalized, z-score, hit-class, control-layout, and missingness views.
- Highlights row or column sweeps with `shift+arrow`.
- Expands focused well detail on `Enter` and emits a `crust.SubmitMsg` for host applications.
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
- `Tab`: cycle view modes.
- `Shift+Tab`: cycle backward.
- `Shift+Arrow`: move while selecting the active row or column.
- `Enter`: expand detail for the focused well and emit a `SubmitMsg`.
- `Esc`: close help, collapse detail, clear selection, then emit `CancelMsg` when nothing else is active.
- `?`: toggle inline help.

<!-- GIF PLACEHOLDER: expanded well detail footer -->

## View Modes

- `Raw Signal`: sample wells use intensity glyphs derived from raw signal values.
- `Normalized`: same layout, but intensity comes from normalized values.
- `Z-Score`: glyph intensity comes from absolute z-score and color reflects sign.
- `Hit Class`: emphasizes hits while preserving controls and missing wells.
- `Control Layout`: shows control geometry across the plate.
- `Missingness`: isolates missing wells while keeping the rest legible.

## Streaming and Embedding

Use the model as a normal Bubble Tea component:

- `SetPlate(PlateData)` replaces the full payload.
- `SetWells([]Well)` replaces only the wells while preserving title and metadata.
- `UpsertWell(Well)` adds or replaces one coordinate, which is useful for live assay updates.
- `SetWidth`, `SetHeight`, `SetViewMode`, and `SetCursor` let a host application drive the view explicitly.

On `Enter`, the model emits a `crust.SubmitMsg` with the focused coordinate, active view, and well payload. On `Esc`, once local UI state is cleared, it emits a `crust.CancelMsg`.

## Data Model

The component works with three public types:

- `PlateFormat`: `Plate96`, `Plate384`, `Plate1536`
- `Well`: row/column plus raw, normalized, z-score, control, sample, reagent, hit, and missingness state
- `PlateData`: format, wells, title, and arbitrary metadata

Rows are zero-based internally and render as spreadsheet labels (`A`, `B`, ..., `AA`, `AB`, ...). Columns are zero-based internally and render as one-based human coordinates.

## Implementation Notes

- Large plates are rendered through a cropped window rather than trying to draw all rows and columns at once.
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
