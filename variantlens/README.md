# VariantLens

VariantLens is a Bubble Tea v2 overlay component for inspecting annotated sequence variants without losing local context.

It keeps four layers aligned in one terminal-native view:

- reference and alternate sequence
- codon and amino-acid consequence
- nearby annotated features
- interpretation lenses for summary, detail, HGVS, and evidence

This package is built for review workflows where the important work is comparison, not navigation chrome. You step between variants, widen or narrow context, and keep consequence, notation, and feature structure in the same visual frame.

> GIF placeholder: stepping between variants and resizing the context window

## What is implemented

- A standalone `tea.Model` with typed constructors and functional options
- Interactive navigation with `j/k`, arrows, `Tab`, `Enter`, `Esc`, and `?`
- Streaming-friendly setters for replacing context, variants, features, and sequence data
- Aligned reference/alternate rendering with feature tracks
- CDS-aware codon and amino-acid summaries when coding context is present
- Overlay interaction semantics that unwind help and focused detail before canceling
- Tests covering navigation, defensive copying, submit/cancel behavior, and rendering invariants

## Interaction model

| Key | Action |
|-----|--------|
| `j` / `k` or `down` / `up` | Step between variants |
| `h` / `l` or `left` / `right` | Narrow or widen the visible sequence window |
| `Tab` | Cycle `summary -> detail -> hgvs -> evidence` |
| `Enter` | Open focused detail, then confirm the current variant |
| `Esc` | Close help, leave focused detail, then cancel the overlay |
| `?` | Toggle help |

> GIF placeholder: switching between summary, HGVS, and evidence lenses

## Rendering anatomy

Each focused variant render is organized as:

1. Header with variant identity, gene, consequence, current lens, and impact level
2. Sequence panel with aligned `ref`, `alt`, caret marker, and codon/amino-acid summary
3. Feature panel clipped to the visible coordinate window
4. Lens-specific body content for summary, detail, HGVS, or evidence
5. Optional focused-detail box used as the confirmation step before `SubmitMsg`

That structure is intentional: identity first, local molecular consequence second, annotation context third, and supporting interpretation last.

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

VariantLens is an overlay inspector, so confirmation is explicit:

- The first `Enter` opens focused detail for the selected variant
- The second `Enter` emits `crust.SubmitMsg` with the focused variant, index, view mode, and context size
- `Esc` closes active UI layers before emitting `crust.CancelMsg`

This lets a host application treat browsing and confirmation as separate steps while keeping the terminal interaction compact.

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
