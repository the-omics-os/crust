# OntologyBrowser

Interactive ontology navigation for Bubble Tea. `ontologybrowser` renders a scrollable term tree with lazy expansion, a built-in search pane, single-node submission, and typed setters for host-driven data loading.

[GIF PLACEHOLDER: browsing roots, expanding branches, and selecting a term]

## What You Get

- A standalone `tea.Model` that can be embedded directly into any Charm v2 application.
- Lazy-loading via `ExpandMsg`, so the component never couples itself to a specific ontology API or file format.
- A visible-node search pane for narrowing the currently loaded tree without switching mental context.
- Submission through `crust.SubmitMsg` and cancellation through `crust.CancelMsg`, matching the rest of Crust.
- Full theme control through `WithTheme`, plus width and height control for overlay integration.

## Quick Start

```go
package main

import (
	tea "charm.land/bubbletea/v2"
	"github.com/the-omics-os/crust/ontologybrowser"
)

func buildModel() ontologybrowser.Model {
	return ontologybrowser.New(
		ontologybrowser.WithWidth(88),
		ontologybrowser.WithHeight(24),
		ontologybrowser.WithRoots([]ontologybrowser.OntologyNode{
			{ID: "GO:0008150", Name: "biological_process"},
			{ID: "GO:0003674", Name: "molecular_function"},
			{ID: "GO:0005575", Name: "cellular_component"},
		}),
	)
}

func main() {
	_ = tea.NewProgram(buildModel())
}
```

## Demo

Run the local interactive demo from this folder:

```bash
go run ./cmd/demo
```

This uses in-memory ontology data and exercises lazy expansion, search, selection, and cancel flows without touching any code outside `ontologybrowser/`.

## Host Integration Pattern

When the user expands an unloaded node, the component emits:

```go
ontologybrowser.ExpandMsg{NodeID: "GO:0008150"}
```

The host then resolves children from its own source of truth and pushes them back:

```go
browser.SetChildren("GO:0008150", children)
```

When the user confirms a node, the browser emits `crust.SubmitMsg` with:

- `id`
- `name`
- `description`
- `depth`
- `path_ids`
- `path_names`

[GIF PLACEHOLDER: lazy-load round trip from expand request to populated branch]

## Interaction Model

- `Up` / `Down`: move through visible nodes
- `Right` / `Enter`: expand a branch, or fetch children if the node is not loaded yet
- `Left`: collapse the current branch or move to its parent
- `Tab` or `/`: focus the search pane
- `Enter` in search: submit the highlighted result
- `Esc`: leave help, leave search, or cancel the browser
- `?`: show help

## Search Behavior

Search only matches currently visible nodes. That keeps the component deterministic and independent of background fetch logic: if a branch has not been expanded yet, its hidden descendants are not part of the search result set.

The search scorer prioritizes:

1. Exact and substring matches in term name
2. Matches in term ID
3. Description matches
4. Token and subsequence fallbacks for shorter fuzzy queries

[GIF PLACEHOLDER: typing into search and selecting a result]

## Public API

```go
func New(opts ...Option) Model
func WithRoots(nodes []OntologyNode) Option
func WithWidth(w int) Option
func WithHeight(h int) Option
func WithTheme(t Theme) Option

func (m *Model) SetRoots(nodes []OntologyNode)
func (m *Model) SetChildren(nodeID string, children []OntologyNode)
func (m *Model) SetWidth(w int)
func (m *Model) SetHeight(h int)
func (m Model) Selected() *OntologyNode
func (m Model) Render() string
```

## Notes

- The component is self-contained inside `ontologybrowser/`; no Lobster or protocol code is imported.
- Search is intentionally scoped to visible nodes. If you need global ontology search, the host should provide it separately.
- I left the GIF anchors inline so you can replace them with recordings later without restructuring the document.
