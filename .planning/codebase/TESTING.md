# Testing Patterns

**Analysis Date:** 2026-03-09

## Test Framework

**Runner:**
- Go standard `testing` package
- No third-party test frameworks (no testify, no gomega)
- Config: none (uses `go test` defaults)

**Assertion Library:**
- Standard library only — manual `if` checks with `t.Fatalf` / `t.Errorf`

**Run Commands:**
```bash
go test ./...              # Run all tests
go test ./qcdashboard/    # Run tests for a specific component
go test -v ./...           # Verbose output
go test -cover ./...       # With coverage
go test -run TestSubmit ./thresholdslider/  # Run specific test
```

## Test File Organization

**Location:**
- Co-located with source code, same package (white-box testing)

**Naming:**
- `<package>_test.go` — e.g., `qcdashboard/qcdashboard_test.go`, `thresholdslider/thresholdslider_test.go`

**Structure:**
```
qcdashboard/
    model.go
    options.go
    metric.go
    qcdashboard_test.go      # Tests for all exported + internal behavior

thresholdslider/
    model.go
    options.go
    thresholdslider_test.go   # Tests for all exported + internal behavior
```

## Test Structure

**Suite Organization:**
- No `TestMain` setup/teardown
- Each test function is independent — no shared state between tests
- Tests are organized by function/behavior, named `Test<What>`:

```go
func TestNew(t *testing.T) {
    m := New(
        WithLabel("p-value cutoff"),
        WithRange(0, 1),
        WithStep(0.01),
        WithDefault(0.05),
    )
    if m.label != "p-value cutoff" {
        t.Fatalf("expected label 'p-value cutoff', got %q", m.label)
    }
    if !floatEq(m.Value(), 0.05) {
        t.Fatalf("expected value 0.05, got %f", m.Value())
    }
}
```

**Test naming conventions:**
- `TestNew` — constructor with options
- `TestNewDefaults` — constructor with zero options (default values)
- `TestNewClampsDefault` — constructor edge case (value clamping)
- `Test<Action>` — behavior tests: `TestAdjustRight`, `TestAdjustLeft`, `TestSubmit`, `TestCancel`
- `Test<Boundary>` — edge cases: `TestClampMin`, `TestClampMax`, `TestNoValueChangedMsg_AtBoundaries`
- `TestView*` — render output tests: `TestViewContainsMetricNames`, `TestViewContainsStatusText`, `TestViewVariousWidths`
- `Test<Type>_<Behavior>` — defensive copy tests: `TestMetrics_ReturnsCopy`, `TestSetMetrics_DefensiveCopy`
- `TestInitNoop` — verify Init returns nil
- `TestUpdateNoop` — verify non-interactive Update is no-op

**Assertion pattern:**
```go
// Always use t.Fatalf or t.Errorf with descriptive messages
if got != expected {
    t.Fatalf("expected %v, got %v", expected, got)
}
```

## Test Helpers

**Float comparison:**
```go
func floatEq(a, b float64) bool {
    return math.Abs(a-b) < 1e-9
}
```
Defined at file scope in `thresholdslider/thresholdslider_test.go`. Use this pattern for any float comparisons.

**Sample data factory:**
```go
func sampleMetrics() []Metric {
    return []Metric{
        {Name: "Reads", Value: 82, Min: 0, Max: 100, Unit: "%", Status: "pass"},
        {Name: "Genes", Value: 65, Min: 0, Max: 100, Unit: "%", Status: "warn"},
        {Name: "Mito %", Value: 3.2, Min: 0, Max: 20, Unit: "%", Status: "pass"},
    }
}
```
Defined at file scope in `qcdashboard/qcdashboard_test.go`. Each test file defines its own factory functions as needed.

## Mocking

**Framework:** None. No mocking framework is used.

**Approach:** Components are tested directly by constructing a `Model`, calling `Update()` with specific `tea.Msg` values, and inspecting the result. BubbleTea messages are constructed inline:

```go
// Simulate key press
updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})

// Simulate shift+key
updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift})

// Simulate non-key message
_, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
```

**Testing tea.Cmd results:**
```go
// Execute the cmd to get the message, then type-assert
_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
if cmd == nil {
    t.Fatal("expected SubmitMsg cmd on enter")
}
msg := cmd()
submit, ok := msg.(crust.SubmitMsg)
if !ok {
    t.Fatalf("expected crust.SubmitMsg, got %T", msg)
}
```

