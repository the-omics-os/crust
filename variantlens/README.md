# VariantLens

VariantLens is a Bubble Tea v2 overlay component for inspecting annotated sequence variants without losing local context.

It keeps four layers aligned in one terminal-native view:

- reference and alternate sequence
- codon and amino-acid consequence
- nearby annotated features
- interpretation lenses for summary, annotation, HGVS, and evidence

This package is built for review workflows where the important work is comparison, not navigation chrome. You step between variants, widen or narrow context, and keep consequence, notation, and feature structure in the same visual frame.

> GIF placeholder: stepping between variants and resizing the context window

## What is implemented

- A standalone `tea.Model` with typed constructors and functional options
- Interactive navigation with arrows, `Tab`, `1-4`, `Enter`, `Esc`, and `?`
- Streaming-friendly setters for replacing context, variants, features, and sequence data
- Aligned reference/alternate rendering with feature tracks
- CDS-aware codon and amino-acid summaries when coding context is present
- A visible navigator rail, tab strip, legend, and footer action bar so the control model is on-screen
- Tests covering navigation, defensive copying, submit/cancel behavior, and rendering invariants

## Interaction model

| Key | Action |
|-----|--------|
| `left` / `right` or `up` / `down` | Move to the previous or next variant |
| `[` / `]` or `-` / `+` | Narrow or widen the visible sequence window |
| `Tab` / `shift+Tab` | Move forward or backward through the lenses |
| `1` / `2` / `3` / `4` | Jump directly to Summary, Annotation, HGVS, or Evidence |
| `Enter` | Submit the focused variant |
| `Esc` | Close help, then cancel the overlay |
| `?` | Toggle help |

`j/k/h/l` are still supported as compatibility aliases, but the advertised controls are the ones shown on-screen.

> GIF placeholder: switching between summary, HGVS, and evidence lenses

## Rendering anatomy

Each focused variant render is organized as:

1. Navigator rail showing the current variant and nearby neighbors
2. Focus header with gene, HGVS, consequence, and current impact
3. Visible lens tabs plus current context-width badge
4. Sequence panel with aligned `ref`, `alt`, caret marker, and codon/amino-acid summary
5. Feature panel clipped to the visible coordinate window
6. Lens-specific body content for summary, annotation, HGVS, or evidence
7. Persistent legend for sequence colors and feature glyphs
8. Persistent footer action bar listing the available controls

That structure is intentional: orient first, inspect second, interpret third, and select only when ready.

> GIF placeholder: reading the aligned sequence, codon, and feature tracks together

## API surface

Construct the component with `New(opts...)`, then update or stream data through typed setters:

```go
ctx := variantlens.VariantContext{
    RefSequence:    "GATTGCGATCCT",
    ReferenceStart: 178,
    ContextSize:    4,
    Variants: []variantlens.Variant{
        {
            Position:    181,
            Ref:         "T",
            Alt:         "G",
            Consequence: "missense",
            HGVS:        "c.181T>G | p.Cys61Gly",
            Gene:        "BRCA1",
            Impact:      "HIGH",
            Evidence:    "ClinVar: Pathogenic | gnomAD: 0.00002",
        },
    },
    Features: []variantlens.Feature{
        {Name: "Exon 5", Type: "exon", Start: 178, End: 189},
        {Name: "CDS 5", Type: "CDS", Start: 178, End: 189},
    },
}

m := variantlens.New(
    variantlens.WithContext(ctx),
    variantlens.WithWidth(88),
)
```

Useful setters:

- `SetContext`
- `SetVariants`
- `SetFeatures`
- `SetReferenceSequence`
- `SetReferenceStart`
- `SetContextSize`
- `SetSelectedVariant`
- `SetWidth`

## Submit and cancel semantics

VariantLens now treats browsing and selection as separate but non-modal concerns:

- `Enter` emits `crust.SubmitMsg` for the focused variant immediately
- `Esc` closes help if it is open, otherwise emits `crust.CancelMsg`
- The current lens and local context stay visible while browsing, so selection no longer depends on a hidden confirmation state

## Notes on coordinates and translation

- `ReferenceStart` anchors the visible sequence so multiple variants and features can share one coordinate system
- If a CDS feature overlaps the focused variant, VariantLens derives the codon frame from that CDS interval
- If no coding frame can be inferred, the component still renders the aligned sequence and features, but omits codon translation

## Package contents

- `model.go`: model state, interaction handling, setters, and emitted messages
- `options.go`: theme and functional options
- `variant.go`: typed domain models and normalization helpers
- `render.go`: aligned sequence rendering, feature tracks, HGVS/evidence lenses, codon translation
- `variantlens_test.go`: behavior and rendering tests
