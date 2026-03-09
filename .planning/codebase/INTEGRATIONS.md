# External Integrations

**Analysis Date:** 2026-03-09

## APIs & External Services

**None.** Crust is a pure UI component library with zero external service dependencies. It makes no network calls, has no API clients, and requires no service accounts.

## Data Storage

**Databases:**
- None — Crust is a stateless rendering library. All data is passed in via Go constructors.

**File Storage:**
- None — no file I/O. Components receive data programmatically.

**Caching:**
- None

## Authentication & Identity

**Auth Provider:**
- Not applicable — library has no authentication concerns

## Monitoring & Observability

**Error Tracking:**
- None

**Logs:**
- None — no logging framework. Components are pure BubbleTea models.

## CI/CD & Deployment

**Hosting:**
- GitHub at `github.com/the-omics-os/crust`
- Distributed as a Go module (consumers add via `go get`)

**CI Pipeline:**
- Not detected — no `.github/workflows/`, no CI configuration files present

**Releases:**
- No release tooling configured (no goreleaser, no release scripts)
- Go modules use git tags for versioning (e.g., `v0.1.0`)

## Environment Configuration

**Required env vars:**
- None

**Secrets location:**
- Not applicable

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Dependency Chain

Crust's only external integration is the **Charm ecosystem** (all compile-time, no runtime services):

| Dependency | Role | Import Path |
|------------|------|-------------|
| BubbleTea v2 | TUI framework (Elm architecture) | `charm.land/bubbletea/v2` |
| Lipgloss v2 | Terminal styling | `charm.land/lipgloss/v2` |
| Bubbles v2 | Component primitives (indirect, available) | `charm.land/bubbles/v2` |

All Charm dependencies are vendored via `go.sum` with integrity checksums. No runtime service connections.

## Consumer Integration Points

Crust is designed to be consumed by other Go applications. The primary known consumer is **Lobster TUI** (`lobster-tui/internal/biocomp/`), which wraps Crust components into a protocol-driven adapter layer. The integration boundary is:

- **Crust exports:** `tea.Model` implementations with typed constructors
- **Consumer provides:** JSON deserialization, protocol lifecycle, error boundaries
- **Message contract:** Components signal completion via `crust.SubmitMsg` and `crust.CancelMsg` (defined in `result.go`)

---

*Integration audit: 2026-03-09*