**What to mock:** Nothing. Crust components are pure — no I/O, no network, no filesystem. Test them directly.

## Fixtures and Factories

**Test Data:**
- Each test file defines package-level helper functions for sample data
- Use domain-realistic values (scRNA-seq metrics, p-value thresholds)
- Inline construction for one-off test cases

**Location:**
- In the `*_test.go` file, not in separate fixture files

## Coverage

**Requirements:** None enforced. No coverage thresholds configured.

**Current state:**
- `qcdashboard` — 11 tests covering: constructor, defaults, view rendering, set/get, defensive copies, Init/Update no-op, theme override
- `thresholdslider` — 14 tests covering: constructor, clamping, all key bindings (left/right/shift), submit/cancel messages, view rendering, boundary behavior, scientific notation parsing, Init no-op, non-key message handling
- Root `crust` package — no tests (only contains two struct definitions)
- `examples/` — no tests (runnable demos, not testable units)

**View Coverage:**
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Test Types

**Unit Tests:**
- All tests are unit tests
- Test one behavior per function
- No subtests (`t.Run`) used — each test case is a separate `Test*` function

**Table-driven tests:**
- Used for parameterized cases (see `TestDecimalPlaces_ScientificNotation`):
```go
func TestDecimalPlaces_ScientificNotation(t *testing.T) {
    tests := []struct {
        step float64
        want int
    }{
        {0.01, 2},
        {0.001, 3},
        {0.000001, 6},
    }
    for _, tt := range tests {
        got := decimalPlaces(tt.step)
        if got != tt.want {
            t.Errorf("decimalPlaces(%g) = %d, want %d", tt.step, got, tt.want)
        }
    }
}
```

**Integration Tests:** Not applicable. Crust components are standalone models with no external dependencies.

**E2E Tests:** Not used. Examples in `examples/` serve as manual integration demos.

## Common Patterns

**Testing constructors:**
```go
func TestNew(t *testing.T) {
    m := New(WithLabel("test"), WithRange(0, 1))
    // Assert fields via getters or direct field access (same package)
}

func TestNewDefaults(t *testing.T) {
    m := New()
    // Assert default values
}
```

**Testing Update (key handling):**
```go
func TestAdjustRight(t *testing.T) {
    m := New(WithRange(0, 1), WithStep(0.01), WithDefault(0.50))
    updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})
    got := updated.(Model)  // type-assert back to concrete Model
    if !floatEq(got.Value(), 0.51) {
        t.Fatalf("expected 0.51, got %f", got.Value())
    }
}
```

**Testing View output (string contains):**
```go
func TestViewContainsMetricNames(t *testing.T) {
    m := New(WithMetrics(sampleMetrics()))
    v := m.Render()  // Use Render() not View() for string checks
    for _, name := range []string{"Reads", "Genes", "Mito %"} {
        if !strings.Contains(v, name) {
            t.Fatalf("View should contain metric name %q", name)
        }
    }
}
```

**Testing width resilience:**
```go
func TestViewVariousWidths(t *testing.T) {
    m := New(WithMetrics(sampleMetrics()))
    for _, w := range []int{20, 40, 60, 80, 120} {
        m.SetWidth(w)
        v := m.Render()
        if v == "" {
            t.Fatalf("empty view at width %d", w)
        }
    }
}
```

**Testing defensive copies:**
```go
func TestMetrics_ReturnsCopy(t *testing.T) {
    m := New(WithMetrics(sampleMetrics()))
    got := m.Metrics()
    got[0].Name = "MUTATED"
    internal := m.Metrics()
    if internal[0].Name == "MUTATED" {
        t.Fatal("Metrics() returned a reference, not a copy")
    }
}
```

## What to Test for New Components

When adding a new Crust component, write tests for:

1. **Constructor** — `TestNew` with options, `TestNewDefaults` for zero-arg
2. **Init** — `TestInitNoop` (should return nil cmd)
3. **Update** — one test per key binding, plus non-key message ignored
4. **View/Render** — output contains expected text, renders at various widths without panic
5. **Getters** — return correct values, return defensive copies for slices
6. **Setters** — mutate state correctly, make defensive copies
7. **Edge cases** — boundary values, empty input, extreme widths
8. **Completion messages** — Submit/Cancel msgs have correct `Component` name and data (interactive components only)

---

*Testing analysis: 2026-03-09*
