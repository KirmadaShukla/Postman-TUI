package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"my-new-go/internal/store"
	"my-new-go/internal/ui"
)

func main() {
	st, err := store.New("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "store: %v\n", err)
		os.Exit(1)
	}

	ws, err := st.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load workspace: %v\n", err)
		os.Exit(1)
	}

	// No mouse capture so the terminal's native text selection / copy works.
	// Scroll the response pane with tab → Response, then ↑/↓ / pgup/pgdown.
	p := tea.NewProgram(
		ui.New(st, ws),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "app: %v\n", err)
		os.Exit(1)
	}
}
