package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/web"
	"github.com/amuxasi/amuxasi/workspace"
)

func main() {
	port := flag.Int("port", 7000, "HTTP server port")
	workspacePath := flag.String("workspace", "", "Path to workspace (default: current dir)")
	flag.Parse()

	// Determine workspace path
	wsPath := *workspacePath
	if wsPath == "" {
		var err error
		wsPath, err = os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting working directory: %v\n", err)
			os.Exit(1)
		}
	}

	// Try to find git repo root
	if repoRoot, err := workspace.FindRepoRoot(); err == nil {
		wsPath = repoRoot
	}

	// Init logger
	dataDir, err := os.UserCacheDir()
	if err != nil {
		dataDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	logDir := filepath.Join(dataDir, "amuxasi", "logs")
	if err := log.Init(logDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not init logger: %v\n", err)
	}
	defer log.Close()

	log.Info("amuxasi-web starting on port %d, workspace: %s", *port, wsPath)

	// Create and start server
	srv := web.NewServer(*port, wsPath)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		log.Info("server stopping")
		os.Exit(0)
	}()

	if err := srv.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}
