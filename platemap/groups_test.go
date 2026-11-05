package platemap

import (
	"image/color"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/the-omics-os/crust"
)

var ansiCodes = regexp.MustCompile(`\x1b\[[0-9;:]*m`)

func plain(s string) string { return ansiCodes.ReplaceAllString(s, "") }

func groupedPlate() PlateData {
	return PlateData{
		Format: Plate96,
		Title:  "Groups",
		Groups: []string{"vehicle", "treated"},
		Wells: []Well{
			{Row: 0, Col: 0, SampleID: "S01", Group: "vehicle"},
			{Row: 0, Col: 1, SampleID: "S02", Group: "treated"},
			{Row: 0, Col: 2, SampleID: "S03", Group: "treated", Noted: true},
			{Row: 0, Col: 3, SampleID: "S04"},
			{Row: 0, Col: 4, SampleID: "S05", Group: "vehicle", Flagged: true},
			{Row: 0, Col: 5, SampleID: "QC1", Excluded: true},
			{Row: 0, Col: 6, Control: ControlPositive, SampleID: "POS"},
		},
	}
}

// rowCells returns the glyphs of one grid row, one per well (blank cells
// included), from a plain grid-only render.
func rowCells(t *testing.T, render string, row int) []string {
	t.Helper()
	lines := strings.Split(plain(render), "\n")
	line := []rune(lines[1+row]) // line 0 holds the column numbers
	var cells []string
	for start := 2; start+cellWidth <= len(line); start += cellWidth + 1 {
		cells = append(cells, strings.TrimSpace(string(line[start:start+cellWidth])))
	}
	return cells
}

func TestGroupsViewDrawsEachWellState(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithGridOnly(true), WithCursor(7, 11), WithWidth(80), WithHeight(9))
	got := rowCells(t, m.Render(), 0)
	want := []string{"●", "●", "◉", "○", "●!", "◌", "+", "·", "·", "·", "·", "·"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("row A = %q, want %q", got, want)
	}
}

func TestGroupsViewColoursFollowTheGroupOrder(t *testing.T) {
	theme := DefaultTheme()
	theme.Groups = []color.Color{lipgloss.Color("21"), lipgloss.Color("46")}
	m := New(WithPlate(groupedPlate()), WithTheme(theme), WithViewMode(ViewGroups))

	if c, ok := m.GroupColor("vehicle"); !ok || c != lipgloss.Color("21") {
		t.Fatalf("vehicle colour = %v %v", c, ok)
	}
	if c, ok := m.GroupColor("treated"); !ok || c != lipgloss.Color("46") {
		t.Fatalf("treated colour = %v %v", c, ok)
	}
	if c, ok := m.GroupColor("other"); ok || c != theme.Sample {
		t.Fatalf("a group not listed is drawn in Theme.Sample, got %v %v", c, ok)
	}

	plate := groupedPlate()
	plate.Groups = append(plate.Groups, "third")
	m.SetPlate(plate)
	if c, _ := m.GroupColor("third"); c != lipgloss.Color("21") {
		t.Fatalf("colours repeat past the end of Theme.Groups, got %v", c)
	}
}

func TestDefaultGroupColoursAreCrustsCategoryPaletteWithoutOrange(t *testing.T) {
	theme := DefaultTheme()
	if len(theme.Groups) < 9 {
		t.Fatalf("want a colour per number key (9), got %d", len(theme.Groups))
	}
	for i, c := range theme.Groups {
		if c == theme.Missing || c == theme.Empty {
			t.Fatalf("group colour %d is a status colour %v", i, c)
		}
	}
}

func TestGroupsViewRendersInColour(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithGridOnly(true))
	out := m.Render()
	for _, code := range []string{"38;5;39m", "38;5;77m", "38;5;214m"} {
		if !strings.Contains(out, code) {
			t.Fatalf("render lacks colour %q", code)
		}
	}
}

