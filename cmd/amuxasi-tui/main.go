package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/amuxasi/amuxasi/agent"
	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/trust"
	"github.com/amuxasi/amuxasi/tui"
	"github.com/amuxasi/amuxasi/workspace"
)

func main() {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = filepath.Join(os.Getenv("HOME"), ".config")
	}
	amuxasiConfigDir := filepath.Join(configDir, "amuxasi")

	dataDir, err := os.UserCacheDir()
	if err != nil {
		dataDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	logDir := filepath.Join(dataDir, "amuxasi", "logs")

	if err := log.Init(logDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: logger: %v\n", err)
	}
	defer log.Close()

	// Find workspace
	repoRoot, err := workspace.FindRepoRoot()
	if err != nil {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		repoRoot = dir
	}

	ws, err := workspace.Open(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if !agent.TmuxExists() {
		fmt.Fprintf(os.Stderr, "Warning: tmux not found. Agents require tmux.\n")
	}

	trustStore, err := trust.LoadStore(amuxasiConfigDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: trust store: %v\n", err)
		trustStore = &trust.Store{}
	}

	m := tui.NewModel(ws, trustStore)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
