// Example: standalone QC dashboard rendered in the terminal.
package main

import (
	"fmt"

	"github.com/the-omics-os/crust/qcdashboard"
)

func main() {
	metrics := []qcdashboard.Metric{
		{Name: "Reads", Value: 82, Min: 0, Max: 100, Unit: "%", Status: "pass"},
		{Name: "Genes", Value: 65, Min: 0, Max: 100, Unit: "%", Status: "warn"},
		{Name: "Mito %", Value: 3.2, Min: 0, Max: 20, Unit: "%", Status: "pass"},
		{Name: "Doublets", Value: 8.5, Min: 0, Max: 15, Unit: "%", Status: "warn"},
		{Name: "Ribosomal", Value: 22, Min: 0, Max: 50, Unit: "%", Status: "fail"},
	}

	model := qcdashboard.New(
		qcdashboard.WithTitle("scRNA-seq QC Summary"),
		qcdashboard.WithMetrics(metrics),
		qcdashboard.WithWidth(70),
	)

	// QCDashboard is non-interactive, so just print the rendered view.
	fmt.Println(model.Render())
	fmt.Println("\n(QCDashboard is non-interactive — this example just prints the view)")
}
