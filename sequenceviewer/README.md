# SequenceViewer

Property-aware DNA, RNA, and protein inspection for Bubble Tea. `sequenceviewer` renders biological sequences as rich residue objects, supports view-cycled coloring, overlays feature annotations, and keeps core analyses close to the terminal interaction loop.

## Hero

- Inline Bubble Tea component with scroll, page jump, help, and view cycling.
- DNA/RNA support includes GC window analysis, complement rendering, ORF search, restriction site detection, and Wallace-rule Tm estimation.
- Protein support includes Clustal-style identity coloring plus hydrophobicity, charge, molecular-weight, and pI-aware views.
- Residues carry optional 3D-oriented metadata (`Coords`, `BFactor`, `VdwRadius`, `Bonds`) so the model is ready for structure-linked consumers.

GIF placeholder:
Add a hero capture here showing Tab view cycling, annotation overlays, and DNA complement toggling.

## What Was Built

The package now includes:

- A full `tea.Model` implementation built on Bubble Tea v2 and `bubbles/viewport`.
- Rich domain types for residues, annotations, ORFs, restriction enzymes/sites, and FASTA records.
- Hardcoded biochemical reference tables for the 20 amino acids, IUPAC nucleotide ambiguity/complements, and a minimal bio-element map.
- A pure-Go analysis layer for GC windows, ORF detection, restriction site scanning, Tm estimation, and pI estimation.
- A FASTA parser for single-record and multi-record inputs.
- A test suite covering rendering, interaction, analysis, amino-acid tables, and FASTA parsing.

## User Experience

The viewer follows Crust’s shared interaction semantics:

- `Up` / `Down`: scroll one line
- `PgUp` / `PgDn`: scroll a page
- `Home` / `End`: jump to the top or bottom
- `Tab`: cycle only the views that make sense for the active sequence type
- `c`: toggle the complement strand for DNA
- `?`: toggle the built-in help block

Annotations render as an inline track above each affected sequence block, and annotated residues are emphasized directly inside the sequence line so the feature layer remains visible while you scroll.

GIF placeholder:
Add a navigation-focused recording here that shows scrolling, paging, and the help overlay.

## Rendering Model

Each rendered block is composed from the same residue slice:

1. Annotation track, when the visible span intersects any feature.
2. Primary sequence line with left/right position labels.
3. Optional complement line for DNA.
4. Optional per-residue property bar for non-identity views.

The component automatically narrows residues-per-line when the terminal width is constrained so small terminals stay legible without horizontal wrapping.

## Public Surface

```go
viewer := sequenceviewer.New(
    sequenceviewer.WithSequence("ATGCGATCGATCG", sequenceviewer.DNA),
    sequenceviewer.WithComplement(true),
    sequenceviewer.WithAnnotations([]sequenceviewer.Annotation{
        {Name: "Promoter", Start: 1, End: 9, Direction: 1},
    }),
    sequenceviewer.WithGCWindow(12),
    sequenceviewer.WithWidth(80),
)
```

Primary API:

- `WithSequence`, `WithResidues`, `WithView`, `WithComplement`
- `WithAnnotations`, `WithResiduesPerLine`, `WithWidth`, `WithTheme`, `WithHeader`, `WithGCWindow`
- `SetSequence`, `SetResidues`, `SetView`, `SetWidth`
- `Sequence`, `Type`, `Length`, `ViewMode`, `Residues`, `ORFs`, `RestrictionSites`, `GCContent`, `MeltingTemp`, `IsoelectricPoint`
- `ParseFASTA` / `ParseFASTAReader`

## Validation

The package was built with local tests for:

- view cycling and key handling
- complement rendering and help rendering
- width-sensitive output
- residue property enrichment
- GC windows, ORFs, restriction sites, Tm, and pI
- FASTA parsing edge cases

GIF placeholder:
Add an analysis-focused recording here that shows GC view, property bars, and feature highlighting on a real sequence.
