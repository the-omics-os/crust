# OntologyBrowser

Interactive ontology navigation for Bubble Tea. `ontologybrowser` renders a scrollable term tree with lazy expansion, a persistent filter bar, single-node submission, and typed setters for host-driven data loading.

[GIF PLACEHOLDER: browsing roots, expanding branches, and selecting a term]

## What You Get

- A standalone `tea.Model` that can be embedded directly into any Charm v2 application.
- Lazy-loading via `ExpandMsg`, so the component never couples itself to a specific ontology API or file format.
- A persistent filter lens that searches all currently loaded terms and reveals matching paths inside the tree.
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

This uses in-memory ontology data and exercises lazy expansion, filtering, selection, and cancel flows without touching any code outside `ontologybrowser/`.

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

- `Up` / `Down`: move through the visible tree, or step through matches while filtering
- `Right`: expand the current branch, or request children if they are not loaded yet
- `Left`: collapse the current branch or move to its parent
- `Enter`: select the currently highlighted term
- `Type` or `/`: start filtering immediately
- `Esc`: clear the filter, return to browse, or cancel the browser
- `Tab`: switch between browse focus and filter focus
- `Space`: toggle the current branch open or closed
- `?`: show help

The footer includes a compact legend, and `?` expands that legend into the help view so the tree symbols and color semantics are explicit in the interface itself.

## Filter Behavior

Filtering searches all currently loaded terms, including descendants inside collapsed branches. When a match is selected, the browser expands the ancestor path so the user stays oriented in the tree instead of jumping into a disconnected result list.

Unloaded descendants are not searchable yet. That boundary is intentional: the component stays data-source agnostic, while the host controls when more ontology structure is fetched.

The match scorer prioritizes:

1. Exact and substring matches in term name
2. Matches in term ID
3. Description matches
4. Token and subsequence fallbacks for shorter fuzzy queries

[GIF PLACEHOLDER: typing into the filter bar and following revealed paths]

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
- Filtering is scoped to the currently loaded ontology graph. If you need global ontology search beyond loaded nodes, the host should provide it separately.
- I left the GIF anchors inline so you can replace them with recordings later without restructuring the document.