func TestGridOnlyDrawsAllRowsAndNoChrome(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithGridOnly(true), WithWidth(60), WithHeight(9))
	out := plain(m.Render())
	lines := strings.Split(out, "\n")
	if len(lines) != 9 {
		t.Fatalf("want 9 lines (column numbers + 8 rows), got %d:\n%s", len(lines), out)
	}
	if !strings.HasPrefix(lines[8], "H ") || !strings.Contains(lines[0], "12") {
		t.Fatalf("rows or columns cut:\n%s", out)
	}
	for _, chrome := range []string{"View:", "Window rows", "---", "[A1]"} {
		if strings.Contains(out, chrome) {
			t.Fatalf("grid-only render has %q:\n%s", chrome, out)
		}
	}
}

func TestFullGroupsRenderHasALegendAndFooter(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithWidth(100), WithHeight(20), WithCursor(0, 4))
	out := plain(m.Render())
	for _, want := range []string{"View: Groups", "● vehicle  ● treated  ○ unassigned  ◉ noted  ! flagged  ◌ excluded", "Group: vehicle", "Flagged"} {
		if !strings.Contains(out, want) {
			t.Fatalf("render lacks %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "H ") {
		t.Fatalf("the legend line pushed rows off:\n%s", out)
	}

	narrow := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithWidth(32), WithHeight(20))
	for _, line := range strings.Split(plain(narrow.Render()), "\n") {
		if strings.HasPrefix(line, "● vehicle") && !strings.HasSuffix(strings.TrimRight(line, " "), "…") {
			t.Fatalf("a legend wider than the plate must end in …, got %q", line)
		}
	}
}

func TestTabReachesTheGroupsViewLast(t *testing.T) {
	m := New(WithViewMode(ViewMissingness))
	updated, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := updated.(Model).Mode(); got != ViewGroups || got.String() != "Groups" {
		t.Fatalf("tab after Missingness = %v", got)
	}
	updated, _ = updated.(Model).Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := updated.(Model).Mode(); got != ViewRawSignal {
		t.Fatalf("tab after Groups = %v", got)
	}
}

func TestSubmitDataCarriesTheGroupFields(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithCursor(0, 4))
	_, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	well := cmd().(crust.SubmitMsg).Data["well"].(map[string]any)
	if well["group"] != "vehicle" || well["flagged"] != true || well["noted"] != false || well["excluded"] != false {
		t.Fatalf("well data = %v", well)
	}
}

func TestCopyKeepsGroupsApart(t *testing.T) {
	plate := groupedPlate()
	m := New(WithPlate(plate))
	plate.Groups[0] = "changed"
	if m.Plate().Groups[0] != "vehicle" {
		t.Fatal("the model shares its Groups with the caller")
	}
	got := m.Plate()
	got.Groups[0] = "changed"
	if m.Plate().Groups[0] != "vehicle" {
		t.Fatal("Plate() shares its Groups with the model")
	}
}

func TestGroupNumbersReplaceTheDots(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithViewMode(ViewGroups), WithGridOnly(true), WithGroupNumbers(true), WithCursor(7, 11))
	got := rowCells(t, m.Render(), 0)
	want := []string{"1", "2", "2", "○", "1!", "◌", "+", "·", "·", "·", "·", "·"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("row A = %q, want %q", got, want)
	}
	m.SetGroupNumbers(false)
	if m.GroupNumbers() || rowCells(t, m.Render(), 0)[0] != "●" {
		t.Fatal("SetGroupNumbers(false) should bring the dots back")
	}
}

func TestTheCursorsRowAndColumnHeadersStandOut(t *testing.T) {
	m := New(WithPlate(groupedPlate()), WithGridOnly(true), WithCursor(3, 4))
	lines := strings.Split(m.Render(), "\n")
	marked := func(s string) bool { return strings.Contains(s, "1;4;") || strings.Contains(s, "[1;4") }
	header := lines[0]
	i := strings.Index(plain(header), " 5 ")
	if i < 0 || !strings.Contains(header, "\x1b[1;4") {
		t.Fatalf("column 5 header not bold and underlined: %q", header)
	}
	if !marked(lines[4]) || marked(lines[3]) {
		t.Fatalf("only row D's letter should be bold and underlined:\n%q\n%q", lines[3], lines[4])
	}
}
