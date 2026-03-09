# Architecture

**Analysis Date:** 2026-03-09

## Pattern Overview

**Overall:** Component Library (standalone BubbleTea v2 `tea.Model` implementations)

**Key Characteristics:**
- Each component is an independent Go package implementing the `tea.Model` interface from Charm BubbleTea v2
- Functional options pattern (`New(opts ...Option)`) for construction, matching Bubbles v2 idiom
- No framework, no routing, no protocol layer -- pure library consumed by embedding in any BubbleTea application
- Shared result signaling via root-level `crust.SubmitMsg` and `crust.CancelMsg` types

## Layers

**Root Package (`crust`):**
- Purpose: Shared message types for component completion signaling
- Location: `/Users/tyo/Omics-OS/crust/result.go`
- Contains: `SubmitMsg` and `CancelMsg` structs (both carry `Component` string identifier and payload)
- Depends on: Nothing
- Used by: Interactive components (e.g., `thresholdslider`) that need to signal submit/cancel to the host app

**Component Packages:**
- Purpose: Self-contained `tea.Model` implementations, one per scientific UI widget
- Location: `/Users/tyo/Omics-OS/crust/{componentname}/`
- Contains: Model definition, functional options, theme, domain types, tests
- Depends on: `charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`, and optionally root `crust` package for message types
- Used by: Any BubbleTea v2 application that imports the package

**Examples:**
- Purpose: Runnable demo programs showing how to embed each component
- Location: `/Users/tyo/Omics-OS/crust/examples/{component}/main.go`
- Contains: Minimal BubbleTea programs wrapping a single component
- Depends on: Component packages and root `crust` package
- Used by: Developers evaluating or learning the library

## Data Flow

**Non-Interactive Component (QCDashboard):**

1. Host creates model via `qcdashboard.New(opts...)` with metrics data
2. Model renders via `View()` / `Render()` -- no user interaction handled
3. Host updates metrics programmatically via `SetMetrics()`, `SetWidth()`, etc.
4. `Update()` is a no-op (returns model unchanged, nil cmd)

**Interactive Component (ThresholdSlider):**

1. Host creates model via `thresholdslider.New(opts...)` with range/step/default
2. Host embeds slider's `Update()` in its own `Update()`, forwarding messages
3. Key presses (`left`/`right`/`shift+left`/`shift+right`) adjust value, emit `ValueChangedMsg` via `tea.Cmd`
4. Host listens for `ValueChangedMsg` to update live previews (e.g., recalculate counts)
5. `Enter` emits `crust.SubmitMsg` with final value in `Data["value"]`
6. `Esc` emits `crust.CancelMsg`
7. Host catches `SubmitMsg`/`CancelMsg` to close the component and act on the result

**State Management:**
- Each component owns its state as unexported struct fields on `Model`
- State is mutated via `Update()` (BubbleTea loop) or explicit setter methods (`SetMetrics`, `SetCount`, `SetWidth`)
- Getters return defensive copies (e.g., `Metrics()` returns a copy of the slice)
- Value receivers for read-only methods (`View`, `Render`, `Value`), pointer receivers for mutations (`SetMetrics`, `SetCount`)

## Key Abstractions

**`tea.Model` Interface:**
- Purpose: Standard BubbleTea component contract
- Examples: `qcdashboard.Model` in `/Users/tyo/Omics-OS/crust/qcdashboard/model.go`, `thresholdslider.Model` in `/Users/tyo/Omics-OS/crust/thresholdslider/model.go`
- Pattern: `Init() tea.Cmd`, `Update(tea.Msg) (tea.Model, tea.Cmd)`, `View() tea.View`

**Functional Options (`Option`):**
- Purpose: Configurable construction without breaking API
- Examples: `qcdashboard.WithMetrics()`, `thresholdslider.WithRange()`, `WithTheme()`
- Pattern: `type Option func(*Model)` defined per component in `options.go`

**Per-Component Theme:**
- Purpose: Dynamic visual customization using `image/color.Color` interface
- Examples: `qcdashboard.Theme` in `/Users/tyo/Omics-OS/crust/qcdashboard/options.go`, `thresholdslider.Theme` in `/Users/tyo/Omics-OS/crust/thresholdslider/options.go`
- Pattern: `Theme` struct with `color.Color` fields, `DefaultTheme()` constructor, `WithTheme()` option

**Result Messages:**
- Purpose: Standardized component completion signaling across all interactive components
- Examples: `crust.SubmitMsg`, `crust.CancelMsg` in `/Users/tyo/Omics-OS/crust/result.go`
- Pattern: Host app type-switches on these in its `Update()` to handle component results

**Domain Types:**
- Purpose: Typed data structures for scientific content
- Examples: `qcdashboard.Metric` in `/Users/tyo/Omics-OS/crust/qcdashboard/metric.go`
- Pattern: Simple structs with exported fields, defined in component package

## Entry Points

**Library Entry (no single entry point):**
- Location: Each package is independently importable
- Triggers: `import "github.com/the-omics-os/crust/qcdashboard"` etc.
- Responsibilities: Provide `New()` constructor and `tea.Model` implementation

**Example Programs:**
- Location: `/Users/tyo/Omics-OS/crust/examples/qc/main.go` (non-interactive print)
- Location: `/Users/tyo/Omics-OS/crust/examples/threshold/main.go` (interactive BubbleTea program)
- Triggers: `go run ./examples/qc/` or `go run ./examples/threshold/`
- Responsibilities: Demonstrate component usage patterns

## Error Handling

**Strategy:** Defensive defaults, no panics, no error returns from constructors

**Patterns:**
- Constructors clamp invalid values silently (e.g., `thresholdslider.New` clamps default to min/max range)
- Width floors prevent rendering breakage (minimum widths of 20 enforced in `render()`)
- Nil-safe: `New()` with no options produces a valid, renderable model with sensible defaults
- No `error` returns from `New()`, `Update()`, or `View()` -- follows BubbleTea convention

## Cross-Cutting Concerns

**Logging:** None. Components are silent -- no logging framework. Host apps handle logging.

**Validation:** Implicit via clamping and floor values. No explicit validation errors surfaced to users.

**Authentication:** Not applicable -- this is a UI component library with no network or auth concerns.

**Rendering:** All components provide both `View() tea.View` (BubbleTea interface) and `Render() string` (for embedding/testing). Both call a shared private `render()` method.

---

*Architecture analysis: 2026-03-09*
