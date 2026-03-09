# Codebase Structure

**Analysis Date:** 2026-03-09

## Directory Layout

```
crust/
├── .claude/                    # Claude Code settings
├── .planning/                  # Planning documents
│   └── codebase/               # Codebase analysis (this file)
├── examples/                   # Runnable demo programs
│   ├── qc/                     # QCDashboard example
│   │   └── main.go
│   └── threshold/              # ThresholdSlider example
│       └── main.go
├── qcdashboard/                # QC metrics dashboard component
│   ├── model.go                # tea.Model implementation + render logic
│   ├── options.go              # Theme struct, DefaultTheme, functional options
│   ├── metric.go               # Metric data type
│   └── qcdashboard_test.go     # Unit tests
├── thresholdslider/            # Interactive threshold slider component
│   ├── model.go                # tea.Model implementation + render logic + helpers
│   ├── options.go              # Theme struct, DefaultTheme, functional options
│   └── thresholdslider_test.go # Unit tests
├── result.go                   # Root package: SubmitMsg, CancelMsg types
├── CLAUDE.md                   # Project instructions and design spec
├── go.mod                      # Go module definition
└── go.sum                      # Dependency checksums
```

## Directory Purposes

**`qcdashboard/`:**
- Purpose: Non-interactive QC metrics display with colored status bars
- Contains: Model, options/theme, metric type, tests
- Key files: `model.go` (all rendering logic), `metric.go` (domain type)

**`thresholdslider/`:**
- Purpose: Interactive numeric threshold adjustment with keyboard control
- Contains: Model, options/theme, tests
- Key files: `model.go` (update logic, render, helper functions like `clamp`, `roundTo`, `decimalPlaces`)

**`examples/`:**
- Purpose: Runnable demonstrations of each component
- Contains: One subdirectory per component with a `main.go`
- Key files: `examples/threshold/main.go` shows full interactive embedding pattern

## Key File Locations

**Entry Points:**
- No single entry point -- library with per-package imports

**Configuration:**
- `/Users/tyo/Omics-OS/crust/go.mod`: Go module `github.com/the-omics-os/crust`, Go 1.25, Charm v2 deps

**Core Logic:**
- `/Users/tyo/Omics-OS/crust/result.go`: Shared `SubmitMsg` and `CancelMsg` message types
- `/Users/tyo/Omics-OS/crust/qcdashboard/model.go`: QCDashboard `tea.Model` implementation
- `/Users/tyo/Omics-OS/crust/thresholdslider/model.go`: ThresholdSlider `tea.Model` implementation

**Domain Types:**
- `/Users/tyo/Omics-OS/crust/qcdashboard/metric.go`: `Metric` struct (Name, Value, Min, Max, Unit, Status)

**Theme/Options:**
- `/Users/tyo/Omics-OS/crust/qcdashboard/options.go`: `Theme`, `DefaultTheme()`, `Option` type, `With*` functions
- `/Users/tyo/Omics-OS/crust/thresholdslider/options.go`: `Theme`, `DefaultTheme()`, `Option` type, `With*` functions

**Testing:**
- `/Users/tyo/Omics-OS/crust/qcdashboard/qcdashboard_test.go`: 11 tests covering construction, rendering, defensive copies
- `/Users/tyo/Omics-OS/crust/thresholdslider/thresholdslider_test.go`: 14 tests covering key handling, clamping, submit/cancel, boundaries

## Naming Conventions

**Files:**
- `model.go`: Always contains the `Model` struct and `tea.Model` implementation
- `options.go`: Always contains `Theme` struct, `DefaultTheme()`, `Option` type, and `With*` option functions
- `metric.go` / `node.go` / `elements.go`: Domain-specific data types, named after the concept
- `{packagename}_test.go`: Test file named after the package

**Directories:**
- Component packages: lowercase single word matching the component name (`qcdashboard`, `thresholdslider`)
- Example programs: short name matching the component concept (`qc`, `threshold`)

**Types and Functions:**
- `Model`: Always the BubbleTea model struct (one per package)
- `Theme`: Always the per-component theme struct
- `Option`: Always `func(*Model)` for functional options
- `New(opts ...Option) Model`: Always the constructor
- `DefaultTheme() Theme`: Always the default theme constructor
- `With*(...)  Option`: Functional option constructors (e.g., `WithWidth`, `WithTheme`, `WithMetrics`)
- `Set*(...)`: Pointer-receiver mutators (e.g., `SetMetrics`, `SetCount`, `SetWidth`)
- `View() tea.View`: BubbleTea interface (value receiver)
- `Render() string`: Plain string rendering for embedding (value receiver)

## Where to Add New Code

**New Component (e.g., SequenceViewer):**
- Create directory: `crust/sequenceviewer/`
- Required files:
  - `model.go` -- `Model` struct, `New()`, `Init()`, `Update()`, `View()`, `Render()`, private `render()`
  - `options.go` -- `Theme` struct, `DefaultTheme()`, `Option` type, `With*` functions
  - `sequenceviewer_test.go` -- unit tests
- Optional domain type file (e.g., `coloring.go` for IUPAC color logic)
- Add example: `crust/examples/sequence/main.go`
- If interactive: import `github.com/the-omics-os/crust` and return `crust.SubmitMsg`/`crust.CancelMsg` on completion
- If non-interactive: `Update()` returns model unchanged with nil cmd

**New Shared Type:**
- Add to `/Users/tyo/Omics-OS/crust/result.go` or create a new file in the root `crust` package
- Only add here if the type is used across multiple component packages

**New Example:**
- Create `crust/examples/{name}/main.go`
- For non-interactive components: create model, call `Render()`, print to stdout
- For interactive components: wrap in a host `model` struct, run with `tea.NewProgram`

**Shared Theme Package (future):**
- If per-component themes converge, extract to `crust/theme/` with `theme.go` and `defaults.go`
- Not yet created -- currently each component defines its own `Theme` struct

## Special Directories

**`.planning/`:**
- Purpose: Project planning and codebase analysis documents
- Generated: By tooling/analysis
- Committed: Yes

**`examples/`:**
- Purpose: Runnable demo programs for each component
- Generated: No (hand-written)
- Committed: Yes

## Planned but Not Yet Created

Per `CLAUDE.md`, these components and directories are planned for v0.1 but do not yet exist:

- `crust/sequenceviewer/` -- DNA/RNA/protein sequence display (flagship component)
- `crust/ontologybrowser/` -- Tree navigation for GO/Reactome/Disease Ontology
- `crust/periodictable/` -- Interactive periodic table
- `crust/theme/` -- Shared theme types (if patterns converge)
- `crust/examples/sequence/`, `crust/examples/ontology/`, `crust/examples/periodic/`

---

*Structure analysis: 2026-03-09*
