# Technology Stack

**Analysis Date:** 2026-03-09

## Languages

**Primary:**
- Go 1.25+ (declared in `go.mod`) — all source code

**Secondary:**
- None

## Runtime

**Environment:**
- Go 1.26.0 (installed on development machine, darwin/arm64)
- Module requires Go 1.25.0 (`go.mod` line 3)

**Package Manager:**
- Go modules (`go mod`)
- Lockfile: `go.sum` present (43 lines, fully resolved)

## Frameworks

**Core:**
- BubbleTea v2 `charm.land/bubbletea/v2` v2.0.1 — TUI framework (the Elm architecture for terminals)
- Lipgloss v2 `charm.land/lipgloss/v2` v2.0.0 — terminal styling (colors, layout, borders)

**Testing:**
- Go standard library `testing` package — all tests use built-in `testing.T`
- No third-party test framework (no testify, no gomega)

**Build/Dev:**
- `go build` / `go test` / `go vet` — standard Go toolchain
- No Makefile, no task runner, no build script

## Key Dependencies

**Critical (direct):**
- `charm.land/bubbletea/v2` v2.0.1 — core TUI runtime (tea.Model, tea.Cmd, tea.Program)
- `charm.land/lipgloss/v2` v2.0.0 — styling engine (colors, layout composition)

**Transitive (indirect, pulled by Charm):**
- `charm.land/bubbles/v2` v2.0.0 — Charm component library (available but not directly imported yet)
- `github.com/charmbracelet/colorprofile` v0.4.2 — terminal color capability detection
- `github.com/charmbracelet/ultraviolet` v0.0.0-20260205 — Charm rendering internals
- `github.com/charmbracelet/x/ansi` v0.11.6 — ANSI escape sequence handling
- `github.com/charmbracelet/x/term` v0.2.2 — terminal dimensions and capabilities
- `github.com/charmbracelet/x/termios` v0.1.1 — terminal I/O settings
- `github.com/charmbracelet/x/windows` v0.2.2 — Windows terminal support
- `github.com/clipperhouse/displaywidth` v0.11.0 — Unicode display width calculation
- `github.com/lucasb-eyer/go-colorful` v1.3.0 — color space conversions
- `github.com/mattn/go-runewidth` v0.0.20 — rune width calculation
- `github.com/muesli/cancelreader` v0.2.2 — cancelable stdin reader
- `github.com/rivo/uniseg` v0.4.7 — Unicode text segmentation
- `github.com/xo/terminfo` v0.0.0-20220910 — terminfo database access
- `golang.org/x/sync` v0.19.0 — concurrency primitives
- `golang.org/x/sys` v0.41.0 — low-level OS interaction

**Dev dependencies (test only, in go.sum):**
- `github.com/charmbracelet/x/exp/golden` v0.0.0-20250806 — golden file test utilities (snapshot testing)
- `github.com/aymanbagabas/go-udiff` v0.4.0 — unified diff for test output comparison

## Configuration

**Environment:**
- No environment variables required
- No `.env` files present
- Pure library — configuration is via Go constructors and functional options

**Build:**
- `go.mod` — module declaration and dependency versions
- `go.sum` — dependency integrity checksums
- No build configuration files (no goreleaser, no golangci-lint config)

## Platform Requirements

**Development:**
- Go 1.25+ toolchain
- No OS-specific requirements (pure Go, cross-platform via Charm)
- Terminal with ANSI color support for visual testing of examples

**Production:**
- Library — consumed as a Go module dependency
- Published at `github.com/the-omics-os/crust`
- No deployment target (not a standalone application)

## Module Structure

The module `github.com/the-omics-os/crust` exports:
- Root package `crust` — shared message types (`SubmitMsg`, `CancelMsg`) in `result.go`
- `crust/qcdashboard` — QC metrics dashboard component
- `crust/thresholdslider` — interactive threshold adjustment component

Consumers import individual packages:
```go
import "github.com/the-omics-os/crust/qcdashboard"
import "github.com/the-omics-os/crust/thresholdslider"
```

## Commands

```bash
go mod tidy          # Resolve and clean dependencies
go test ./...        # Run all tests
go vet ./...         # Static analysis
go run ./examples/qc/         # Run QC dashboard example
go run ./examples/threshold/  # Run threshold slider example
```

---

*Stack analysis: 2026-03-09*
