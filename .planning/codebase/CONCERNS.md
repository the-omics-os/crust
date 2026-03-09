# Codebase Concerns

**Analysis Date:** 2026-03-09

## Tech Debt

**3 of 5 v0.1 components not yet implemented:**
- Issue: Only `qcdashboard` and `thresholdslider` exist. The CLAUDE.md spec calls for 5 components in v0.1: SequenceViewer (flagship), OntologyBrowser, and PeriodicTable are missing.
- Files: Project root — missing `sequenceviewer/`, `ontologybrowser/`, `periodictable/` directories
- Impact: Cannot release v0.1 or demonstrate the library's scientific breadth. SequenceViewer is labeled the "flagship" component.
- Fix approach: Build components in priority order: SequenceViewer (highest value, most domain-specific), PeriodicTable (self-contained data, good second), OntologyBrowser (most complex — tree navigation, lazy loading).

**No shared theme package:**
- Issue: `qcdashboard` and `thresholdslider` each define independent `Theme` structs with no shared base. The CLAUDE.md spec mentions a potential shared `crust/theme` package. Current themes have different fields and no overlap.
- Files: `qcdashboard/options.go` (Theme with Pass/Warn/Fail/Border/Text/TextMuted), `thresholdslider/options.go` (Theme with Value/Filled/Empty/Cursor/Count/Title)
- Impact: Consumers who want consistent theming across components must configure each independently. As more components are added, theme divergence will grow.
- Fix approach: Extract shared `theme/` package with base colors (Primary, Secondary, Pass, Warn, Fail, Border, Text, TextMuted) as defined in CLAUDE.md. Each component embeds or references the shared theme plus component-specific extensions.

**Metric.Status is a raw string, not a typed enum:**
- Issue: `qcdashboard.Metric.Status` is `string` with magic values "pass", "warn", "fail". No validation, no compile-time safety.
- Files: `qcdashboard/metric.go` (line 11), `qcdashboard/model.go` (lines 167-178 — `statusColor` uses string matching)
- Impact: Typos like "Pass" or "passed" silently fall through to the default color. Easy to misuse.
- Fix approach: Define `type Status string` with constants (`StatusPass`, `StatusWarn`, `StatusFail`). Add a `Valid()` method. Update `statusColor` to use the typed constants.

**SubmitMsg.Data uses `map[string]any` — loses type safety:**
- Issue: `crust.SubmitMsg` carries `Data map[string]any`, requiring type assertions at the consumer. The thresholdslider puts `{"value": float64}` but this contract is only documented by convention.
- Files: `result.go` (lines 10-13), `thresholdslider/model.go` (lines 103-108)
- Impact: Consumer code must do `msg.Data["value"].(float64)` which can panic if the key is missing or the type is wrong. As more components are added, each will have different Data shapes with no compile-time guarantees.
- Fix approach: Consider making `SubmitMsg` and `CancelMsg` generic or defining component-specific result types (e.g., `thresholdslider.SubmitMsg{Value: float64}`). Alternatively, add a typed helper: `thresholdslider.ExtractValue(msg crust.SubmitMsg) (float64, error)`.

**Go version set to 1.25.0 (unreleased):**
- Issue: `go.mod` specifies `go 1.25.0`. Go 1.25 is not yet released (current stable is 1.24.x as of March 2026). This may cause build issues for contributors.
- Files: `go.mod` (line 3)
- Impact: Users with stable Go toolchains cannot build the module. `go mod tidy` may behave unexpectedly.
- Fix approach: Downgrade to `go 1.24` in `go.mod` unless there is a specific 1.25 feature dependency. The CLAUDE.md spec says "Go 1.24+".

**No README.md:**
- Issue: The repository has no README. For an open-source library, this is the primary discovery and onboarding surface.
- Files: Missing `README.md` at project root
- Impact: GitHub renders nothing. Developers cannot evaluate the library without reading source code.
- Fix approach: Create README.md with: module path, install command, component inventory table, minimal usage example, screenshot/GIF of each component, link to examples/.

## Known Bugs

**QCDashboard border width mismatch with long titles:**
- Symptoms: The top border calculation uses `len(titleText)` which counts bytes, not display width. Titles with multi-byte Unicode characters will produce misaligned borders.
- Files: `qcdashboard/model.go` (lines 73-83)
- Trigger: Set a title containing non-ASCII characters (e.g., emoji, CJK).
- Workaround: Use ASCII-only titles. Fix by using `lipgloss.Width()` instead of `len()`.

**QCDashboard metric name truncation uses byte slicing:**
- Symptoms: `name[:12]` on line 117 of `qcdashboard/model.go` slices by bytes, not runes. Multi-byte characters at position 12 could be split, producing invalid UTF-8.
- Files: `qcdashboard/model.go` (line 117)
- Trigger: Metric name with multi-byte characters where the 12th byte falls mid-rune.
- Workaround: Use ASCII metric names. Fix with `[]rune(name)[:12]` or `lipgloss.Width`-based truncation.

## Security Considerations

**No significant security concerns:**
- Risk: Crust is a pure TUI rendering library with no network access, file I/O, or user data persistence. Attack surface is minimal.
- Current mitigation: No external inputs beyond constructor arguments from the embedding application.
- Recommendations: If future components accept external data (e.g., OntologyBrowser loading from URLs), add input validation and size limits.

## Performance Bottlenecks

