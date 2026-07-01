package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/amuxasi/amuxasi/log"
	"github.com/amuxasi/amuxasi/web"
	"github.com/amuxasi/amuxasi/workspace"
)

func main() {
	// Environment defaults (Docker-friendly)
	defaultPort := 7000
	if p := os.Getenv("AMUXASI_PORT"); p != "" {
		if v, err := strconv.Atoi(p); err == nil {
			defaultPort = v
		}
	}
	defaultWorkspace := os.Getenv("AMUXASI_WORKSPACE")
	if defaultWorkspace == "" {
		defaultWorkspace, _ = os.Getwd()
	}

	port := flag.Int("port", defaultPort, "HTTP server port")
	workspacePath := flag.String("workspace", defaultWorkspace, "Path to workspace")
	flag.Parse()

	// Resolve workspace: use --workspace flag > AMUXASI_WORKSPACE env > git root > cwd
	wsPath := *workspacePath
	if wsPath == "" {
		wsPath, _ = os.Getwd()
	}

	// Init logger (pre-create dirs for the amuxasi user in Docker)
	dataDir, err := os.UserCacheDir()
	if err != nil {
		dataDir = filepath.Join(os.Getenv("HOME"), ".cache")
	}
	logDir := filepath.Join(dataDir, "amuxasi", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create log dir: %v\n", err)
	}
	if err := log.Init(logDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not init logger: %v\n", err)
	}
	defer log.Close()

	// Try git repo root (only if it doesn't override the user's choice)
	if *workspacePath == "" {
		if repoRoot, err := workspace.FindRepoRoot(); err == nil {
			wsPath = repoRoot
		}
	}

	log.Info("amuxasi-web starting on port %d, workspace: %s", *port, wsPath)

	srv := web.NewServer(*port, wsPath)

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
