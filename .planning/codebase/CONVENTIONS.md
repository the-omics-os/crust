# Coding Conventions

**Analysis Date:** 2026-03-09

## Naming Patterns

**Files:**
- `model.go` — BubbleTea `tea.Model` implementation (one per component package)
- `options.go` — `Option` type, functional options (`With*`), `Theme` struct, `DefaultTheme()`
- `metric.go` — domain types used by the component (e.g., `Metric` struct)
- `*_test.go` — named after the package: `qcdashboard_test.go`, `thresholdslider_test.go`

**Packages:**
- Flat single-word, all lowercase, matching the component name: `qcdashboard`, `thresholdslider`
- One component per package. Import as `github.com/the-omics-os/crust/thresholdslider`

**Functions:**
- Constructors: `New(opts ...Option) Model` — always named `New`, always returns `Model`
- Functional options: `With<Field>` — e.g., `WithTitle`, `WithRange`, `WithStep`, `WithTheme`
- Getters: bare noun — `Value()`, `Title()`, `Metrics()`
- Setters: `Set<Field>` — `SetMetrics()`, `SetCount()`, `SetWidth()`
- Internal helpers: unexported camelCase — `clamp()`, `roundTo()`, `decimalPlaces()`, `formatValue()`
- Theme factory: `DefaultTheme() Theme`

**Types:**
- Component model: always `Model`
- Options: always `type Option func(*Model)`
- Theme: always `type Theme struct { ... }` — per-component in `options.go`
- Domain types: PascalCase nouns — `Metric`, `ValueChangedMsg`
- Message types: `*Msg` suffix — `ValueChangedMsg`, `SubmitMsg`, `CancelMsg`

**Variables:**
- Local: short camelCase — `m`, `w`, `km`, `val`, `fillRatio`
- Loop iterators: single letter or short — `i`, `tt`
- Style variables: `<purpose>Style` — `borderStyle`, `titleStyle`, `valueStyle`

## Code Style

**Formatting:**
- Standard `gofmt` (no custom formatter config detected)
- No `.golangci.yml`, `.editorconfig`, or `Makefile` present
- Run `go vet ./...` for static analysis

**Linting:**
- No linter configuration. Use `go vet ./...` as the minimum check.

**Line length:**
- No enforced limit. Lines stay reasonable (under ~120 chars) by convention.

## Import Organization

**Order:**
1. Standard library (`fmt`, `math`, `strings`, `image/color`, `strconv`)
2. External dependencies (`charm.land/bubbletea/v2`, `charm.land/lipgloss/v2`)
3. Internal project imports (`github.com/the-omics-os/crust`)

**Aliases:**
- BubbleTea is always aliased: `tea "charm.land/bubbletea/v2"`
- Lipgloss uses bare import: `"charm.land/lipgloss/v2"`

**Rules:**
- Groups separated by blank lines
- No dot imports
- No underscore imports (except for side effects, not yet observed)

## Component Architecture Pattern

Every Crust component follows this exact structure. Use this as a template for new components.

**Constructor pattern (functional options):**
```go
func New(opts ...Option) Model {
    m := Model{
        // sensible defaults
        title: "Default Title",
        theme: DefaultTheme(),
        width: 80,
    }
    for _, opt := range opts {
        opt(&m)
    }
    // Post-construction validation (e.g., clamping)
    return m
}
```

**BubbleTea interface:**
```go
func (m Model) Init() tea.Cmd                           { return nil }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m Model) View() tea.View                          { return tea.NewView(m.render()) }
```

**Dual render pattern — every component exposes both:**
```go
// View() for BubbleTea integration (returns tea.View)
func (m Model) View() tea.View { return tea.NewView(m.render()) }

// Render() for standalone/embedding use (returns string)
func (m Model) Render() string { return m.render() }

// render() — private, does the real work
func (m Model) render() string { ... }
```

**Receiver conventions:**
- Value receiver (`m Model`) for `Init()`, `Update()`, `View()`, `Render()`, and all getters
- Pointer receiver (`m *Model`) for setters: `SetMetrics()`, `SetCount()`, `SetWidth()`

**Completion signaling (interactive components only):**
- Submit: return `crust.SubmitMsg{Component: "component_name", Data: map[string]any{...}}`
- Cancel: return `crust.CancelMsg{Component: "component_name", Reason: "user cancelled"}`
- Value change: return component-specific `ValueChangedMsg` for live preview

## Defensive Copying

**Always make defensive copies for slice fields:**
```go
// In WithMetrics option:
func WithMetrics(metrics []Metric) Option {
    return func(m *Model) { m.metrics = append([]Metric(nil), metrics...) }
}

// In SetMetrics setter:
func (m *Model) SetMetrics(metrics []Metric) {
    m.metrics = append([]Metric(nil), metrics...)
}

// In Metrics getter:
func (m Model) Metrics() []Metric { return append([]Metric(nil), m.metrics...) }
```

This is tested explicitly — see `TestMetrics_ReturnsCopy`, `TestSetMetrics_DefensiveCopy`, `TestWithMetrics_DefensiveCopy` in `qcdashboard/qcdashboard_test.go`.

## Theme System

**Per-component themes in `options.go`:**
```go
type Theme struct {
    Pass      color.Color  // use image/color.Color interface
    Warn      color.Color
    // ...component-specific fields
}

func DefaultTheme() Theme {
    return Theme{
        Pass: lipgloss.Color("42"),  // ANSI 256 color strings
        // ...
    }
}
```

- All color fields use `color.Color` interface (from `image/color`)
- Default values use `lipgloss.Color("N")` with ANSI 256 color codes
- Theme is set via `WithTheme(t Theme) Option`
- No hardcoded colors in rendering logic — always go through `m.theme.*`

## Error Handling

**Patterns:**
- No explicit error returns from constructors — invalid values are clamped (see `thresholdslider.New` clamping value to range)
- Minimum width guards in render functions: `if w < 20 { w = 20 }`
- No panics. No `log.Fatal`. Components degrade gracefully.

## Logging

**Framework:** None. Components are silent — no logging at all. The host application handles all logging.

## Comments

**Package-level doc comments:**
- Every package has a doc comment on the `package` line in `model.go`
- Format: `// Package <name> provides <description>.` followed by blank line and usage notes

**Exported symbol comments:**
- Every exported type, function, and method has a `//` comment
- Format: `// <Name> <verb phrase>.` — e.g., `// SetMetrics replaces all metrics (defensive copy).`
- `// <Method> implements tea.Model.` for BubbleTea interface methods

**Inline comments:**
- Explain "why" not "what": `// Clamp value to range.`, `// Handle scientific notation: 1e-06 means 6 decimal places.`
- Color annotations in theme defaults: `lipgloss.Color("42")  // green`

## Module Design

**Exports:**
- Each package exports: `Model`, `Option`, `Theme`, `DefaultTheme()`, `New()`, `With*()` options
- Domain types exported as needed: `Metric`, `ValueChangedMsg`
- No barrel files — Go packages are already self-contained

**Root package (`crust`):**
- Contains only shared message types: `SubmitMsg`, `CancelMsg` in `result.go`
- Minimal — components import `crust` only when they need `SubmitMsg`/`CancelMsg`

**Dependencies between packages:**
- Component packages may import root `crust` package for message types
- Component packages must NOT import other component packages
- No Lobster imports allowed — Crust is a standalone library

---

*Convention analysis: 2026-03-09*