**Lipgloss style creation on every render:**
- Problem: Both components create new `lipgloss.NewStyle()` objects on every `render()` call. QCDashboard creates styles per metric line as well.
- Files: `qcdashboard/model.go` (lines 63, 119, 125, 155-156, 161), `thresholdslider/model.go` (lines 136-141)
- Cause: Styles are not cached on the Model struct. Each View/Render call allocates new style objects.
- Improvement path: Pre-compute styles in the constructor and store on the Model. Update them only when the theme changes. This matters for streaming components (QCDashboard is described as supporting streaming metric updates).

**QCDashboard Update is a complete no-op:**
- Problem: `Update()` returns `m, nil` for all messages, including `tea.WindowSizeMsg`. The component cannot adapt to terminal resizes.
- Files: `qcdashboard/model.go` (lines 43-45)
- Cause: Designed as non-interactive, but window resize is not "interaction" — it is layout.
- Improvement path: Handle `tea.WindowSizeMsg` in Update to auto-adjust width. This is standard BubbleTea practice.

## Fragile Areas

**ThresholdSlider floating-point precision:**
- Files: `thresholdslider/model.go` (lines 224-262 — `clamp`, `decimalPlaces`, `roundTo`)
- Why fragile: Floating-point arithmetic with repeated additions (e.g., `m.value + m.step`) accumulates error over many adjustments. The `roundTo` function mitigates this but only at display time — internal state drifts.
- Safe modification: Always round after arithmetic, not just on output. Test with edge-case steps like 0.3 and many sequential adjustments.
- Test coverage: `decimalPlaces` has good edge case tests. Missing: test for accumulated floating-point drift after many sequential adjustments.

**Border rendering arithmetic in QCDashboard:**
- Files: `qcdashboard/model.go` (lines 57-95, 114-164)
- Why fragile: Manual character-counting for border alignment with multiple variables (`innerW`, `overhead`, `barWidth`, `fillCount`). Any change to the metric line format requires recalculating multiple offsets.
- Safe modification: Change one variable at a time and test at multiple widths. The `TestViewVariousWidths` test exists but only checks for non-empty output — does not verify alignment.
- Test coverage: No test verifies that border top/bottom widths match. No test checks that metric lines fit within the border width.

## Scaling Limits

**No concerns at current scale:**
- The library renders TUI components with small data sets (a few metrics, one sequence at a time). No scaling bottlenecks expected until components like OntologyBrowser handle thousands of tree nodes or SequenceViewer handles megabase sequences.
- Future concern: SequenceViewer will need viewport-based lazy rendering for long sequences. Plan to use `bubbles/viewport` as specified in CLAUDE.md.

## Dependencies at Risk

**Charm v2 is very new:**
- Risk: `charm.land/bubbletea/v2` (v2.0.1), `charm.land/lipgloss/v2` (v2.0.0), `charm.land/bubbles/v2` (v2.0.0) are at initial release versions. APIs may change in minor releases.
- Impact: Breaking changes in Charm v2 would require updates across all components.
- Migration plan: Pin exact versions in `go.mod`. Monitor Charm changelog. The library is small enough that updates are manageable.

## Missing Critical Features

**No `bubbles/v2` direct dependency:**
- Problem: `bubbles/v2` appears only as an indirect dependency. None of the current components use Bubbles primitives (viewport, textinput, key bindings). The CLAUDE.md spec says "Use `bubbles/viewport` for scrolling, `bubbles/textinput` for text fields, `bubbles/key` for bindings."
- Blocks: SequenceViewer needs viewport for scrolling. OntologyBrowser needs key bindings for tree navigation. These components cannot be built following spec without promoting `bubbles/v2` to a direct dependency.

**No godoc package documentation:**
- Problem: Root package `crust` has a one-line package comment. Component packages have brief doc comments but no usage examples in doc format.
- Blocks: `go doc` and pkg.go.dev rendering will be minimal.

**No LICENSE file:**
- Problem: CLAUDE.md mentions "TBD (Apache 2.0 or MIT)" but no LICENSE file exists.
- Blocks: Open-source consumers cannot legally use the library without a license.

## Test Coverage Gaps

**Root package `crust` has no tests:**
- What's not tested: `SubmitMsg` and `CancelMsg` types in `result.go` have no direct tests. They are exercised indirectly through `thresholdslider` tests.
- Files: `result.go`
- Risk: Low — these are simple structs. But if they gain methods (e.g., validation), tests will be needed.
- Priority: Low

**No visual regression / snapshot tests:**
- What's not tested: Rendered output is only checked for non-emptiness and substring presence. No test verifies exact visual output or that borders align correctly.
- Files: `qcdashboard/qcdashboard_test.go`, `thresholdslider/thresholdslider_test.go`
- Risk: Rendering regressions (misaligned borders, wrong colors, broken layout at edge widths) will go undetected. The QCDashboard border arithmetic is particularly susceptible.
- Priority: Medium — add golden file tests or at minimum assert line counts and character widths.

**No test for zero or negative ranges:**
- What's not tested: `thresholdslider` with `min == max` (zero range) or `min > max` (inverted range). `qcdashboard` with `Min == Max` for a metric.
- Files: `thresholdslider/model.go` (line 159 — division by zero guarded), `qcdashboard/model.go` (line 149 — division by zero guarded)
- Risk: The division-by-zero is guarded but behavior with inverted ranges (min > max) is undefined. `clamp` would clamp to `hi` which is less than `lo`.
- Priority: Medium — add tests and consider returning an error or swapping min/max in the constructor.

**No test for empty metrics list:**
- What's not tested: `qcdashboard.New()` with no metrics produces a box with only borders. Not tested.
- Files: `qcdashboard/model.go`
- Risk: Low — likely works, but untested edge case.
- Priority: Low

---

*Concerns audit: 2026-03-09*
