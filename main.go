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
		fmt.Fprintf(os.Stderr, "Warning: could not init logger: %v\n", err)
	}
	defer log.Close()

	log.Info("amuxasi starting")

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cmdInit()
			return
		case "open", "":
			cmdOpen(amuxasiConfigDir)
			return
		case "add-worktree":
			cmdAddWorktree()
			return
		case "archive":
			cmdArchive()
			return
		case "help", "--help", "-h":
			printHelp()
			return
		case "version":
			fmt.Println("amuxasi v0.2.0")
			return
		}
	}

	cmdOpen(amuxasiConfigDir)
}

func cmdInit() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if err := workspace.Init(dir); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdOpen(configDir string) {
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
		fmt.Fprintf(os.Stderr, "Warning: tmux not found in PATH. Agents require tmux.\n")
		fmt.Fprintf(os.Stderr, "  Install: brew install tmux  /  apt install tmux  /  pacman -S tmux\n")
	}

	trustStore, err := trust.LoadStore(configDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load trust store: %v\n", err)
		trustStore = &trust.Store{}
	}

	m := tui.NewModel(ws, trustStore)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdAddWorktree() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: amuxasi add-worktree <path> [branch]\n")
		os.Exit(1)
	}

	repoRoot, err := workspace.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ws, err := workspace.Open(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	path := os.Args[2]
	branch := ""
	if len(os.Args) > 3 {
		branch = os.Args[3]
	}

	if err := ws.AddWorktree(path, branch); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdArchive() {
	repoRoot, err := workspace.FindRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	ws, err := workspace.Open(repoRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := ws.Archive(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println(`Amuxasi — Multi-Agent Coding Dashboard

Usage:
  amuxasi                Open dashboard in current repo
  amuxasi open           Open dashboard (same as above)
  amuxasi init           Create amuxasi.toml in current repo
  amuxasi add-worktree <path> [branch]  Create git worktree + config
  amuxasi archive        Archive workspace
  amuxasi help           Show this help
  amuxasi version        Show version

First time:
  1. cd your-project
  2. amuxasi init
  3. amuxasi
  4. Press 'l' to launch an agent
  5. Press '?' for keyboard shortcuts

Configuration:
  See amuxasi.toml in your repo root.
  Supported agents: claude, opencode, codex, gemini, amp, droid
  Custom agents: add any command to amuxasi.toml

Requires: tmux >= 3.3, git >= 2.5

Docs: https://github.com/amuxasi/amuxasi`)
}
