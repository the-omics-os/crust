// Package platemap provides an interactive assay plate viewer for Bubble Tea.
package platemap

import (
	"fmt"
	"strings"
)

// PlateFormat describes the dimensions of a supported assay plate.
type PlateFormat int

const (
	Plate96   PlateFormat = 96
	Plate384  PlateFormat = 384
	Plate1536 PlateFormat = 1536
)

const (
	ControlPositive = "positive"
	ControlNegative = "negative"
	ControlSample   = "sample"
	ControlEmpty    = "empty"
)

// Well is a single plate position.
type Well struct {
	Row        int
	Col        int
	Signal     float64
	Normalized float64
	ZScore     float64
	Control    string
	SampleID   string
	Reagent    string
	Hit        bool
	Missing    bool
}

// PlateData is the full payload rendered by PlateMap.
type PlateData struct {
	Format   PlateFormat
	Wells    []Well
	Title    string
	Metadata map[string]string
}

// Copy returns a deep copy of the plate payload.
func (p PlateData) Copy() PlateData {
	out := PlateData{
		Format: p.Format,
		Title:  p.Title,
		Wells:  append([]Well(nil), p.Wells...),
	}
	if p.Metadata != nil {
		out.Metadata = make(map[string]string, len(p.Metadata))
		for key, value := range p.Metadata {
			out.Metadata[key] = value
		}
	}
	return out
}

// Rows returns the number of rows in the format.
func (f PlateFormat) Rows() int {
	switch f {
	case Plate96:
		return 8
	case Plate384:
		return 16
	case Plate1536:
		return 32
	default:
		return Plate96.Rows()
	}
}

// Cols returns the number of columns in the format.
func (f PlateFormat) Cols() int {
	switch f {
	case Plate96:
		return 12
	case Plate384:
		return 24
	case Plate1536:
		return 48
	default:
		return Plate96.Cols()
	}
}

// WellCount returns the total well count for the format.
func (f PlateFormat) WellCount() int {
	return f.Rows() * f.Cols()
}

// Valid reports whether the format is one of the supported plate sizes.
func (f PlateFormat) Valid() bool {
	switch f {
	case Plate96, Plate384, Plate1536:
		return true
	default:
		return false
	}
}

// String returns a human-readable label.
func (f PlateFormat) String() string {
	if f.Valid() {
		return fmt.Sprintf("%d-Well Plate", int(f))
	}
	return Plate96.String()
}

func normalizePlateData(plate PlateData) PlateData {
	plate = plate.Copy()
	if !plate.Format.Valid() {
		plate.Format = inferPlateFormat(len(plate.Wells))
	}
	return plate
}

func inferPlateFormat(wellCount int) PlateFormat {
	switch {
	case wellCount > Plate384.WellCount():
		return Plate1536
	case wellCount > Plate96.WellCount():
		return Plate384
	default:
		return Plate96
	}
}

func requiredFormatForCoordinate(row, col int) (PlateFormat, bool) {
	switch {
	case row < 0 || col < 0:
		return Plate96, false
	case row < Plate96.Rows() && col < Plate96.Cols():
		return Plate96, true
	case row < Plate384.Rows() && col < Plate384.Cols():
		return Plate384, true
	case row < Plate1536.Rows() && col < Plate1536.Cols():
		return Plate1536, true
	default:
		return Plate1536, false
	}
}

func normalizeControl(control string) string {
	switch strings.ToLower(strings.TrimSpace(control)) {
	case "", ControlSample:
		return ControlSample
	case ControlPositive:
		return ControlPositive
	case ControlNegative:
		return ControlNegative
	case ControlEmpty:
		return ControlEmpty
	default:
		return ControlSample
	}
}

func rowLabel(row int) string {
	if row < 0 {
		return ""
	}

	label := ""
	for row >= 0 {
		label = string(rune('A'+(row%26))) + label
		row = row/26 - 1
	}
	return label
}

func coordinate(row, col int) string {
	return fmt.Sprintf("%s%d", rowLabel(row), col+1)
}
