package main

import (
	"fmt"
	"os"

	"github.com/abdenasser/neohtop-cli/model"
	"github.com/abdenasser/neohtop-cli/monitor"

	tea "charm.land/bubbletea/v2"
)

func main() {
	// Initialize the native Go monitor (reads OS interfaces directly — no FFI)
	mon := monitor.New()
	defer mon.Destroy()

	// Create and run the Bubble Tea program
	app := model.NewApp(mon)
	p := tea.NewProgram(app)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
