package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/web"
	"github.com/amuxasi/amuxasi/workspace"
)

func main() {
	// Init logging
	dataDir, err := os.UserCacheDir()
	if err != nil {
		dataDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	if err := log.Init(filepath.Join(dataDir, "amuxasi", "logs")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: logger: %v\n", err)
	}
	defer log.Close()

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			cmdInit()
			return
		case "web":
			cmdWeb()
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
		case "version", "--version", "-v":
			fmt.Println("amuxasi v0.2.0")
			return
		}
	}

	// Default: launch TUI
	launchTUI()
}

func launchTUI() {
	// Try to exec the TUI binary
	tuiBin := "amuxasi-tui"
	if path, err := exec.LookPath(tuiBin); err == nil {
		cmd := exec.Command(path, os.Args[1:]...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Check if we're in the source directory
	localTUI := filepath.Join("cmd", "amuxasi-tui", "main.go")
	if _, err := os.Stat(localTUI); err == nil {
		fmt.Println("TUI binary not found. Build it with:")
		fmt.Println("  go build -o $GOPATH/bin/amuxasi-tui ./cmd/amuxasi-tui/")
		os.Exit(0)
	}

	// No TUI available, show help
	printHelp()
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

func cmdWeb() {
	port := 7000
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--port", "-p":
			if i+1 < len(os.Args) {
				if p, err := strconv.Atoi(os.Args[i+1]); err == nil {
					port = p
				}
				i++
			}
		}
	}
	repoRoot, _ := workspace.FindRepoRoot()
	if repoRoot == "" {
		repoRoot, _ = os.Getwd()
	}
	log.Info("starting web server on port %d, workspace: %s", port, repoRoot)
	fmt.Printf("🌐 Amuxasi Web UI → http://localhost:%d\n", port)
	fmt.Fprintf(os.Stderr, "   Workspace: %s\n", repoRoot)
	fmt.Fprintf(os.Stderr, "   Presiona Ctrl+C para detener\n")
	if err := web.StartServer(port, repoRoot); err != nil {
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
  amuxasi                Open TUI dashboard
  amuxasi web            Start Web UI (http://localhost:7000)
  amuxasi init           Create amuxasi.toml in current directory
  amuxasi add-worktree <path> [branch]  Create git worktree + config
  amuxasi archive        Archive workspace
  amuxasi help           Show this help
  amuxasi version        Show version

First time:
  1. cd your-project
  2. amuxasi init
  3. go install ./cmd/amuxasi-tui/  (if TUI not installed)
  4. amuxasi
  5. Press 'l' to launch an agent

Requires: tmux >= 3.3, git >= 2.5
Web UI:   http://localhost:7000 (run: amuxasi web)
Docker:   docker compose up (includes web UI)

Docs: https://github.com/kraken545/amuxasi`)
}
